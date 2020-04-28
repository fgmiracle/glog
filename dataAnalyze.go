package glog

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

const (
	streamHeadSize = 2
	msgNameSize    = 2
	registerSize   = 2
	packHeadSize   = 6
	maxPackSize    = 128 * 1024
	reservedSize   = 8
)

//<uint32>logName + tplName + msg
// uint32:
// 0~4  	日志等级
// 4~12		日志名称
// 12~20	模板名称
// 20 ~ 32	预留
func analyseMsg(body []byte, size int) (level uint8, logName, tplName string, msg []byte, err error) {
	if size < packHeadSize {
		err = errors.New("packHeadSize too small")
		return
	}

	if body[0] != '<' || body[5] != '>' {
		err = errors.New("Reserved format is not correct")
		return
	}

	logType := binary.LittleEndian.Uint32(body[1:5])
	level = uint8(logType & 0xf)
	logNameLen := int(logType >> 4 & 0xff)
	templateLen := int(logType >> 12 & 0xffff)
	if size < (packHeadSize + templateLen + logNameLen) {
		fmt.Println("size:", size, "logType:", logType, "level:", level, "packHeadSize:", packHeadSize, "templateLen:", templateLen, "logNameLen:", logNameLen)
		err = errors.New(fmt.Sprintln("msg size too small,less than packHeadSize + templateLen + logNameLen!msgSize:%d ", size))
		return
	}

	var pos = packHeadSize
	logName = string(body[pos : pos+logNameLen])
	pos += logNameLen
	tplName = string(body[pos : pos+templateLen])
	pos += templateLen

	msg = body[pos:size]

	return
}

func packMsg(msg []byte, level Level, logName, tplName string) (body []byte, err error) {
	if msg == nil || logName == "" || tplName == "" {
		err = errors.New("pack msg param error")
		return
	}

	msgLen := len(msg) + len(logName) + len(tplName) + 6
	if msgLen > math.MaxUint16 {
		err = errors.New("pack msg error,msg too long")
		return
	}

	body = make([]byte, streamHeadSize)
	binary.LittleEndian.PutUint16(body, uint16(msgLen))
	body = append(body, '<')

	uLenBuf := make([]byte, 4)
	uLen := (uint32(len(tplName)) << 12) | (uint32(len(logName)) << 4) | (uint32(level) & 0xf)
	binary.LittleEndian.PutUint32(uLenBuf, uLen)
	body = append(body, uLenBuf...)
	body = append(body, '>')
	body = append(body, logName...)
	body = append(body, tplName...)
	body = append(body, msg...)

	return
}
