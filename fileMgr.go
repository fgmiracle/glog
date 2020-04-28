package glog

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// file mgr
type filer struct {
	logFile   *os.File
	mqQueue   *queue
	count     uint32
	path      string
	cond      *sync.Cond
	logLocker sync.Mutex
	exit      chan bool
}

var (
	dataPool       *sync.Pool
	fileMgr        = make(map[string]*filer)
	filerMapLocker sync.RWMutex
)

func init() {
	dataPool = new(sync.Pool)
	dataPool.New = func() interface{} {
		return make([]interface{}, 10)
	}
}

func getFiler(path string) (*filer, error) {
	filerMapLocker.Lock()
	if filePtr, ok := fileMgr[path]; ok {
		filePtr.addRef()
		filerMapLocker.Unlock()
		return filePtr, nil
	} else {
		fd, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			filerMapLocker.Unlock()
			return nil, err
		}

		filePtr := new(filer)
		filePtr.logFile = fd
		filePtr.count = 1
		filePtr.mqQueue = newQueue()
		filePtr.path = path
		filePtr.cond = sync.NewCond(&filePtr.logLocker)
		filePtr.exit = make(chan bool, 1)

		fileMgr[path] = filePtr
		go flushData(filePtr)
		filerMapLocker.Unlock()
		return filePtr, nil
	}
}

func releaseFiler(path string) {
	filerMapLocker.RLock()
	if filePtr, ok := fileMgr[path]; ok {
		needDelete := filePtr.release()
		if needDelete {
			delete(fileMgr, path)
		}
	}
	filerMapLocker.RUnlock()
}

func (f *filer) addRef() {
	atomic.AddUint32(&f.count, 1)
}

func (f *filer) msgLen() uint32 {
	return f.mqQueue.Len()
}

func (f *filer) addData(msg []byte) {
	f.mqQueue.Push(msg)
	f.cond.Signal()
}

func (f *filer) flushToFile() uint32 {
	msgList := dataPool.Get().([]interface{})
	var count = f.mqQueue.Pops(msgList)
	if count > 0 {
		for i := uint32(0); i < count; i++ {
			tmpmsg := msgList[i]
			msg, ok := tmpmsg.([]byte)
			if !ok {
				printSyslog("flushToFile msg can not change []byte")
				continue
			}
			var pos int
			for pos < len(msg) {
				n, err := f.logFile.Write(msg[pos:])
				if err != nil {
					printSyslog(fmt.Sprintf("flushToFile write data er: %s", err))
					// 写入出错 继续写
					//break
				}

				pos += n
			}
		}

		var overload = f.mqQueue.mqOverload()
		if overload > 0 {
			printSyslog(fmt.Sprintf("<%s> May overload, message queue length = %d", f.path, overload))
		}
	}

	dataPool.Put(msgList)
	return count
}

func (self *filer) release() (needDelete bool) {
	atomic.AddUint32(&self.count, ^uint32(-(-1)-1))
	if atomic.LoadUint32(&self.count) == 0 {
		go exitMonitor(self)
		needDelete = true
	}
	needDelete = false
	return
}

func exitMonitor(f *filer) {
	for {
		select {
		case <-f.exit:
			return
		case <-time.After(1 * time.Second):
			f.cond.Broadcast()
		}
	}
}

func flushData(f *filer) {
	for {
		if f.flushToFile() == 0 {
			if atomic.LoadUint32(&f.count) == 0 {
				f.logFile.Close()
				f.exit <- true
				break
			}

			f.logLocker.Lock()
			f.cond.Wait()
			f.logLocker.Unlock()
		}
	}
}
