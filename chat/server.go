package chat

import (
    "log"
    "net"
    "sync"
    "time"
    "crypto/tls"
)

const serverKey = `-----BEGIN EC PARAMETERS-----
BggqhkjOPQMBBw==
-----END EC PARAMETERS-----
-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIHg+g2unjA5BkDtXSN9ShN7kbPlbCcqcYdDu+QeV8XWuoAoGCCqGSM49
AwEHoUQDQgAEcZpodWh3SEs5Hh3rrEiu1LZOYSaNIWO34MgRxvqwz1FMpLxNlx0G
cSqrxhPubawptX5MSr02ft32kfOlYbaF5Q==
-----END EC PRIVATE KEY-----
`

const serverCert = `-----BEGIN CERTIFICATE-----
MIIB+TCCAZ+gAwIBAgIJAL05LKXo6PrrMAoGCCqGSM49BAMCMFkxCzAJBgNVBAYT
AkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRlcm5ldCBXaWRn
aXRzIFB0eSBMdGQxEjAQBgNVBAMMCWxvY2FsaG9zdDAeFw0xNTEyMDgxNDAxMTNa
Fw0yNTEyMDUxNDAxMTNaMFkxCzAJBgNVBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0
YXRlMSEwHwYDVQQKDBhJbnRlcm5ldCBXaWRnaXRzIFB0eSBMdGQxEjAQBgNVBAMM
CWxvY2FsaG9zdDBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABHGaaHVod0hLOR4d
66xIrtS2TmEmjSFjt+DIEcb6sM9RTKS8TZcdBnEqq8YT7m2sKbV+TEq9Nn7d9pHz
pWG2heWjUDBOMB0GA1UdDgQWBBR0fqrecDJ44D/fiYJiOeBzfoqEijAfBgNVHSME
GDAWgBR0fqrecDJ44D/fiYJiOeBzfoqEijAMBgNVHRMEBTADAQH/MAoGCCqGSM49
BAMCA0gAMEUCIEKzVMF3JqjQjuM2rX7Rx8hancI5KJhwfeKu1xbyR7XaAiEA2UT7
1xOP035EcraRmWPe7tO0LpXgMxlh2VItpc2uc2w=
-----END CERTIFICATE-----
`

type ChatServer struct {
    sync.RWMutex
    rooms   map[string]*Room
    opts    *Options
    filter  *Filter
}

func NewChatServer(opts *Options) *ChatServer {
    rooms :=  make(map[string]*Room)

    filter := NewFilter()
    filter.StartAndServe()

    server := &ChatServer{rooms: rooms, filter: filter}
    return server
}

// GetRoom return a room, if this name of room is not exist,
// create a new room and return.
func (self *ChatServer) GetRoom(name string) *Room {
    self.RLock()
    _, ok := self.rooms[name]
    self.RUnlock()

    if !ok {
        room := NewRoom(name, self)

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
            rb := room.ringBuffer
            pos := rb.producerSequence.get()
            log.Printf("Status: %s: online %d, pos %d\n", room.name, len(room.clients), pos)
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
    tcpAddr, err := net.ResolveTCPAddr("tcp", self.opts.Listen)
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

    // use tls
    cert, err := tls.LoadX509KeyPair("keys/server.pem", "keys/server.key")
    //cert, err := tls.X509KeyPair([]byte(serverCert), []byte(serverKey))
    if err != nil {
        log.Fatal(err)
    }
    config := &tls.Config{Certificates: []tls.Certificate{cert}}

    ln := tls.NewListener(listener, config)
    defer ln.Close()

    go self.reportStatus()
    go self.clearRoom()

    // Main loop
    for {
        conn, err := ln.Accept()
        if err != nil {
            log.Printf("Accept error: %s\n", err.Error())
            continue
        }

        tcpConn, ok := conn.(*net.TCPConn)
        if ok {
            //tcpConn.SetLinger(0)
            tcpConn.SetNoDelay(false)
            tcpConn.SetKeepAlive(true)
            tcpConn.SetKeepAlivePeriod(120 * time.Second)
        }

        client := NewClient(conn, self)
        go client.handler()
    }
}
