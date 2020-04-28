package glog

import (
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
)

type tcper struct {
	addr     string
	conn     net.Conn
	mqQueue  *queue
	count    uint32
	cond     *sync.Cond
	locker   sync.Mutex
	isLinked bool

	currMsg []byte
	currPos int
}

var (
	tcperMap       map[string]*tcper
	tcperMapLocker sync.RWMutex
)

func init() {
	tcperMap = make(map[string]*tcper)
}

func (self *tcper) addRef() {
	atomic.AddUint32(&self.count, 1)
}

func (self *tcper) writeMsg(msg []byte) {
	self.mqQueue.Push(msg)
	self.cond.Signal()
}

func (self *tcper) release() (needDelete bool) {
	atomic.AddUint32(&self.count, ^uint32(-(-1)-1))
	if atomic.LoadUint32(&self.count) == 0 {
		needDelete = true
	}

	needDelete = false
	self.cond.Signal()
	return
}

func (self *tcper) checkLink() {
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

func (self *tcper) flushToNet() {
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

func (self *tcper) forwardMsg() {

	for atomic.LoadUint32(&self.count) > 0 || self.mqQueue.Len() > 0 {

		self.checkLink()

		if self.isLinked {
			if len(self.currMsg) > self.currPos {
				self.flushToNet()
			} else {
				tmpMsg, err := self.mqQueue.Pop()
				if err != nil {
					if atomic.LoadUint32(&self.count) == 0 {
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

func releaseTcper(addr string) {
	tcperMapLocker.RLock()
	if tcpPtr, ok := tcperMap[addr]; ok {
		needDelete := tcpPtr.release()
		if needDelete {
			delete(tcperMap, addr)
		}
	}
	tcperMapLocker.RUnlock()
}

func getTcper(addr string) *tcper {
	tcperMapLocker.Lock()
	if tcpPtr, ok := tcperMap[addr]; ok {
		tcpPtr.addRef()
		tcperMapLocker.Unlock()
		return tcpPtr
	} else {
		tcpPtr := new(tcper)
		tcpPtr.count = 1
		tcpPtr.mqQueue = newQueue()
		tcpPtr.addr = addr
		tcpPtr.cond = sync.NewCond(&tcpPtr.locker)
		tcperMap[addr] = tcpPtr
		go tcpPtr.forwardMsg()
		tcperMapLocker.Unlock()
		return tcpPtr
	}
}

type tcpOutput struct {
	tcper *tcper
	addr  string
}

func (self *tcpOutput) checkOutput() error {
	return nil
}

func (self *tcpOutput) releaseOutput() {
	if self.tcper != nil {
		releaseTcper(self.addr)
	}
}

func (self *tcpOutput) writeMsg(msg []byte) {
	if self.tcper != nil {
		self.tcper.writeMsg(msg)
	}
}

func (self *tcpOutput) msgLen() uint32 {
	return 0
}

///////////////////////
type tcpOutputCreator struct {
}

func (self *tcpOutputCreator) createOutput(info outputSetinfo) outputInterface {

	return &tcpOutput{
		addr:  info.conf,
		tcper: getTcper(info.conf),
	}
}
