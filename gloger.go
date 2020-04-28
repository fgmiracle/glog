package glog

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

type logBase struct {
	logName    string
	fromhostIP string
}
type logInfo struct {
	logbase logBase
	time    int64
	level   Level
}

const (
	registerVerid = uint16(1000)
)

const (
	tplFile       = "file"
	tplUnixGram   = "unixGram"
	tplUnixDomain = "unixDomain"
	tplTcp        = "tcp"
	tplUdp        = "udp"
)

// 系统日志
var (
	globleLogFile *os.File
)

func initGlobleLog(path string) {
	if path == "" {
		dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			log.Fatal(err)
		}

		path = dir + "/glog.log"
	}

	fd, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Println("can not open file!->", path)
		return
	}

	// 时区
	time.LoadLocation("Asia/Shanghai")
	globleLogFile = fd
}

func printSyslog(msg string) {
	//fmt.Println(msg)
	if globleLogFile != nil {
		globleLogFile.WriteString(getTimeString(time.Now()))
		globleLogFile.Write([]byte(msg))
		globleLogFile.WriteString("\n")
	} else {
		log.Println(msg)
	}
}

func getTimeString(t time.Time) string {
	return fmt.Sprintf("[%02d-%02d-%02d %02d:%02d:%02d.%06d]", t.Year(),
		int(t.Month()), t.Day(), t.Hour(), t.Minute(),
		t.Second(), t.Nanosecond()/1000)
}
