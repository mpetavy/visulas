package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/mpetavy/common"
)

var (
	readFilename  *string
	writeFilename *string
	client        *string
	server        *string
	readTimeout   *int
	stepTimeout   *int
	loopCount     *int
	useTls        *bool
	useKey        *bool
)

const (
	dumpfile = "QQ1WSVNVTEFTNTAwDTEuMA1TdGFuZGFyZCAgICAgICAgICAgICAgICANT0QNMw1TZWxlY3RpdmUgICAgICAgICAgICAgICANLS0gICAgICAgICAgICAgICAgICAgICAgDTAzMTINMDAwNQ0wMDA1DTAwMDUNMDAwMA0wMDAwDTAwMDANMDA1MA0wMDUwDTAwNTANMDAwMA0wMDAwDTAwMDANMDAyNTUNMDAyNTUNMDAyNTUNMDAwMDc5NTYwDTAwMDAwMDAwMDANMg02RUY2QTU5NDg1NzVCREEzRjk1Nzk4NzA4RjU1RkJGOQ0E"
)

func init() {
	writeFilename = flag.String("w", "visualas.dmp", "filename for dumping received Visulas data")
	readFilename = flag.String("r", "visualas.dmp", "filename for sending Visulas data")
	client = flag.String("c", "", "client socket address to read from")
	server = flag.String("s", "", "server socket address to listen to")
	readTimeout = flag.Int("rt", 3000, "read timeout")
	stepTimeout = flag.Int("st", 1000, "step timeout")
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
	common.Init(false, "1.0.0", "", "", "2019", "Emulation tool", "mpetavy", fmt.Sprintf("https://github.com/mpetavy/%s", common.Title()), common.APACHE, nil, nil, nil, run, 0)
}

func convert(txt string) string {
	return strings.ReplaceAll(txt, "\r", "\r\n")
}

func read(reader io.Reader) ([]byte, error) {
	if *stepTimeout > 0 {
		time.Sleep(common.MillisecondToDuration(*stepTimeout))
	}

	if *client != "" && *readTimeout > 0 {
		reader = common.NewTimeoutReader(reader, common.MillisecondToDuration(*readTimeout), true)
	}

	if *server != "" {
		common.Info("--------------------")
	}

	common.Info("read from...")

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

	common.Info("%d bytes read: %s", buf.Len(), convert(txt))

	return buf.Bytes(), nil
}

func write(writer io.Writer, txt string) error {
	if *stepTimeout > 0 {
		time.Sleep(common.MillisecondToDuration(*stepTimeout))
	}

	if *client != "" {
		common.Info("--------------------")
	}
	common.Info("write to: %s...", convert(txt))

	var err error

	n, err := writer.Write([]byte(txt))
	if common.Error(err) {
		return err
	}

	common.Info("%d bytes written: %s", n, convert(txt))

	return nil
}

func process(connector common.EndpointConnector) error {
	conn, err := connector()
	if common.Error(err) {
		return err
	}

	defer func() {
		common.Error(conn.Close())
	}()

	if *client != "" {
		write(conn, forum_ready)

		var ba []byte

		if bytes.Compare(ba, []byte(visulas_ready)) != 0 {
			write(conn, forum_receive_ready)
		}

		for {
			ba, err = read(conn)
			if common.Error(err) {
				return err
			}

			if bytes.Compare(ba, []byte(visulas_ready)) != 0 {
				break
			}
		}

		common.Error(os.WriteFile(*writeFilename, ba, common.DefaultFileMode))

		write(conn, review_ready)
	} else {
		var fileContent []byte
		var err error

		if *readFilename != "" {
			fileContent, err = ioutil.ReadFile(*readFilename)
			if common.Error(err) {
				return err
			}
		} else {
			fileContent, err = base64.StdEncoding.DecodeString(dumpfile)
			if common.Error(err) {
				return err
			}
		}

		ba, err := read(conn)
		if common.Error(err) {
			return err
		}

		if bytes.Compare(ba, []byte(forum_ready)) != 0 {
			fmt.Errorf("expected %s", convert(forum_ready))
		}

		write(conn, visulas_ready)

		ba, err = read(conn)
		if common.Error(err) {
			return err
		}

		if bytes.Compare(ba, []byte(forum_receive_ready)) != 0 {
			fmt.Errorf("expected %s", convert(forum_receive_ready))
		}

		write(conn, string(fileContent))

		ba, err = read(conn)
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

	for i := 0; i < *loopCount; i++ {
		if *server != "" {
			i--

			if *useKey {
				common.Info("--------------------")
				common.Info("Press RETURN to get ready...")
				reader := bufio.NewReader(os.Stdin)
				reader.ReadString('\n')
			}
		}

		common.Info("--------------------")
		common.Info("#%d", i)

		err := process(connector)
		if common.Error(err) {
			return err
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
