package main

import (
	"bytes"
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"strings"
	"time"

	"github.com/mpetavy/common"
)

var (
	filename    *string
	address     *string
	readTimeout *int
	stepTimeout *int
	useTls      *bool
)

func init() {
	filename = flag.String("f", "visualas.dmp", "filename for dumping received Visulas data")
	address = flag.String("c", "", "socket address to read from")
	readTimeout = flag.Int("rt", 3000, "readTimeout")
	stepTimeout = flag.Int("st", 500, "stepTimeout")
	useTls = flag.Bool("tls", false, "use tls")
}

const (
	forum_ready   = "A\rFORUM_READY\r\x04"
	visulas_ready = "A\rVISULAS_READY\r\x04"
	receive_ready = "A\rFORUM_RECEIVE_READY\r\x04"
	review_error  = "A\rREVIEW_ERROR\r\x04"
	review_ready  = "A\rREVIEW_READY\r\x04"
)

func init() {
	common.Init("1.0.0", "2019", "VISULAS query via Silex", "mpetavy", common.APACHE, false, nil, nil, run, 0)
}

func convert(txt string) string {
	return strings.ReplaceAll(txt, "\r", "\r\n")
}

func read(conn net.Conn, expect string) []byte {
	if *stepTimeout > 0 {
		time.Sleep(time.Duration(*stepTimeout) * time.Millisecond)
	}

	if *readTimeout > 0 {
		common.Error(conn.SetDeadline(common.DeadlineByMsec(*readTimeout)))
	}

	fmt.Printf("---------- read from Silex: %+q\n", expect)

	b1 := make([]byte, 1)
	buf := bytes.Buffer{}

	for {
		nread, err := conn.Read(b1)
		if err != nil {
			panic(err)
		}
		if nread > 0 {
			buf.Write(b1)

			if b1[0] == '\x04' {
				break
			}
		}
	}
	txt := string(buf.Bytes())

	fmt.Printf("%d bytes read\n", buf.Len())
	fmt.Printf("%s\n", convert(txt))

	return buf.Bytes()
}

func write(conn net.Conn, txt string) {
	if *stepTimeout > 0 {
		time.Sleep(time.Duration(*stepTimeout) * time.Millisecond)
	}

	fmt.Printf("---------- write to Silex: %+q\n", txt)

	var err error

	n, err := conn.Write([]byte(txt))
	if err != nil {
		panic(err)
	}
	fmt.Printf("%d bytes written\n", n)
	fmt.Printf("%s\n", convert(txt))
}

func run() error {
	fmt.Printf("dial\n")

	var conn net.Conn
	var err error

	if *useTls {
		config := &tls.Config{
			InsecureSkipVerify: true,
		}
		conn, err = tls.Dial("tcp", *address, config)
		if err != nil {
			panic(err)
		}
	} else {
		conn, err = net.Dial("tcp", *address)
		if err != nil {
			panic(err)
		}
	}

	defer func() {
		common.Error(conn.Close())
	}()

	fmt.Printf("%s connected successfully\n", *address)

	write(conn, forum_ready)
	read(conn, visulas_ready)
	write(conn, receive_ready)
	common.Error(ioutil.WriteFile(*filename, read(conn, "data"), common.DefaultFileMode))
	write(conn, review_ready)

	return nil
}

func main() {
	defer common.Done()

	common.Run([]string{"c"})
}
