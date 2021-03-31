package main

import (
	"bytes"
	"crypto/tls"
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
	common.Init(false, "1.0.0", "", "2019", "VISULAS query via Silex", "mpetavy", fmt.Sprintf("https://github.com/mpetavy/%s", common.Title()), common.APACHE, nil, nil, nil, run, 0)
}

func convert(txt string) string {
	return strings.ReplaceAll(txt, "\r", "\r\n")
}

func read(reader io.Reader, expect string) []byte {
	if *stepTimeout > 0 {
		time.Sleep(common.MillisecondToDuration(*stepTimeout))
	}

	if *readTimeout > 0 {
		reader = common.NewTimeoutReader(reader, common.MillisecondToDuration(*readTimeout), true)
	}

	common.Info("---------- read from Silex: %+q", expect)

	b1 := make([]byte, 1)
	buf := bytes.Buffer{}

	for {
		nread, err := reader.Read(b1)
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

	common.Info("%d bytes read", buf.Len())
	common.Info("%s", convert(txt))

	return buf.Bytes()
}

func write(writer io.Writer, txt string) {
	if *stepTimeout > 0 {
		time.Sleep(common.MillisecondToDuration(*stepTimeout))
	}

	common.Info("---------- write to Silex: %+q", txt)

	var err error

	n, err := writer.Write([]byte(txt))
	if err != nil {
		panic(err)
	}
	common.Info("%d bytes written", n)
	common.Info("%s", convert(txt))
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

	ep, connector, err := common.NewEndpoint(*address, true, tlsConfig)
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

	common.Info("%s connected successfully", *address)

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
