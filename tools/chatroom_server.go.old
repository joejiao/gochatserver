package main

import (
	"bufio"
    "runtime"
    "runtime/debug"
    //"runtime/pprof"
    "net/http"
    _ "net/http/pprof"
    "time"
	"log"
	"net"
    "sync"
    "io"
    //"os"
)

const (
    MAXCLIENTS = 100000
)

type Message struct {
    cmd string
    content string
}

type Client struct {
    conn *net.TCPConn
    incoming chan *Message
    outgoing chan *Message
    reader   *bufio.Reader
    writer   *bufio.Writer
    quiting chan *net.TCPConn
}

type Room struct {
    lock   *sync.RWMutex
    clients map[*net.TCPConn]*Client
    incoming chan *Message
    //outgoing chan *Message
    pending chan *net.TCPConn
    quiting chan *net.TCPConn
}

func createClient(conn *net.TCPConn) *Client {
    conn.SetLinger(0)
    conn.SetNoDelay(false)
    conn.SetKeepAlive(true)
    conn.SetKeepAlivePeriod(60 * time.Second)

    reader := bufio.NewReader(conn)
    writer := bufio.NewWriter(conn)

    client := &Client{
        conn:     conn,
        incoming: make(chan *Message),
        outgoing: make(chan *Message),
        reader:   reader,
        writer:   writer,
        quiting:  make(chan *net.TCPConn),
    }
    return client
}

func (self *Client) listen() {
    go self.read()
    go self.write()
    //go self.flush()
}

func (self *Client) read() {
    defer func() {
        // recover from panic caused by writing to a closed channel
        if r := recover(); r != nil {
            log.Printf("runtime panic: send on closed channel client.incoming %v\n", r)
        }
    }()

    var msg *Message

    for {
        line, err := self.reader.ReadString('\n')
        if err != nil {
            if err == io.EOF {
                log.Printf("Remote Closed: %s\n", self.conn.RemoteAddr().String())
            } else {
                log.Printf("ReadString error: %s\n", err)
            }
            //msg = &Message{cmd: "QUIT", content: line}
            self.quit()
            return
        }

        msg = &Message{cmd: "MSG", content: line}
        self.incoming <- msg
    }

}

func (self *Client) write() {
    //defer self.writer.Flush()       //对应于self.writer

    isClosed := false
    for msg := range self.outgoing {
        if isClosed {
            continue
        }
        data := msg.content
        //_, err := self.writer.WriteString(data)
        _, err := self.conn.Write([]byte(data))
        if nerr, ok := err.(net.Error); ok && nerr.Temporary() {
            //time.Sleep(1e9)
            continue
        }
        if err != nil {
            isClosed = true
            log.Printf("client.conn.Write error: %s\n", err)
            //self.quit()
            //return
        }
        /*
        if err := self.writer.Flush(); err != nil {
            log.Printf("Write error: %s\n", err)
            self.quit()
            return
        }
        */
    }
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
        // recover from panic caused by writing to a closed channel
        if r := recover(); r != nil {
            log.Printf("runtime panic: send on closed channel client.quiting %v\n", r)
        }
    }()

    log.Printf("send %s to client.quiting\n", self.conn.RemoteAddr().String())
    self.quiting <- self.conn
}

// room class
func createRoom() *Room {
    room := &Room{
        lock:     new(sync.RWMutex),
        clients:  make(map[*net.TCPConn]*Client),
        incoming: make(chan *Message),
        pending:  make(chan *net.TCPConn),
        quiting:  make(chan *net.TCPConn),
    }
    return room
}

// 每一个room有一个goro负责路由
func (self *Room) listen() {
    go func() {
        for {
            select {
            case message := <-self.incoming:
                go self.broadcast(message)
            case conn := <-self.pending:
                self.join(conn)
            case conn := <-self.quiting:
                log.Printf("delete and close conn: %s\n", conn.RemoteAddr().String())
                self.lock.Lock()
                delete(self.clients, conn)
                self.lock.Unlock()
                conn.Close()
            }
        }
    }()
}

func (self *Room) start() {
    tcpAddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:9999")
    if err != nil {
        log.Println("ResolveTCPAddr error: " + err.Error())
        return
    }
	tcpListener, err := net.ListenTCP("tcp", tcpAddr)
    if err != nil {
        log.Println("listenTCP error: " + err.Error())
        return
    }
	defer tcpListener.Close()

    log.Printf("Chat Room %#v\n", self)

    for {
        tcpConn, err := tcpListener.AcceptTCP()
        if err != nil {
            log.Printf("accept failed: %s\n", err)
            continue
        }

        //log.Printf("A new connection %v kicks\n", tcpConn)
        self.pending <- tcpConn
    }
}

// 新client处理
func (self *Room) join(conn *net.TCPConn) {
    client := createClient(conn)
    client.listen()

    self.lock.Lock()
    self.clients[conn] = client
    self.lock.Unlock()

    go func() {
        for {
            select {
                // 传递msg, client -> room
            case msg, ok := <-client.incoming:
                if ok {
                    //log.Printf("Got message: %#v from client %s\n", msg, client.conn.RemoteAddr().String())
                    self.incoming <- msg
                } else {
                    log.Println("client.incoming closed:", client.conn.RemoteAddr().String())
                    return
                }
                // 传递close信号: client -> room
            case conn, ok := <-client.quiting:
                if ok {
                    self.quiting <- conn
                    close(client.incoming)
                    close(client.outgoing)
                    close(client.quiting)
                    self = nil
                    log.Println("close client channel: incoming, outgoing, quiting:", client.conn.RemoteAddr().String())
                } else {
                    log.Println("client.quiting closed", client.conn.RemoteAddr().String())
                }
                return
            }
        }
    }()
}

func (self *Room) broadcast(message *Message) {
    defer func() {
        // recover from panic caused by writing to a closed channel
        if r := recover(); r != nil {
            log.Printf("runtime panic: send on closed channel client.outgoing: %s\n", client.conn.RemoteAddr().String())
        }
    }()

    //log.Printf("Broadcasting message: %s\n", message)
    for _, client := range self.clients {
        client.outgoing <- message
    }
}

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
        (http.ListenAndServe("42.62.3.93:3339", nil))
    }()

    room := createRoom()
    room.listen()
    room.start()
}
