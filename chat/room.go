package chat

import (
    "sync"
    "net"
    "log"

    "github.com/nats-io/nats"
)

var (
    serverAddr = "nats://10.1.64.2:4222"
)

type Room struct {
    sync.RWMutex
    clients     map[*net.TCPConn]*Client
    name        string
    incoming    chan *Message
    outgoing    chan string
    quiting     chan *net.TCPConn
    nsqSendChan chan string
    nsqRecvChan chan string
}

func NewRoom(name string) *Room {
    room := &Room{
        name:       name,
        clients:    make(map[*net.TCPConn]*Client),
        incoming:   make(chan *Message),
        quiting:    make(chan *net.TCPConn),
        outgoing:   make(chan string),
        nsqSendChan: make(chan string),
        nsqRecvChan: make(chan string),
    }
    return room
}

// 每一个room有一个goro负责路由
func (self *Room) listen() {
    go self.writeToNATS()
    go self.readFromNATS()

    for {
        select {
        case msgData, ok := <-self.outgoing:
            if !ok {
                return
            }
            //log.Printf("Received on [%s]: '%s'\n", m.Subject, string(m.Data))
            self.broadcast(msgData)
        case conn, ok := <-self.quiting:
            if !ok {
                return
            }
            self.delClient(conn)
        }
        //runtime.Gosched()
    }
}

func (self *Room) addClient(conn *net.TCPConn, client *Client) {
    self.Lock()
    self.clients[conn] = client
    self.Unlock()
}
func (self *Room) delClient(conn *net.TCPConn) {
    self.Lock()
    delete(self.clients, conn)
    self.Unlock()
    //log.Printf("delete and close conn: %s\n", conn.RemoteAddr().String())
}

func (self *Room) writeToNATS() {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("runtime panic: room.writeToNATS: %s\n", r)
        }
    }()

    nc, _  := nats.Connect(serverAddr)
    ec, err := nats.NewEncodedConn(nc, "json")
    if err != nil {
        log.Fatal(err)
    }
    defer ec.Close()

    ec.BindSendChan(self.name, self.nsqSendChan)

    for msg := range self.incoming {
        self.nsqSendChan <- msg.data
    }
}

func (self *Room) readFromNATS() {
    nc, _ := nats.Connect(serverAddr)
    ec, err := nats.NewEncodedConn(nc, "json")
    if err != nil {
        log.Fatalf("Can't connect: %v\n", err)
    }
    defer ec.Close()

    ec.BindRecvChan(self.name, self.nsqRecvChan)

    for msgData := range self.nsqRecvChan {
       self.outgoing <- msgData
    }
}

func (self *Room) broadcast(msgData string) {
    //timeout := time.Second * 2
    //tw := time.NewTimer(timeout)

    defer func() {
        if r := recover(); r != nil {
            log.Printf("runtime panic: send on closed channel client.outgoing: %v\n", r)
        }
        //tw.Stop()
    }()

    /*
    // 复制到新的map防止lock冲突 
    newMap := make(map[*net.TCPConn]*Client)
    self.RLock()
    for k, v := range self.clients {
        newMap[k] = v
    }
    self.RUnlock()
    */

    self.RLock()
    for _, client := range self.clients {
        // 防止写入超时
        //self.timer.Reset(timeout)
        // A channel-based ring buffer solution
        select {
        //case <- self.timer.C:
        case client.outgoing <- msgData:
        default:
            l := int(len(client.outgoing) / 3)
            for i := 0; i < l; i++ {
                <-client.outgoing
            }
            client.outgoing <- msgData
            log.Printf("client.outgoing buffer is full, drop %d items, client ip: %s\n", l, client.conn.RemoteAddr().String())
        }
        //runtime.Gosched()
    }
    self.RUnlock()
}

func (self *Room) quit() {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("runtime panic: room.quit: %v\n", r)
        }
    }()

    log.Printf("close room %s:%d\n", self.name, len(self.clients))
    close(self.quiting)
    close(self.incoming)
    close(self.outgoing)
    //close(self.closeNSQ)
    close(self.nsqRecvChan)
    close(self.nsqSendChan)
    //self = nil
}
