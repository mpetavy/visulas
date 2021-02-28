package main

import (
	"bytes"
	"crypto/tls"
	"flag"
	"fmt"
	"os"
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
	stepTimeout = flag.Int("st", 1000, "stepTimeout")
	useTls = flag.Bool("tls", false, "use tls")
}

const (
	forum_ready   = "A\rFORUM_READY\r\x04"
	visulas_ready = "A\rVISULAS_READY\r\x04"
	receive_ready = "A\rFORUM_RECEIVE_READY\r\x04"
	//review_error  = "A\rREVIEW_ERROR\r\x04"
	review_ready = "A\rREVIEW_READY\r\x04"
)

func init() {
	common.Init(false, "1.0.0", "", "2019", "VISULAS query via Silex", "mpetavy", fmt.Sprintf("https://github.com/mpetavy/%s", common.Title()), common.APACHE, nil, nil, run, 0)
}

func convert(txt string) string {
	return strings.ReplaceAll(txt, "\r", "\r\n")
}

func read(conn *common.NetworkConnection, expect string) []byte {
	if *stepTimeout > 0 {
		time.Sleep(common.MillisecondToDuration(*stepTimeout))
	}

	if *readTimeout > 0 {
		common.Error(conn.Socket.SetDeadline(common.DeadlineByMsec(*readTimeout)))
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

func write(conn *common.NetworkConnection, txt string) {
	if *stepTimeout > 0 {
		time.Sleep(common.MillisecondToDuration(*stepTimeout))
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

	var tlsConfig *tls.Config

	if *useTls {
		var err error

		tlsConfig, err = common.NewTlsConfigFromFlags()
		if common.Error(err) {
			return err
		}
	}

	client, err := common.NewNetworkClient(*address, tlsConfig)
	if common.Error(err) {
		return err
	}

	defer func() {
		common.Error(client.Stop())
	}()

	conn, err := client.Connect()
	if common.Error(err) {
		return err
	}

	defer func() {
		common.Error(conn.Close())
	}()

	fmt.Printf("%s connected successfully\n", *address)

	write(conn, forum_ready)
	read(conn, visulas_ready)
	write(conn, receive_ready)
	common.Error(os.WriteFile(*filename, read(conn, "data"), common.DefaultFileMode))
	write(conn, review_ready)

	return nil
}

func main() {
	defer common.Done()

	common.Run([]string{"c"})
}
