package chat

import (
    "log"
    "net"
    "sync"
    "time"
)

type ChatServer struct {
    sync.RWMutex
    bind_to string
    rooms   map[string]*Room
}

func NewChatServer(bind_to string) *ChatServer {
    rooms :=  make(map[string]*Room)
    return &ChatServer{bind_to: bind_to, rooms: rooms}
}

// GetRoom return a room, if this name of room is not exist,
// create a new room and return.
func (self *ChatServer) GetRoom(name string) *Room {
    self.RLock()
    _, ok := self.rooms[name]
    self.RUnlock()

    if !ok {
        room := NewRoom(name)

        self.Lock()
        self.rooms[name] = room
        self.Unlock()

        go room.listen()
    }

    return self.rooms[name]
}

// This method maybe should add a lock.
func (self *ChatServer) reportStatus() {
    ticker := time.NewTicker(time.Second * 10)
    defer ticker.Stop()

    for _ = range ticker.C {
        self.RLock()
        for _, room := range self.rooms {
            log.Printf("Status: %s:%d\n", room.name, len(room.clients))
        }
        self.RUnlock()
    }
}

func (self *ChatServer) clearRoom() {
    ticker := time.NewTicker(time.Second * 120)
    defer ticker.Stop()

    for _ = range ticker.C {
        self.RLock()
        for _, room := range self.rooms {
            if len(room.clients) == 0 {
                delete(self.rooms, room.name)
                room.quit()
            }
        }
        self.RUnlock()
    }
}

func (self *ChatServer) ListenAndServe() {
    tcpAddr, err := net.ResolveTCPAddr("tcp", self.bind_to)
    if err != nil {
        log.Println("ResolveTCPAddr error: " + err.Error())
        return
    }
    listener, err := net.ListenTCP("tcp", tcpAddr)
    if err != nil {
        log.Fatal("listenTCP error: " + err.Error())
        return
    }
    defer listener.Close()

    go self.reportStatus()
    go self.clearRoom()

    // Main loop
    for {
        conn, err := listener.AcceptTCP()
        if err != nil {
            log.Printf("Accept error: %s\n", err.Error())
            continue
        }

        conn.SetLinger(0)
        conn.SetNoDelay(false)
        conn.SetKeepAlive(true)
        conn.SetKeepAlivePeriod(120 * time.Second)

        client := NewClient(conn, self)
        go client.handler()
    }
}
