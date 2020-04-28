package glog

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"syscall"
)

//StartServer ...
func StartServer(config string) {
	ok, err := parseConfig(config)
	if !ok {
		printSyslog(fmt.Sprintf("ParseConfig er: %s", err))
		os.Exit(1)
		return
	}

	tplInit()
	for netType, ld := range serverList {
		switch netType {
		case tplTcp:
			prot := ld.(int)
			if prot > 0 {
				addr := "0.0.0.0:" + strconv.Itoa(prot)
				go tcpServer(addr)
			}
		case tplUdp:
			prot := ld.(int)
			if prot > 0 {
				addr := "0.0.0.0:" + strconv.Itoa(prot)
				go udpServer(addr)
			}
		case tplUnixDomain:
			addr := ld.(string)
			if addr != "" {
				go unixDomainServer(addr)
			}
		case tplUnixGram:
			addr := ld.(string)
			if addr != "" {
				go unixGramServer(addr)
			}
		default:
			printSyslog(fmt.Sprintf("unkown netType :%s", netType))
		}
	}
}

func getRemoteIP(conn net.Conn) string {
	remoteAddr := conn.RemoteAddr()
	remoteIP, _, _ := net.SplitHostPort(remoteAddr.String())
	if remoteIP == "::1" {
		remoteIP = "127.0.0.1"
	}

	return remoteIP
}

func getIntranetIP() string {
	var IP = "127.0.0.1"
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		fmt.Println(err)
		return IP
	}

	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}

	return IP
}

func streamMsgRecv(conn net.Conn) {
	defer func() {
		if err := recover(); err != nil {
			printSyslog(fmt.Sprintf("caught an error:%s", err))
		}
	}()
	defer conn.Close()

	var headerBuffer = make([]byte, streamHeadSize)
	reader, ok := conn.(io.Reader)
	if !ok || reader == nil {
		printSyslog(fmt.Sprintf("conn chang io.reader err"))
		return
	}

	remoteIP := getRemoteIP(conn)

	for {
		// 读取头
		_, err := io.ReadFull(reader, headerBuffer)
		if err != nil {
			printSyslog(fmt.Sprintf("read message from header failed, error:<%s>", err))
			return
		}

		// 再次校验下
		if len(headerBuffer) < streamHeadSize {
			printSyslog(fmt.Sprintf("read header length er, error:<%s>", err))
			return
		}

		//包大小(小端)
		packSize := (int)(binary.LittleEndian.Uint16(headerBuffer))
		if packSize >= maxPackSize {
			printSyslog(fmt.Sprintf("pack  too big length <%d>", packSize))
			return
		}

		// 分配包体大小
		bodyBuffer := make([]byte, packSize)
		_, err = io.ReadAtLeast(reader, bodyBuffer, packSize)
		if err != nil {
			printSyslog(fmt.Sprintf("read msg body err <%s>", err))
			return
		}

		level, logName, tpl, msg, err := analyseMsg(bodyBuffer, packSize)
		// 分发消息
		if err != nil {
			printSyslog(err.Error())
			continue
		}

		log := &tplMessage{
			log: logInfo{
				logbase: logBase{
					logName:    logName,
					fromhostIP: remoteIP,
				},
				level: Level(level),
			},
			msg: msg,
		}
		// 消息的重新处理
		log.msg = tplMsgHandle(tpl, log, bodyBuffer)
		// 消息分发
		tplDispatch(tpl, logName, log)
	}
}

func tcpServer(addr string) {
	printSyslog(fmt.Sprintf("start tcp server addr(%s)", addr))
	tcpListerner, err := net.Listen("tcp", addr)
	if err != nil {
		printSyslog(fmt.Sprintf("failed to listen server (%s), error(%s)", addr, err))
		return
	}

	defer tcpListerner.Close()
	for {
		conn, err := tcpListerner.Accept()
		if err != nil {
			printSyslog(fmt.Sprintf("failed to accept the new connection,err(%s)", err))
			continue
		}

		go streamMsgRecv(conn)
	}
}

func unixDomainServer(addr string) {
	printSyslog(fmt.Sprintf("start unixdomain server addr(%s)", addr))
	os.Remove(addr)
	uaddr, err := net.ResolveUnixAddr("unix", addr)
	if err != nil {
		fmt.Println("unixDomainServer, resolve unix addr err->", err)
		return
	}

	unixListener, err := net.ListenUnix("unix", uaddr)
	if err != nil {
		fmt.Println("unixDomainServer ListenUnix err -->", err)
		return
	}

	defer unixListener.Close()

	for {
		uconn, err := unixListener.AcceptUnix()
		if err != nil {
			fmt.Println(err)
			continue
		}

		go streamMsgRecv(uconn)
	}

}

func udpServer(addr string) {
	printSyslog(fmt.Sprintf("start udp server addr(%s)", addr))
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		printSyslog(fmt.Sprintf("udpServer, resolve addr! err<%s>", err))
		return
	}
	udpListener, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		printSyslog(fmt.Sprintf("udpServer, err<%s>", err))
		return
	}

	isExit := false
	defer udpListener.Close()
	defer func() { isExit = true }()
	recvFun := func() {
		var buf = make([]byte, maxPackSize)
		for {
			if isExit {
				return
			}

			size, addr, err := udpListener.ReadFromUDP(buf)
			if err != nil {
				printSyslog(fmt.Sprintf("ReadFromUnix err<%s>", err))
				continue
			}

			// 包太大了
			if size >= maxPackSize-reservedSize {
				printSyslog("ReadFromUnix pack too big! recv ")
				continue
			}

			level, logName, tpl, msg, err := analyseMsg(buf[2:], size)
			// 分发消息
			if err != nil {
				printSyslog(err.Error())
				continue
			}

			log := &tplMessage{
				log: logInfo{
					logbase: logBase{
						logName:    logName,
						fromhostIP: addr.IP.String(),
					},
					level: Level(level),
				},
				msg: msg,
			}
			// 消息的重新处理
			log.msg = tplMsgHandle(tpl, log, buf[:size])
			// 消息分发
			tplDispatch(tpl, logName, log)
		}
	}

	for i := 0; i < svrWorker-1; i++ {
		go recvFun()
	}

	recvFun()
}

func unixGramServer(addr string) {
	printSyslog(fmt.Sprintf("start unixgram server addr(%s)", addr))
	os.Remove(addr)
	uaddr, err := net.ResolveUnixAddr("unixgram", addr)
	if err != nil {
		printSyslog(fmt.Sprintf("unixGramServer, resolve unix addr! err<%s>", err))
		return
	}
	syscall.Unlink(addr)
	unixListener, err := net.ListenUnixgram("unixgram", uaddr)
	if err != nil {
		printSyslog(fmt.Sprintf("ListenUnixgram, err<%s>", err))
		return
	}

	os.Chmod(addr, 0666)
	isExit := false
	defer unixListener.Close()
	defer func() { isExit = true }()
	localIP := getIntranetIP()

	recvFun := func() {
		var buf = make([]byte, maxPackSize)
		for {
			if isExit {
				return
			}

			size, _, err := unixListener.ReadFromUnix(buf)
			if err != nil {
				printSyslog(fmt.Sprintf("ReadFromUnix err<%s>", err))
				continue
			}

			// 包太大了
			if size >= maxPackSize-reservedSize {
				printSyslog("ReadFromUnix pack too big! recv ")
				continue
			}

			level, logName, tpl, msg, err := analyseMsg(buf[2:], size-2)
			// 分发消息
			if err != nil {
				printSyslog(err.Error())
				continue
			}

			log := &tplMessage{
				log: logInfo{
					logbase: logBase{
						logName:    logName,
						fromhostIP: localIP,
					},
					level: Level(level),
				},
				msg: msg,
			}
			// 消息的重新处理
			log.msg = tplMsgHandle(tpl, log, buf[:size])
			// 消息分发
			tplDispatch(tpl, logName, log)
		}
	}

	for i := 0; i < svrWorker-1; i++ {
		go recvFun()
	}

	recvFun()
}
