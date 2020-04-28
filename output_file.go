package glog

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func pathExistsDir(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, nil
	}

	if !info.IsDir() {
		return false, nil
	}

	return true, nil
}

// file output
type fileOutput struct {
	filer    *filer
	sInfo    outputSetinfo
	currPath string
	cfgDir   string
	cfgFile  string
}

func (self *fileOutput) checkOutput() error {

	preFile := self.cfgDir + "/" + self.cfgFile
	currentTime := time.Now()
	Year := fmt.Sprintf("%02d", currentTime.Year())
	Month := fmt.Sprintf("%02d", int(currentTime.Month()))
	Day := fmt.Sprintf("%02d", currentTime.Day())
	Hour := fmt.Sprintf("%02d", currentTime.Hour())
	Min := fmt.Sprintf("%02d", currentTime.Minute())

	var placer = strings.NewReplacer("$remoteIP", self.sInfo.logbase.fromhostIP, "$YEAR", Year,
		"$MONTH", Month, "$DAY", Day, "$HOUR", Hour, "$logName", self.sInfo.logbase.logName,
		"$MINUTE", Min)

	preFile = placer.Replace(preFile)
	if preFile != self.currPath {
		preDir := placer.Replace(self.cfgDir)

		// 判断路径是否存在
		if ok, _ := pathExistsDir(preDir); !ok {
			err := os.MkdirAll(preDir, os.ModePerm)
			if err != nil {
				return errors.New(fmt.Sprintf("MkdirAll -->err<%s>", err.Error()))
			}
		}

		if self.filer != nil {
			releaseFiler(self.currPath)
		}

		newfile, err := getFiler(preFile)
		if newfile != nil {
			self.filer = newfile
			self.currPath = preFile
		} else {
			return errors.New(fmt.Sprintf("MkdirAll -->err<%s>", err.Error()))
		}

		return nil
	}

	return nil
}

func (self *fileOutput) writeMsg(msg []byte) {
	self.filer.addData(msg)
}

func (self *fileOutput) msgLen() uint32 {
	if self.filer != nil {
		return self.filer.msgLen()
	}

	return 0
}

func (self *fileOutput) releaseOutput() {
	if self.filer != nil {
		releaseFiler(self.currPath)
	}
}

// output creator

type fileOutputCreator struct {
}

func (self *fileOutputCreator) createOutput(info outputSetinfo) outputInterface {

	return &fileOutput{
		cfgDir:  filepath.Dir(info.conf),
		cfgFile: filepath.Base(info.conf),
		sInfo:   info,
	}
}
