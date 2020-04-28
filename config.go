package glog

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
)

//output

type template struct {
	Name       string `xml:"name"`
	Path       string `xml:"path"`
	Worker     int    `xml:"worker"`
	Tcp        string `xml:"tcp"`
	UnixDomain string `xml:"unixDomain"`
	unixGram   string `xml:"unixGram"`
	Udp        string `xml:"udp"`
	logRule    bool   `xml:"logRule"`
}

type outList struct {
	FileList []template `xml:"template"`
}

//server
type server struct {
	TcpPort    int    `xml:"tcp"`
	UdpPort    int    `xml:"udp"`
	UnixDomain string `xml:"unixDomain"`
	UnixGram   string `xml:"unixGram"`
}
type groupItem struct {
	OutPut       outList `xml:"output"`
	Server       server  `xml:"server"`
	MaxMsgQueque int     `xml:"maxMsgQuequ"`
	SyslogPath   string  `xml:"syslogPath"`
	svrWorker    int     `xml:"svrWorker"`
}

type outputType struct {
	Type    string
	Path    string
	Worker  int
	logRule bool
}

var (
	outputList    = make(map[string]outputType)  // name -> path
	serverList    = make(map[string]interface{}) // type -> interface
	maxLenTplChan = 1024
	svrWorker     = 1
)

// ParseConfig ....
func parseConfig(path string) (bool, error) {
	file, err := os.Open(path) // For read access.
	if err != nil {
		printSyslog(fmt.Sprintf("parseConfig open error: %s\n", err))
		return false, err
	}
	defer file.Close()
	data, err := ioutil.ReadAll(file)
	if err != nil {
		printSyslog(fmt.Sprintf("parseConfig read error: %s\n", err))
		return false, err
	}

	configInfo := groupItem{}
	err = xml.Unmarshal(data, &configInfo)
	if err != nil {
		printSyslog(fmt.Sprintf("parseConfig Unmarshal error: %s\n", err))
		return false, err
	}

	//serverList
	t := reflect.TypeOf(&configInfo.Server).Elem()
	v := reflect.ValueOf(configInfo.Server)
	for i := 0; i < t.NumField(); i++ {
		serverList[t.Field(i).Tag.Get("xml")] = v.Field(i).Interface()
	}

	// outputList
	for _, info := range configInfo.OutPut.FileList {
		var (
			Path    string
			Type    string
			Worker  int
			logRule bool
		)

		if info.Path != "" {
			Path = info.Path
			Type = tplFile
			Worker = info.Worker
			logRule = info.logRule
		} else if info.UnixDomain != "" {
			Path = info.UnixDomain
			Type = tplUnixDomain
			Worker = info.Worker
		} else if info.unixGram != "" {
			Path = info.unixGram
			Type = tplUnixGram
			Worker = info.Worker
		} else if info.Tcp != "" {
			Path = info.Tcp
			Type = tplTcp
			Worker = info.Worker
		} else if info.Udp != "" {
			Path = info.Udp
			Type = tplUdp
			Worker = info.Worker
		}

		if Path != "" && Type != "" {
			outputList[info.Name] = outputType{
				Type:    Type,
				Path:    Path,
				Worker:  Worker,
				logRule: logRule,
			}
		}

	}

	//
	svrWorker = configInfo.svrWorker
	// chan 长度
	// if configInfo.MaxMsgQueque != 0 {
	// 	maxLenTplChan = configInfo.MaxMsgQueque
	// }

	initGlobleLog(configInfo.SyslogPath)
	return true, nil
}
