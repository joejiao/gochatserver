package main

import (
    "github.com/nats-io/nats"
    "strconv"
    "log"
    "sync"
)

var (
    serverAddr = "nats://10.1.64.2:4222"
    topic = "topic.test"
    sendCh = make(chan string)
    recvCh = make(chan string)
)

func main() {
    //go startConsumer1()
    //startProducer()
    go consumerChan()
    producerChan()
}

func startChan() {
}

func producerChan() {
    nc, _ := nats.Connect(serverAddr)
    ec, err := nats.NewEncodedConn(nc, "json")
    if err != nil {
        log.Fatal(err)
    }
    defer ec.Close()

    ec.BindSendChan(topic, sendCh)

    i := 0
    for {
        msg := "test message " + strconv.Itoa(i)
        sendCh <- msg
        i++
    }
}

func consumerChan() {
    nc, _ := nats.Connect(serverAddr)
    ec, err := nats.NewEncodedConn(nc, "json")
    if err != nil {
        log.Fatal(err)
    }
    defer ec.Close()

    ec.BindRecvChan(topic, recvCh)

    for m := range recvCh {
        log.Printf("Received on [%s]: '%s'\n", topic, m)
    }
}

func startProducer() {
    nc, err  := nats.Connect(serverAddr)
    if err != nil {
        log.Fatal(err)
    }
    defer nc.Close()

    i := 0
    for {
        msg := "test message " + strconv.Itoa(i)
        nc.Publish(topic, []byte(msg))
        nc.Flush()
        if err := nc.LastError(); err != nil {
            log.Fatal(err)
        }
        i++
    }
}

func startConsumer1() {
    nc, err := nats.Connect(serverAddr)
    if err != nil {
        log.Fatalf("Can't econnect: %v\n", err)
    }
    defer nc.Close()

    ch := make(chan *nats.Msg, 64)

    sub, err := nc.ChanSubscribe(topic, ch)
    if err := nc.LastError(); err != nil {
        log.Fatal(err)
    }

    for m := range ch {
        log.Printf("Received on [%s]: '%s'\n", m.Subject, string(m.Data))
    }

    sub.Unsubscribe()
    // Block forever
    //select {} 
    //wg := sync.WaitGroup{}
    //wg.Add(1)
    //wg.Wait()
}

func startConsumer() {
    nc, err  := nats.Connect(serverAddr)
    if err != nil {
        log.Fatalf("Can't connect: %v\n", err)
    }
    defer nc.Close()

    nc.Subscribe(topic, func(m *nats.Msg) {
        log.Printf("Received on [%s]: '%s'\n", m.Subject, string(m.Data))
    })
    if err := nc.LastError(); err != nil {
        log.Fatal(err)
    }
    // Block forever
    //select {} 
    wg := sync.WaitGroup{}
    wg.Add(1)
    wg.Wait()
}
