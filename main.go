package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"flag"
	"fmt"
	"github.com/mpetavy/common"
	"io"
	"os"
	"strconv"
	"sync"
	"time"
)

var (
	filename     *string
	client       *string
	server       *string
	readTimeout  *int
	pacerTimeout *int
	loopTimeout  *int
	loopCount    *int
	useTls       *bool
	useKey       *bool
	scale        *int
	tlsConfig    *tls.Config
)

const (
	dumpfile = "QQ1WSVNVTEFTNTAwDTEuMA1TdGFuZGFyZCAgICAgICAgICAgICAgICANT0QNMw1TZWxlY3RpdmUgICAgICAgICAgICAgICANLS0gICAgICAgICAgICAgICAgICAgICAgDTAzMTINMDAwNQ0wMDA1DTAwMDUNMDAwMA0wMDAwDTAwMDANMDA1MA0wMDUwDTAwNTANMDAwMA0wMDAwDTAwMDANMDAyNTUNMDAyNTUNMDAyNTUNMDAwMDc5NTYwDTAwMDAwMDAwMDANMg02RUY2QTU5NDg1NzVCREEzRjk1Nzk4NzA4RjU1RkJGOQ0E"
)

func init() {
	filename = flag.String("f", "", "filename for dumping (-c) received data or send data (-s)")
	client = flag.String("c", "", "client socket address to read from")
	server = flag.String("s", "", "server socket address to listen to")
	readTimeout = flag.Int("rt", 3000, "read timeout")
	pacerTimeout = flag.Int("pt", 0, "pacer timeout")
	loopTimeout = flag.Int("lt", 0, "loop timeout")
	loopCount = flag.Int("lc", 1, "loop count")
	useTls = flag.Bool("tls", false, "use tls")
	useKey = flag.Bool("key", false, "use tls")
	scale = flag.Int("scale", 1, "scale instances")
}

const (
	forum_ready         = "A\rFORUM_READY\r\x04"
	visulas_ready       = "A\rVISULAS_READY\r\x04"
	forum_receive_ready = "A\rFORUM_RECEIVE_READY\r\x04"
	//review_error  = "A\rREVIEW_ERROR\r\x04"
	review_ready = "A\rREVIEW_READY\r\x04"
)

func init() {
	common.Init("visulas", "1.3.0", "", "", "2019", "Emulation tool", "mpetavy", fmt.Sprintf("https://github.com/mpetavy/%s", common.Title()), common.APACHE, nil, nil, nil, run, 0)
}

func readBytes(reader io.Reader, timeout time.Duration, asString bool) ([]byte, error) {
	common.Info("--------------------")
	common.Info("read...")

	if timeout != 0 {
		reader = common.NewTimeoutReader(reader, true, common.MillisecondToDuration(*readTimeout))
	}

	ba := make([]byte, 1024)
	buf := bytes.Buffer{}

	for {
		n, err := reader.Read(ba)
		if common.Error(err) {
			return nil, err
		}

		if n > 0 {
			buf.Write(ba[:n])

			if ba[n-1] == '\x04' {
				break
			}
		}
	}

	common.Info("read %d bytes: %s", buf.Len(), common.PrintBytes(buf.Bytes(), asString))

	return buf.Bytes(), nil
}

func writeBytes(writer io.Writer, txt string, asString bool) error {
	if *pacerTimeout > 0 {
		common.Sleep(common.MillisecondToDuration(*pacerTimeout))
	}

	common.Info("--------------------")

	common.Info("write %d bytes: %s", len(txt), common.PrintBytes([]byte(txt), asString))

	_, err := writer.Write([]byte(txt))
	if common.DebugError(err) {
		return err
	}

	return nil
}

func bufferError(expected, received []byte) error {
	return fmt.Errorf("expected %s but received %s", common.PrintBytes(expected, false), common.PrintBytes(received, false))
}

func process(conn common.EndpointConnection) error {
	defer func() {
		common.Error(conn.Close())
	}()

	if *client != "" {
		common.Error(writeBytes(conn, forum_ready, false))

		ba, err := readBytes(conn, common.MillisecondToDuration(*readTimeout), true)
		if common.Error(err) {
			return err
		}
		if len(ba) == 0 {
			return nil
		}

		if bytes.Compare(ba, []byte(visulas_ready)) != 0 {
			err := bufferError([]byte(visulas_ready), ba)
			if common.Error(err) {
				return err
			}
		}

		common.Error(writeBytes(conn, forum_receive_ready, false))
		ba, err = readBytes(conn, common.MillisecondToDuration(*readTimeout), true)
		if common.Error(err) {
			return err
		}

		if *filename != "" {
			common.Error(os.WriteFile(*filename, ba, common.DefaultFileMode))
		}

		common.Error(writeBytes(conn, review_ready, false))
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

		ba, err := readBytes(conn, 0, false)
		if common.Error(err) {
			return err
		}

		if bytes.Compare(ba, []byte(forum_ready)) != 0 {
			err := bufferError([]byte(forum_ready), ba)
			if common.Error(err) {
				return err
			}
		}

		common.Error(writeBytes(conn, visulas_ready, false))

		ba, err = readBytes(conn, common.MillisecondToDuration(*readTimeout), true)
		if common.Error(err) {
			return err
		}

		if bytes.Compare(ba, []byte(forum_receive_ready)) != 0 {
			err := bufferError([]byte(forum_receive_ready), ba)
			if common.Error(err) {
				return err
			}
		}

		common.Error(writeBytes(conn, string(fileContent), true))

		ba, err = readBytes(conn, common.MillisecondToDuration(*readTimeout), false)
		if common.Error(err) {
			return err
		}
	}

	return nil
}

func instance(address string) error {
	if *server != "" {
		common.Info("Listen on %s...", address)
	} else {
		common.Info("Connect %s...", address)
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

	var conn common.EndpointConnection

	for i := 0; i < *loopCount; i++ {
		if *useKey {
			common.Info("--------------------")
			common.Info("Press RETURN to get ready...")
			reader := bufio.NewReader(os.Stdin)
			_, err := reader.ReadString('\n')
			common.DebugError(err)

		}

		common.Info("connection open")
		conn, err = connector()
		if common.Error(err) {
			return err
		}

		common.Info("#%d", i)

		common.Error(process(conn))

		if i < *loopCount-1 {
			common.Sleep(common.MillisecondToDuration(*loopTimeout))
		}
	}

	return nil
}

func run() error {
	var err error

	if *useTls {
		tlsConfig, err = common.NewTlsConfigFromFlags()
		if common.Error(err) {
			return err
		}
	}

	address := *client
	if address == "" {
		address = *server

		*loopCount = 9999999

		if *scale > 1 {
			*useKey = false
		}
	} else {
		*scale = 1
	}

	wg := sync.WaitGroup{}

	for i := 0; i < *scale; i++ {
		wg.Add(1)
		go func(address string) {
			defer common.UnregisterGoRoutine(common.RegisterGoRoutine(1))

			defer wg.Done()

			common.Error(instance(address))
		}(address)

		if *scale > 1 {
			a, err := strconv.Atoi(address[1:])
			if common.Error(err) {
				return err
			}

			a++
			address = fmt.Sprintf(":%d", a)
		}
	}

	wg.Wait()

	return nil
}

func main() {
	common.Run([]string{"c|s"})
}
