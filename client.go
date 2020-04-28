package glog

import "fmt"

type clientInterface interface {
	WriteMsg([]byte, Level, string, string)
	Start()
	Close()
	Type() string
}

func NewClinet(sType, addr string) clientInterface {
	if sType == "tcp" {
		return newTcpClient(addr)
	} else if sType == "unixgram" {
		return newUnixgramClient(addr)
	} else {
		panic(fmt.Sprintf("unknow client type <%s>", sType))
	}

	return nil
}

func regClinet() {

}
