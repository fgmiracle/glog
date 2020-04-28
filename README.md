# glog
rsyslog日志服务的简易版,支持tcp/udp/unixGram/unixDomain通信

# 服务程序
```golang
	var path string
	flag.StringVar(&path, "c", "parse args err", "config path")
	flag.Parse()

	_, err := os.Stat(path)
	if err != nil {
		fmt.Println("config not exist", path)
		return
	}

	glog.StartServer(path)

```

# 客户端
客户端可以根据协议自定义 不限语音
如golang 
```golang
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

```
