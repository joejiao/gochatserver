package main

import (
    "flag"
    "runtime"
    "runtime/debug"
    "net/http"
    _ "net/http/pprof"
    "log"
    "time"
    "./chat"
)

var (
    clusterQueue = flag.String("cluster_queue", "nats://10.1.64.2:4222", "Cluster Gnats URL")
    filterQueue = flag.String("filter_queue", "nats://127.0.0.1:4222", "Msg Filter Gnats URL")
    withFilter =flag.Bool("with_filter", false, "Use Msg Filter")
    listen = flag.String("listen", "0.0.0.0:9999", "Server Listen Address:Port")
)

func status() {
    var stats runtime.MemStats

    for {
        debug.FreeOSMemory()
        runtime.ReadMemStats(&stats)
        log.Printf("HeapAlloc %d HeapSys %d HeapRelease %d Goroutines %d\n",
        stats.HeapAlloc, stats.HeapSys, stats.HeapReleased, runtime.NumGoroutine())
        time.Sleep(5 * time.Second)
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

    opts := &chat.Options{ClusterQueue: *clusterQueue, FilterQueue: *filterQueue, WithFilter: *withFilter, Listen: *listen, FilterTopic: "origMsgQueue"}
    server := chat.NewChatServer(opts)
    server.ListenAndServe()
}
