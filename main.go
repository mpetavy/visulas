package main

import (
	"bytes"
	"flag"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/mpetavy/common"
)

var (
	address *string
	udp     *bool
)

func init() {
	address = flag.String("c", "", "server:port to test")
}

const (
	forum_ready   = "A\rFORUM_READY\r\x04"
	visulas_ready = "A\rVISULAS_READY\r\x04"
	receive_ready = "A\rFORUM_RECEIVE_READY\r\x04"
	review_error  = "A\rREVIEW_ERROR\r\x04"
	review_ready  = "A\rREVIEW_READY\r\x04"
)

func convert(txt string) string {
	return strings.ReplaceAll(txt, "\r", "\r\n")
}

func read(conn net.Conn) {
	fmt.Printf("---------- read from Silex\n")

	//var buf bytes.Buffer
	//n, err := io.Copy(&buf, conn)
	//if err != nil {
	//	panic(err)
	//}
	//txt := string(buf.Bytes())

	//buf := make([]byte, 102400)
	//tmp := make([]byte, 256)
	//n := 0
	//for {
	//	conn.SetDeadline(time.Now().Add(time.Second * 3))
	//
	//	nread, err := conn.Read(tmp)
	//	if err != nil {
	//		if err != io.EOF {
	//			break
	//		}
	//		break
	//	}
	//	n += nread
	//	buf = append(buf, tmp[:n]...)
	//}
	//txt := string(buf[:n])

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
	read(conn)
	write(conn, review_ready)

	return nil
}

func main() {
	defer common.Cleanup()

	common.New(&common.App{"visulas", "1.0.0", "2019", "Can connect to server:port", "mpetavy", common.APACHE, "VISULAS query via Silex", false, nil, nil, nil, run, time.Duration(0)}, []string{"c"})
	common.Run()
}
