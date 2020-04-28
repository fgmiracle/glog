package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"net/http"
	_ "net/http/pprof"

	"github.com/fgmiracle/glog"
)

func main() {

	go func() {
		http.ListenAndServe("0.0.0.0:8899", nil)
	}()
	var path string
	flag.StringVar(&path, "c", "parse args err", "config path")
	flag.Parse()

	_, err := os.Stat(path)
	if err != nil {
		fmt.Println("config not exist", path)
		return
	}

	glog.StartServer(path)

	for {
		time.Sleep(time.Duration(2) * time.Second)
	}

}
