package main

import (
    "runtime"
    "runtime/debug"
    "net/http"
    _ "net/http/pprof"
    "log"
    "time"
    "./chat"
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

    server := chat.NewChatServer("0.0.0.0:9999")
    server.ListenAndServe()
}
