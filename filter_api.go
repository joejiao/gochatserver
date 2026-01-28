package chat1

import (
    "log"
    "github.com/nats-io/nats"
)

type Filter struct {
    clusterQueue string
    filterQueue string
    filterTopic string
    SrcChan      chan *Message
    DstChan     chan *Message
    exitChan    chan struct{}
}

func NewFilter(clusterQueue, filterQueue, filterTopic string) *Filter {
    filter := &Filter{
        clusterQueue:   clusterQueue,
        filterQueue:    filterQueue,
        filterTopic:    filterTopic,
        SrcChan:        make(chan *Message, 100),
        DstChan:        make(chan *Message, 100),
        exitChan:       make(chan struct{}),
    }
    return filter
}

func (self *Filter) StartAndServe() {
    go self.readFromFilterNATS()
    go self.writeToClusterNATS()
}

func (self *Filter) readFromFilterNATS() {
    nc, _ := nats.Connect(self.filterQueue)
    ec, err := nats.NewEncodedConn(nc, nats.JSON_ENCODER)
    if err != nil {
        log.Fatalf("nat.NewEncodedConn error: %v\n", err)
    }
    defer ec.Close()

    // 订阅主题, 当收到subject时候执行后面的func函数
    ec.Subscribe(self.filterTopic, func(msg *Message) {
        //log.Printf("readFromNATS: %+v\n", msg)
        self.SrcChan <- msg
    })

    <-self.exitChan
}

func (self *Filter) writeToClusterNATS() {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("runtime panic: room.writeToNATS: %s\n", r)
        }
    }()

    nc, _  := nats.Connect(self.clusterQueue)
    ec, err := nats.NewEncodedConn(nc, nats.JSON_ENCODER)
    if err != nil {
        log.Fatal(err)
    }
    defer ec.Close()

    for msg := range self.DstChan {
        //log.Printf("writeToNATS: %+v\n", msg)
        ec.Publish(msg.Receiver, msg)
    }
}

func (self *Filter) Quit() {
    close(self.exitChan)
    close(self.DstChan)
    close(self.SrcChan)
}
