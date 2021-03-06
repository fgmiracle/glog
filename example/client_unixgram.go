package main

import (
	"fmt"
	"time"

	"github.com/fgmiracle/glog"
)

var addr = "/tmp/glog.gram"

func sendLog(runID int) {
	glogClinet := glog.NewClinet("unixgram", addr)
	var index int
	glogClinet.Start()
	for {
		index++
		msg := fmt.Sprintf("rundID:%d,msgindex:%d! unixgram unixgram unixgram unixgram unixgram!!!!!!!", runID, index)
		tplName := "PRINTLOG"
		if runID%2 == 0 {
			tplName = "BILLLOG"
		}
		glogClinet.WriteMsg([]byte(msg), 3, fmt.Sprintf("runID_%d", runID), tplName)
		time.Sleep(time.Millisecond * 50)
	}

	glogClinet.Close()
}

func main() {

	close := make(chan int, 1)
	// for i := 0; i < 10; i++ {
	// 	go sendLog(i)
	// }

	glogClinet := glog.NewClinet("unixgram", addr)
	glogClinet.Start()

	msg := fmt.Sprintf("unixgram unixgram unixgram unixgram unixgram!!!!!!!")
	glogClinet.WriteMsg([]byte(msg), 3, fmt.Sprintf("runID_%d", 1), "PRINTLOG")

	<-close
}
