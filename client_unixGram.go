package glog

import (
	"fmt"
	"net"
)

type unixgramClient struct {
	addr string
	conn *net.UnixConn
}

func (self *unixgramClient) WriteMsg(msg []byte, lv Level, logName, tplName string) {

	sendMsg, err := packMsg(msg, lv, logName, tplName)
	if err != nil {
		fmt.Println(fmt.Sprintf("pack msg err:%s", err.Error()))
		return
	}

	if self.conn != nil {
		_, err = self.conn.Write(sendMsg)
		if err != nil {
			fmt.Println(fmt.Sprintf("pack msg err:%s", err.Error()))
		}
	}
}

func (self *unixgramClient) Start() {
	uaddr, err := net.ResolveUnixAddr("unixgram", self.addr)
	if err != nil {
		fmt.Println("unixgramClient, resolve unix addr err->", err)
		return
	}

	conn, err := net.DialUnix("unixgram", nil, uaddr)
	if err != nil {
		fmt.Println("unixgramClient DialUnix, resolve unix addr err->", err)
	}

	self.conn = conn
}

func (self *unixgramClient) Close() {
	if self.conn != nil {
		self.conn.Close()
		self.conn = nil
	}
}

func (self *unixgramClient) Type() string {
	return "unixgram"
}

func newUnixgramClient(addr string) *unixgramClient {

	return &unixgramClient{
		addr: addr,
	}
}
