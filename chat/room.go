package chat

import (
	"log"
	"net"
	"sync"
)

const (
	ringBufferMaxSize = 512
)

type Room struct {
	sync.RWMutex
	name       string
	clients    map[net.Conn]*Client
	incoming   chan *Message
	outgoing   chan *Message
	quiting    chan struct{}
	ringBuffer *RingBuffer
	server     *ChatServer
}

func NewRoom(name string, server *ChatServer) *Room {
	room := &Room{
		name:       name,
		clients:    make(map[net.Conn]*Client),
		incoming:   make(chan *Message),
		outgoing:   make(chan *Message, 1000),
		quiting:    make(chan struct{}),
		ringBuffer: NewRingBuffer(ringBufferMaxSize),
		server:     server,
	}
	return room
}

// 每一个room有一个goro负责路由
func (self *Room) listen() {
	go self.writeToNATS()

	go self.readFromNATS()

	for msg := range self.outgoing {
		//log.Printf("Received: %+v\n", msg)
		//self.broadcast(msgData)
		self.writeToRingBuffer(msg)
	}
}

func (self *Room) addClient(conn net.Conn, client *Client) {
	self.Lock()
	self.clients[conn] = client
	self.Unlock()
}
func (self *Room) delClient(conn net.Conn) {
	self.Lock()
	delete(self.clients, conn)
	self.Unlock()
	//log.Printf("delete and close conn: %s\n", conn.RemoteAddr().String())
}

/*
func (self *Room) writeToFilterNATS() {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("runtime panic: room.writeToNATS: %s\n", r)
        }
    }()

    topic := self.server.opts.FilterTopic
    nc, _  := nats.Connect(self.server.opts.FilterQueue)
    ec, err := nats.NewEncodedConn(nc, nats.JSON_ENCODER)
    if err != nil {
        log.Fatal(err)
    }
    defer ec.Close()

    for msg := range self.incoming {
        //log.Printf("writeToNATS: %+v\n", msg)
        ec.Publish(topic, msg)
    }
}
*/

func (self *Room) writeToNATS() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("runtime panic: room.writeToNATS: %s\n", r)
		}
	}()

	poolConn := self.server.GetNATSConnection()
	if poolConn == nil {
		log.Printf("Failed to get NATS connection from pool for room %s\n", self.name)
		return
	}

	ec, err := poolConn.GetEncodedConn()
	if err != nil {
		log.Printf("Failed to get encoded connection: %v\n", err)
		return
	}
	defer ec.Close()

	for msg := range self.incoming {
		ec.Publish(self.name, msg)
	}
}

func (self *Room) readFromNATS() {
	poolConn := self.server.GetNATSConnection()
	if poolConn == nil {
		log.Printf("Failed to get NATS connection from pool for room %s\n", self.name)
		return
	}

	ec, err := poolConn.GetEncodedConn()
	if err != nil {
		log.Printf("Failed to get encoded connection for room %s: %v\n", self.name, err)
		return
	}

	defer func() {
		if r := recover(); r != nil {
			log.Printf("readFromNATS runtime panic: %+v\n", r)
		}
		ec.Close()
	}()

	ec.Subscribe(self.name, func(msg *Message) {
		self.outgoing <- msg
	})

	<-self.quiting
}

func (self *Room) writeToRingBuffer(msg *Message) {
	self.ringBuffer.Put(msg.Data)
}

/*
func (self *Room) broadcast(msgData string) {
    //timeout := time.Second * 2
    //tw := time.NewTimer(timeout)

    defer func() {
        if r := recover(); r != nil {
            log.Printf("runtime panic: send on closed channel client.outgoing: %v\n", r)
        }
        //tw.Stop()
    }()

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
*/

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
	//self = nil
}
