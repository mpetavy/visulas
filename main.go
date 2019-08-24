package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/mpetavy/common"
	"io/ioutil"
	"net"
	"os"
	"strings"
)

var (
	address  *string
	filename *string
	udp      *bool
)

func init() {
	address = flag.String("c", "", "server:port to test")
	filename = flag.String("f", "visualas.dmp", "filename for dumping received Visulas data")
}

const (
	forum_ready   = "A\rFORUM_READY\r\x04"
	visulas_ready = "A\rVISULAS_READY\r\x04"
	receive_ready = "A\rFORUM_RECEIVE_READY\r\x04"
	review_error  = "A\rREVIEW_ERROR\r\x04"
	review_ready  = "A\rREVIEW_READY\r\x04"
)

func init() {
	common.Init("visulas", "1.0.0", "2019", "Can connect to server:port", "mpetavy", common.APACHE, "VISULAS query via Silex", false, nil, nil, run, 0)
}

func convert(txt string) string {
	return strings.ReplaceAll(txt, "\r", "\r\n")
}

func read(conn net.Conn) []byte {
	fmt.Printf("---------- read from Silex\n")

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
	fmt.Printf("---------- write to Silex\n")

	var err error

	n, err := conn.Write([]byte(txt))
	if err != nil {
		panic(err)
	}
	fmt.Printf("%d bytes written\n", n)
	fmt.Printf("%s\n", convert(txt))
}

func run() error {
	conn, err := net.Dial("tcp", *address)
	if err != nil {
		return err
	}

	defer conn.Close()

	fmt.Printf("%s connected successfully\n", *address)

	write(conn, forum_ready)
	read(conn)
	write(conn, receive_ready)
	ioutil.WriteFile(*filename, read(conn), os.ModePerm)
	write(conn, review_ready)

	return nil
}

func main() {
	defer common.Done()

	common.Run([]string{"c"})
}
