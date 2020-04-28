package glog

import (
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
)

type tcpClient struct {
	addr    string
	mqQueue *queue

	conn     net.Conn
	cond     *sync.Cond
	locker   sync.Mutex
	isLinked bool
	bCount   uint32
	currMsg  []byte
	currPos  int
}

func (self *tcpClient) WriteMsg(msg []byte, lv Level, logName, tplName string) {

	sendMsg, err := packMsg(msg, lv, logName, tplName)
	if err != nil {
		fmt.Println(fmt.Sprintf("pack msg err:%s", err.Error()))
		return
	}

	self.mqQueue.Push(sendMsg)
	self.cond.Signal()
}

func (self *tcpClient) Start() {
	atomic.AddUint32(&self.bCount, 1)
	go self.forwardMsg()
}

func (self *tcpClient) Close() {
	atomic.AddUint32(&self.bCount, ^uint32(-(-1)-1))
	self.cond.Signal()
}

func (self *tcpClient) Type() string {
	return "tcp"
}

func (self *tcpClient) checkLink() {
	if !self.isLinked {
		var tryTimes int
		for {
			tryTimes++
			conn, err := net.Dial("tcp", self.addr)
			if err != nil {
				if tryTimes > 3 {
					printSyslog(fmt.Sprintf("connect err! conf:%s", self.addr))
					break
				}
				continue
			}

			self.isLinked = true
			self.conn = conn
			break
		}
	}
}

func (self *tcpClient) flushToNet() {
	netWriter, ok := self.conn.(io.Writer)
	if !ok {
		printSyslog("net connect can not change io.Writer")
		return
	}
	// 上次没发玩的消息
	n, err := netWriter.Write(self.currMsg[self.currPos:])
	if err != nil {
		printSyslog(fmt.Sprintf("net write data er: %s", err.Error()))
		// 写入出错 重新连接 继续写？
		self.isLinked = false
		self.conn.Close()
		return
	}

	self.currPos += n
}

func (self *tcpClient) forwardMsg() {

	for atomic.LoadUint32(&self.bCount) > 0 || self.mqQueue.Len() > 0 {

		self.checkLink()

		if self.isLinked {
			if len(self.currMsg) > self.currPos {
				self.flushToNet()
			} else {
				tmpMsg, err := self.mqQueue.Pop()
				if err != nil {
					if atomic.LoadUint32(&self.bCount) == 0 {
						break
					}
					// 没消息了 等待下
					self.locker.Lock()
					self.cond.Wait()
					self.locker.Unlock()
					continue
				}

				if tmpMsg, ok := tmpMsg.([]byte); ok {
					self.currMsg = tmpMsg
					self.currPos = 0
				}

				self.flushToNet()
			}

		} else {
			var overload = self.mqQueue.mqOverload()
			if overload > 0 {
				printSyslog(fmt.Sprintf("<%s> May overload, message queue length = %d", self.addr, overload))
			}

			if self.mqQueue.Len() > 1024*10 {
				printSyslog(fmt.Sprintf("<%s> too many msg wait handle, and give up them ", self.addr, self.mqQueue.Len()))
				//
			}

		}
	}

	if self.isLinked {
		self.conn.Close()
	}
}

//
func newTcpClient(addr string) *tcpClient {

	tcpClinet := new(tcpClient)
	tcpClinet.mqQueue = newQueue()
	tcpClinet.addr = addr
	tcpClinet.cond = sync.NewCond(&tcpClinet.locker)
	return tcpClinet
}
