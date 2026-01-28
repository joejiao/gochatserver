package chat

import (
	"bufio"
	"errors"
	"io"
	"log"
	"net"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	passwd = "pw"
)

type Client struct {
	sync.RWMutex
	server   *ChatServer
	uid      string
	roomName string
	conn     net.Conn
	rooms    map[string]*Room
	incoming chan *Message
	quiting  chan struct{}
	reader   *bufio.Reader
	writer   *bufio.Writer
}

func NewClient(conn net.Conn, server *ChatServer) *Client {
	client := &Client{
		server:   server,
		uid:      "-1",
		roomName: "",
		conn:     conn,
		rooms:    make(map[string]*Room),
		incoming: make(chan *Message),
		quiting:  make(chan struct{}),
		reader:   bufio.NewReaderSize(conn, 1024),
		writer:   bufio.NewWriterSize(conn, 1024),
	}
	return client
}

func (self *Client) handler() {
	if b, err := self.auth(); !b {
		log.Println("auth failed:", err)
		self.quit()
		return
	}
	if err := self.getUserId(); err != nil {
		log.Println("client.getUserId:", err)
		self.quit()
		return
	}
	if err := self.getRoomName(); err != nil {
		log.Println("client.getRoomName:", err)
		self.quit()
		return
	}
	//log.Println("get room name:", self.roomName)

	self.join()

	go self.listen()
	go self.read()
	//go self.write()
	go self.getFromRingBuffer()
}

func (self *Client) listen() {
	for {
		select {
		case msg, ok := <-self.incoming:
			//log.Printf("recive msg: %+v\n", msg)
			if !ok {
				return
			}

			self.RLock()
			room := self.rooms[self.roomName]
			self.RUnlock()

			room.incoming <- msg
			// 传递close信号: client -> room
		case _, ok := <-self.quiting:
			if ok {
				self.quit()
			}
			return
		}
		//runtime.Gosched()
	}
}

func (self *Client) join() {
	room := self.server.GetRoom(self.roomName)

	self.Lock()
	self.rooms[self.roomName] = room
	self.Unlock()

	room.addClient(self.conn, self)
	//room.Lock()
	//room.clients[self.conn] = self
	//room.Unlock()
}

func (self *Client) read() {
	defer func() {
		// recover from panic caused by writing to a closed channel
		if r := recover(); r != nil {
			log.Printf("runtime panic client.read: %v\n", r)
		}
	}()

	//br := bufio.NewReaderSize(self.conn, 1024)

	filter := self.server.filter
	for {
		//self.setDeadLine()
		line, err := self.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				log.Printf("Remote Closed: %s\n", self.conn.RemoteAddr().String())
			} else {
				log.Printf("ReadString error: %s\n", err)
			}
			//msg = &Message{cmd: "QUIT", data: "", receiver: receiver}
			self.quiting <- struct{}{}
			return
		}
		line = strings.TrimRight(line, "\n")

		msg := &Message{Data: line, Receiver: self.roomName}

		if filter.IsBlocked(self.uid, self.roomName) == false {
			self.incoming <- msg
		}
		//runtime.Gosched()
	}
}

func (self *Client) getFromRingBuffer() {
	self.RLock()
	room := self.rooms[self.roomName]
	self.RUnlock()

	rb := room.ringBuffer
	pos := rb.producerSequence.get()

	consumer := NewConsumer(rb)
	consumer.sequence.set(pos)

	isClosed := false
	for {
		select {
		case <-self.quiting:
			return

		default:
			if isClosed {
				//log.Println("client.writeMsg isClosed")
				return
			}

			l := consumer.Len()
			if l == 0 {
				time.Sleep(time.Second * 1)
				continue
			}

			items, err := consumer.BatchGet()
			if err != nil {
				log.Println("consumer.batchGet:", err)
				continue
			}
			for _, v := range items {
				//isClosed = self.writeMsg(v.(string))
				if self.writeMsg(v.(string)) == false {
					isClosed = true
				}
			}
			self.writer.Flush()
		}
	}
}

/*
func (self *Client) write() {
    //ticker := time.NewTicker(time.Second * 1)       //定时Flush,减少系统调用
    //ticker := time.NewTimer(time.Second * 1)

    defer func() {
        //ticker.Stop()
        self.writer.Flush()       //对应于self.writer
    }()

    isClosed := false

    for msgData := range self.outgoing {
        if isClosed {
            //log.Println("client.writeMsg isClosed")
            continue
        }
        isClosed = self.writeMsg(msgData)

        l := len(self.outgoing)
        if l == 0 {
            self.writer.Flush()
            time.Sleep(time.Second * 1)
            continue
        }

        for n := l; n > 0; n-- {
            msgData, ok := <-self.outgoing
            if !ok {
                return
            }
            isClosed = self.writeMsg(msgData)
        }
        self.writer.Flush()
    }
}
*/

func (self *Client) writeMsg(msgData string) bool {
	msgData = msgData + "\n"
	//data := msg.data
	//self.conn.SetWriteDeadline(time.Now().Add(1 * time.Second))
	_, err := self.writer.WriteString(msgData)
	// _, err := self.conn.Write([]byte(msgData))

	//如果写失败，返回false
	if nerr, ok := err.(net.Error); ok && nerr.Temporary() {
		return true
	}
	if err != nil {
		//log.Printf("client.conn.Write error: %s\n", err)
		return false
	}
	return true
}

func (self *Client) auth() (m bool, err error) {
	var line string

	line, err = self.reader.ReadString('\n')
	if err != nil {
		m = false
		return
	}
	line = strings.TrimRight(line, "\n")

	re := "^auth " + passwd + "$"
	m, err = regexp.MatchString(re, line)
	return
}

func (self *Client) getRoomName() (err error) {
	var line string

	line, err = self.reader.ReadString('\n')
	if err != nil {
		log.Println("get room name faild:", line, err)
		return
	}
	line = strings.TrimRight(line, "\n")

	re, _ := regexp.Compile("^join (.+)$")
	matches := re.FindStringSubmatch(line)
	//log.Printf("room matches: %s, %#v", len(matches), matches)

	if len(matches) < 2 {
		err = errors.New("not room name: " + line)
	} else {
		err = nil
		self.roomName = matches[1]
	}
	return
}

func (self *Client) getUserId() (err error) {
	var line string

	line, err = self.reader.ReadString('\n')
	if err != nil {
		log.Println("get userId faild:", line, err)
		return
	}
	line = strings.TrimRight(line, "\n")

	re, _ := regexp.Compile("^uid (.+)$")
	matches := re.FindStringSubmatch(line)
	//log.Printf("room matches: %s, %#v", len(matches), matches)

	if len(matches) < 2 {
		err = errors.New("not userId: " + line)
	} else {
		err = nil
		self.uid = matches[1]
	}
	return
}

func (self *Client) setDeadLine() {
	self.conn.SetDeadline(time.Now().Add(10 * time.Second))
}

func (self *Client) flush() {
	ticker := time.NewTicker(time.Second * 1)
	for _ = range ticker.C {
		//log.Printf("ticked at %s\n", time.Now())
		self.writer.Flush()
	}
}

func (self *Client) quit() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("runtime panic: client.quit: %v\n", r)
		}
	}()

	for _, room := range self.rooms {
		room.delClient(self.conn)
	}

	//close(self.outgoing)
	close(self.quiting)
	close(self.incoming)

	log.Println("close client channel: incoming, outgoing, quiting:", self.conn.RemoteAddr().String())
	self.conn.Close()
	//self = nil
}
