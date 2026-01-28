package main

import (
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"time"

	"gochatserver/chat"
)

var (
	natsUrl   = flag.String("nats_url", "nats://10.1.64.2:4222", "Cluster Gnats URL")
	filterDir = flag.String("filter_dir", "./filter", "Msg Filter File Dir")
	listen    = flag.String("listen", "0.0.0.0:9999", "Server Listen Address:Port")
)

func status() {
	var stats runtime.MemStats

	for {
		//debug.FreeOSMemory() //强制进行垃圾回收
		runtime.ReadMemStats(&stats)
		log.Printf("HeapAlloc %d HeapSys %d HeapRelease %d Goroutines %d\n",
			stats.HeapAlloc, stats.HeapSys, stats.HeapReleased, runtime.NumGoroutine())
		time.Sleep(10 * time.Second)
	}
}

func main() {
	flag.Parse()

	//go status()

	/*
	   //f, err := os.OpenFile("./tmp/cpu.prof", os.O_RDWR|os.O_CREATE, 0644)
	   f, err := os.Create("./tmp/cpu.prof")
	   if err != nil {
	       log.Fatal(err)
	   }
	   defer f.Close()
	   pprof.StartCPUProfile(f)
	   defer pprof.StopCPUProfile()
	*/
	go func() {
		http.ListenAndServe("0.0.0.0:3339", nil)
	}()

	opts := &chat.Options{NatsUrl: *natsUrl, FilterDir: *filterDir, Listen: *listen}
	server := chat.NewChatServer(opts)
	server.ListenAndServe()
}
