package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/mpetavy/common"
)

var (
	filename    *string
	client      *string
	server      *string
	readTimeout *int
	stepTimeout *int
	loopTimeout *int
	loopCount   *int
	useTls      *bool
	useKey      *bool
)

const (
	dumpfile = "QQ1WSVNVTEFTNTAwDTEuMA1TdGFuZGFyZCAgICAgICAgICAgICAgICANT0QNMw1TZWxlY3RpdmUgICAgICAgICAgICAgICANLS0gICAgICAgICAgICAgICAgICAgICAgDTAzMTINMDAwNQ0wMDA1DTAwMDUNMDAwMA0wMDAwDTAwMDANMDA1MA0wMDUwDTAwNTANMDAwMA0wMDAwDTAwMDANMDAyNTUNMDAyNTUNMDAyNTUNMDAwMDc5NTYwDTAwMDAwMDAwMDANMg02RUY2QTU5NDg1NzVCREEzRjk1Nzk4NzA4RjU1RkJGOQ0E"
)

func init() {
	filename = flag.String("f", "", "filename for dumping (-c) received data or send data (-s)")
	client = flag.String("c", "", "client socket address to read from")
	server = flag.String("s", "", "server socket address to listen to")
	readTimeout = flag.Int("rt", 3000, "read timeout")
	stepTimeout = flag.Int("st", 1000, "pacer timeout")
	loopTimeout = flag.Int("lt", 0, "loop timeout")
	loopCount = flag.Int("lc", 1, "loop count")
	useTls = flag.Bool("tls", false, "use tls")
	useKey = flag.Bool("key", false, "use tls")
}

const (
	forum_ready         = "A\rFORUM_READY\r\x04"
	visulas_ready       = "A\rVISULAS_READY\r\x04"
	forum_receive_ready = "A\rFORUM_RECEIVE_READY\r\x04"
	//review_error  = "A\rREVIEW_ERROR\r\x04"
	review_ready = "A\rREVIEW_READY\r\x04"
)

func init() {
	common.Init(false, "1.0.1", "", "", "2019", "Emulation tool", "mpetavy", fmt.Sprintf("https://github.com/mpetavy/%s", common.Title()), common.APACHE, nil, nil, nil, run, 0)
}

func convert(txt string) string {
	return strings.ReplaceAll(txt, "\r", "\r\n")
}

func read(reader io.Reader, asString bool) ([]byte, error) {
	if *stepTimeout > 0 {
		time.Sleep(common.MillisecondToDuration(*stepTimeout))
	}

	if *client != "" && *readTimeout > 0 {
		//reader = common.NewTimeoutReader(reader, common.MillisecondToDuration(*readTimeout), true)
		reader = common.NewTimeoutReader(reader, false, func() (context.Context, context.CancelFunc) {
			return context.WithTimeout(context.Background(), common.MillisecondToDuration(*readTimeout))
		})
	}

	common.Info("--------------------")
	common.Info("read...")

	b1 := make([]byte, 1)
	buf := bytes.Buffer{}

	for {
		nread, err := reader.Read(b1)
		if common.Error(err) {
			return nil, err
		}
		if nread > 0 {
			buf.Write(b1)

			if b1[0] == '\x04' {
				break
			}
		}
	}

	txt := string(buf.Bytes())

	if asString {
		common.Info("read %d bytes: %s", buf.Len(), convert(txt))
	} else {
		common.Info("read %d bytes: %+q", buf.Len(), txt)
	}

	return buf.Bytes(), nil
}

func write(writer io.Writer, txt string, asString bool) error {
	if *stepTimeout > 0 {
		time.Sleep(common.MillisecondToDuration(*stepTimeout))
	}

	common.Info("--------------------")
	if asString {
		common.Info("write %d bytes: %s", len(txt), convert(txt))
	} else {
		common.Info("write %d bytes: %+q", len(txt), txt)
	}

	var err error

	n, err := writer.Write([]byte(txt))
	if common.Error(err) {
		return err
	}

	common.Info("write %d bytes", n)

	return nil
}

func process(conn io.ReadWriteCloser) error {
	if *client != "" {
		write(conn, forum_ready, false)

		ba, err := read(conn, false)
		if common.Error(err) {
			return err
		}

		if bytes.Compare(ba, []byte(visulas_ready)) != 0 {
			return fmt.Errorf("expected %s", convert(visulas_ready))
		}

		write(conn, forum_receive_ready, false)

		ba, err = read(conn, true)
		if common.Error(err) {
			return err
		}

		if *filename != "" {
			common.Error(os.WriteFile(*filename, ba, common.DefaultFileMode))
		}

		write(conn, review_ready, false)
	} else {
		var fileContent []byte
		var err error

		if *filename != "" {
			fileContent, err = os.ReadFile(*filename)
			if common.Error(err) {
				return err
			}
		} else {
			fileContent, err = base64.StdEncoding.DecodeString(dumpfile)
			if common.Error(err) {
				return err
			}
		}

		ba, err := read(conn, false)
		if common.Error(err) {
			return err
		}

		if bytes.Compare(ba, []byte(forum_ready)) != 0 {
			return fmt.Errorf("expected %s", convert(forum_ready))
		}

		write(conn, visulas_ready, false)

		ba, err = read(conn, false)
		if common.Error(err) {
			return err
		}

		if bytes.Compare(ba, []byte(forum_receive_ready)) != 0 {
			return fmt.Errorf("expected %s", convert(forum_receive_ready))
		}

		write(conn, string(fileContent), true)

		ba, err = read(conn, false)
		if common.Error(err) {
			return err
		}
	}

	return nil
}

func run() error {
	var err error
	var tlsConfig *tls.Config

	if *useTls {
		tlsConfig, err = common.NewTlsConfigFromFlags()
		if common.Error(err) {
			return err
		}
	}

	address := *client
	if address == "" {
		common.Info("Listen on %s...", *server)

		address = *server

		*loopCount = 9999999
		*stepTimeout = 0
	} else {
		common.Info("Connect %s...", *client)
	}

	ep, connector, err := common.NewEndpoint(address, *client != "", tlsConfig)
	if common.Error(err) {
		return err
	}

	err = ep.Start()
	if common.Error(err) {
		return err
	}

	defer func() {
		common.Error(ep.Stop())
	}()

	conn, err := connector()
	if common.Error(err) {
		return err
	}

	defer func() {
		common.Error(conn.Close())
	}()

	for i := 0; i < *loopCount; i++ {
		if *server != "" {
			if *useKey {
				common.Info("--------------------")
				common.Info("Press RETURN to get ready...")
				reader := bufio.NewReader(os.Stdin)
				reader.ReadString('\n')
			} else {
				if *loopTimeout > 0 {
					time.Sleep(common.MillisecondToDuration(*loopTimeout))
				}
			}
		}

		common.Info("--------------------")
		common.Info("#%d", i)

		err := process(conn)
		if common.Error(err) {
			if *client != "" {
				return err
			}
		}

		if i < *loopCount-1 {
			time.Sleep(common.MillisecondToDuration(*stepTimeout))
		}
	}

	return nil
}

func main() {
	defer common.Done()

	common.Run([]string{"c|s"})
}
