package glog

import (
	"encoding/binary"
	"fmt"
	"sync"
	"time"
)

type outputInterface interface {
	checkOutput() error
	writeMsg([]byte)
	releaseOutput()
	msgLen() uint32
}

type outputCreator interface {
	createOutput(outputSetinfo) outputInterface
}

type outputSetinfo struct {
	logbase logBase
	conf    string
}

type tplMessage struct {
	log logInfo
	msg []byte
}

type tplInfo struct {
	mq      *queue
	lock    sync.Mutex
	cond    *sync.Cond
	tpl     outputType
	currLog *tplMessage
	parts   []partFunc
	fOutput outputCreator
}

var (
	tplList  map[string]*tplInfo
	overTime = int64(10)
)

func (self *tplInfo) createOutput(info outputSetinfo) outputInterface {
	if self.fOutput != nil {
		return self.fOutput.createOutput(info)
	}

	return nil
}

////////////////
func tplInit() {
	tplList = make(map[string]*tplInfo)
	for name, info := range outputList {
		temTpl := new(tplInfo)
		temTpl.mq = newQueue()
		temTpl.tpl = info
		temTpl.cond = sync.NewCond(&temTpl.lock)
		if info.logRule {
			temTpl.parts = []partFunc{logPart_colorbegin}
			temTpl.parts = append(temTpl.parts, logPart_time, logPart_level, logPart_data, logPart_colorend, logPart_line)
		} else {
			temTpl.parts = []partFunc{logPart_data}
			temTpl.parts = append(temTpl.parts, logPart_line)
		}

		goNum := info.Worker
		if goNum < 1 {
			goNum = 1
		}

		var tmpCreator outputCreator
		switch info.Type {
		case tplFile:
			tmpCreator = new(fileOutputCreator)
		case tplTcp:
			tmpCreator = new(tcpOutputCreator)
		default:
			printSyslog(fmt.Sprintf("output type error! Type:%s", info.Type))
			break
		}

		temTpl.fOutput = tmpCreator
		for i := 0; i < goNum; i++ {
			go tplRun(temTpl)
		}

		printSyslog(fmt.Sprintf("out tpl name<%s>, addr<%s> parts:%d", info.Type, info.Path, len(temTpl.parts)))
		tplList[name] = temTpl
	}

	go tplWake()
}

func tplWake() {
	var checkWakeTime = time.Now().Unix()
	for {
		currTime := time.Now().Unix()
		if checkWakeTime < currTime-5 {
			for _, tpl := range tplList {
				tpl.cond.Broadcast()
			}

			checkWakeTime = currTime
		}
		time.Sleep(time.Duration(1) * time.Second)
	}
}

func tplMsgHandle(tpl string, log *tplMessage, src []byte) []byte {
	if tpl, ok := tplList[tpl]; ok {
		var data []byte
		if tpl.tpl.Type == tplFile {
			data = make([]byte, 1024)
			data = data[:0]
			tpl.currLog = log
			for _, p := range tpl.parts {
				p(&data, tpl)
			}
		} else {
			// 除了文件 其他都是转发 tcp  udp
			data = make([]byte, len(src)+streamHeadSize)
			binary.LittleEndian.PutUint16(data, uint16(len(src)))
			copy(data[2:], src)
		}
		return data
	}
	return nil
}

func tplDispatch(tplName, logname string, msg *tplMessage) {
	if tpl, ok := tplList[tplName]; ok {
		//printSyslog(fmt.Sprintf("tplDispatch--->push:%s,len:%d", logname, tpl.mq.Len()))
		tpl.mq.Push(msg)
		//printSyslog(fmt.Sprintf("tplDispatch--->pop:%s,len:%d", logname, tpl.mq.Len()))
		tpl.cond.Signal()
	} else {
		printSyslog(fmt.Sprintf("can not find tpl<%s>", tplName))
	}
}

func tplRun(tpl *tplInfo) {

	linkList := make(map[string]*struct {
		ping   int64
		pOuter outputInterface
	})

	var checkOutTime int64
	for {
		currTime := time.Now().Unix()
		//printSyslog(fmt.Sprintf("tplRun--->push %s", tpl.tpl.Type))
		msgInfo, err := tpl.mq.Pop()
		//printSyslog(fmt.Sprintf("tplRun--->pop type:%s,len:%d", tpl.tpl.Type, tpl.mq.Len()))
		if err != nil {
			tpl.lock.Lock()
			tpl.cond.Wait()
			tpl.lock.Unlock()
		} else {
			tplMsg, ok := msgInfo.(*tplMessage)
			if !ok {
				printSyslog("msgInfo change *tplMessage error")
				continue
			}

			outInfo, ok := linkList[tplMsg.log.logbase.logName]
			var tplOutputInstance outputInterface
			if !ok {
				tplOutputInstance = tpl.createOutput(outputSetinfo{
					logbase: tplMsg.log.logbase,
					conf:    tpl.tpl.Path,
				})
				if tplOutputInstance == nil {
					printSyslog(fmt.Sprintf("createOutput err!<type:%s,path:%s>", tpl.tpl.Type, tpl.tpl.Path))
					continue
				}

				outInfo = &struct {
					ping   int64
					pOuter outputInterface
				}{
					ping:   currTime,
					pOuter: tplOutputInstance,
				}

				linkList[tplMsg.log.logbase.logName] = outInfo
			} else {

				tplOutputInstance = outInfo.pOuter
			}

			err := tplOutputInstance.checkOutput()
			if err != nil {
				printSyslog(err.Error())
				continue
			}
			tplOutputInstance.writeMsg(tplMsg.msg)
			outInfo.ping = currTime
		}

		// 30秒一次 判断超时
		if checkOutTime < currTime-overTime {
			//fmt.Println("checkOutTime logname", checkOutTime, currTime-overTime, ", linkList", len(linkList))
			for key, outInfo := range linkList {
				// 超时了 就删除
				//printSyslog(fmt.Sprintf("checkOutTime ping:%d,vo:%d", outInfo.ping, currTime-overTime))
				if outInfo.ping < currTime-overTime {
					outInfo.pOuter.releaseOutput()
					delete(linkList, key)
					printSyslog(fmt.Sprintf("delete linkList key:%s", key))
				}
			}
			checkOutTime = currTime
		}
	}
}

////

func logPart_time(b *[]byte, tpl *tplInfo) {
	s := getTimeString(time.Now())
	*b = append(*b, s...)
}

func logPart_data(b *[]byte, tpl *tplInfo) {
	*b = append(*b, tpl.currLog.msg...)
}

func logPart_line(b *[]byte, tpl *tplInfo) {
	*b = append(*b, "\n"...)
}
