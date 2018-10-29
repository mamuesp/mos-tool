package timezone

import (
	"fmt"
	"archive/zip"
	"io"
	"log"
	"bytes"
	"errors"
  "strings"	
  "syscall"
  "os"
  
  "cesanta.com/mos/config"
	"cesanta.com/mos/dev"
	"github.com/cesanta/errors"
  flag "github.com/spf13/pflag"
)

var badData = errors.New("malformed time zone information")

func ReadTzData(name string) (tzInfo string, err error) {
	root, _ := syscall.Getenv("GOROOT")
	source := root + "/lib/time/zoneinfo.zip"

	r, err := zip.OpenReader(source)
	if err != nil {
		log.Fatal(err)
	}
	defer r.Close()
	
	var buf bytes.Buffer

	for _, f := range r.File {
		 if !f.FileInfo().IsDir() {
			if f.Name == name {
				rc, err := f.Open()
				if err != nil {
					log.Fatal(err)
				}
				_, err = io.Copy(&buf, rc)
				if err != nil {
					log.Fatal(err)
				}
				rc.Close()
				
				magic := bytes.Index(buf.Bytes(), []byte("TZif"))
				if magic == -1 {
					return "", badData
				}			

				line := "-"
				var tzInfo string
				for line != "" {
					tzInfo = line
					line, _ = buf.ReadString(0x0A)
				}
				buf.Reset()
				return strings.TrimSpace(tzInfo), err
			}
		}
	}
	
	return "", err
}

func Set(ctx context.Context, devConn *dev.DevConn) error {
	args := flag.Args()
	if len(args) < 1 {
		return errors.Errorf("Usage: %s tzset TIMEZONE_NAME", os.Args[0])
	}

  tzName := args[0]
  tzInfo, err := ReadTzData(tzName)
	fmt.Printf("TZ Info %q, %q\n", tzInfo, err)

	params := []string{
		fmt.Sprintf("sys.tz_info=%s", tzInfo)
	}
  
	return config.SetWithArgs(ctx, devConn, params)
}
