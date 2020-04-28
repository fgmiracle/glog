package main

import (
	"fmt"
	"time"

	"github.com/fgmiracle/glog"
)

var addr = "0.0.0.0:4398"

func sendLog(runID int) {
	glogClinet := glog.NewClinet("tcp", addr)
	var index int
	glogClinet.Start()
	for {
		index++
		msg := fmt.Sprintf("rundID:%d,msgindex:%d! TCP TCP TCP TCP TCP TCP!!!!!!!")
		tplName := "PRINTLOG"
		if runID%2 == 0 {
			tplName = "BILLLOG"
		}
		glogClinet.WriteMsg([]byte(msg), 0, fmt.Sprintf("runID_%d", runID), tplName)
		time.Sleep(time.Millisecond * 50)
	}

	glogClinet.Close()
}

func main() {

	close := make(chan int, 1)
	for i := 0; i < 10; i++ {
		go sendLog(i)
	}

	// glogClinet := glog.NewClinet("tcp", addr)
	// glogClinet.Start()

	// msg := fmt.Sprintf("rundID:%d,msgindex:%d! TCP TCP TCP TCP TCP TCP!!!!!!!")
	// glogClinet.WriteMsg([]byte(msg), 0, fmt.Sprintf("runID_%d", 1), "BILLLOG")

	<-close
}
