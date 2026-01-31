package chat

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"sync"
	"time"
)

const (
	// Fallback self-signed certificate for development
	defaultServerKey = `-----BEGIN EC PARAMETERS-----
BggqhkjOPQMBBw==
-----END EC PARAMETERS-----
-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIHg+g2unjA5BkDtXSN9ShN7kbPlbCcqcYdDu+QeV8XWuoAoGCCqGSM49
AwEHoUQDQgAEcZpodWh3SEs5Hh3rrEiu1LZOYSaNIWO34MgRxvqwz1FMpLxNlx0G
cSqrxhPubawptX5MSr02ft32kfOlYbaF5Q==
-----END EC PRIVATE KEY-----
`

	defaultServerCert = `-----BEGIN CERTIFICATE-----
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
)

type ChatServer struct {
	sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	rooms     map[string]*Room
	opts      *Options
	filter    *Filter
	natsPool  *NATSConnectionPool
}

func NewChatServer(opts *Options) *ChatServer {
	rooms := make(map[string]*Room)
	natsPool := NewNATSConnectionPool(opts.NatsUrl, 3)

	filter := NewFilter(opts)
	filter.StartAndServe()

	ctx, cancel := context.WithCancel(context.Background())

	server := &ChatServer{
		ctx:      ctx,
		cancel:   cancel,
		rooms:    rooms,
		opts:     opts,
		filter:   filter,
		natsPool: natsPool,
	}
	return server
}

func (self *ChatServer) GetNATSConnection() *NATSConnection {
	return self.natsPool.GetConnection()
}

// GetRoom return a room, if this name of room is not exist,
// create a new room and return.
func (self *ChatServer) GetRoom(name string) *Room {
	self.RLock()
	room, ok := self.rooms[name]
	self.RUnlock()

	if !ok {
		self.Lock()
		// Double-check: another goroutine might have created the room
		if room, ok = self.rooms[name]; ok {
			self.Unlock()
			return room
		}

		room = NewRoom(name, self)
		self.rooms[name] = room
		self.Unlock()

		go room.listen()
	}

	return room
}

func (self *ChatServer) DelRoom(name string) {
	self.Lock()
	delete(self.rooms, name)
	self.Unlock()
}

// This method maybe should add a lock.
func (self *ChatServer) reportStatus() {
	defer self.wg.Done()

	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()

	for {
		select {
		case <-self.ctx.Done():
			log.Println("reportStatus stopped")
			return
		case <-ticker.C:
			self.RLock()
			for _, room := range self.rooms {
				rb := room.ringBuffer
				pos := rb.producerSequence.get()
				log.Printf("Status: %s: online %d, pos %d\n", room.name, len(room.clients), pos)
			}
			self.RUnlock()
		}
	}
}

func (self *ChatServer) cleanRoom() {
	defer self.wg.Done()

	ticker := time.NewTicker(time.Second * 120)
	defer ticker.Stop()

	for {
		select {
		case <-self.ctx.Done():
			log.Println("cleanRoom stopped")
			return
		case <-ticker.C:
			self.Lock()
			for _, room := range self.rooms {
				if len(room.clients) == 0 {
					delete(self.rooms, room.name)
					room.quit()
				}
			}
			self.Unlock()
		}
	}
}

func (self *ChatServer) Close() {
	log.Println("Shutting down ChatServer...")
	self.natsPool.Close()
	log.Println("ChatServer shutdown complete")
}

// Shutdown gracefully shuts down the server
func (self *ChatServer) Shutdown() {
	log.Println("Starting graceful shutdown...")

	// Cancel context to stop all goroutines
	self.cancel()

	// Wait for background goroutines to finish
	self.wg.Wait()

	// Close all rooms
	self.Lock()
	for _, room := range self.rooms {
		room.quit()
	}
	self.rooms = make(map[string]*Room)
	self.Unlock()

	// Close NATS pool and filter
	self.natsPool.Close()
	log.Println("Graceful shutdown complete")
}

func (self *ChatServer) ListenAndServe() {
	tcpAddr, err := net.ResolveTCPAddr("tcp", self.opts.Listen)
	if err != nil {
		log.Println("ResolveTCPAddr error: " + err.Error())
		return
	}
	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		log.Printf("Failed to listen on %s: %v", self.opts.Listen, err)
		return
	}
	defer listener.Close()

	var cert tls.Certificate
	if self.opts.CertFile != "" && self.opts.KeyFile != "" {
		// Load certificate from files
		cert, err = tls.LoadX509KeyPair(self.opts.CertFile, self.opts.KeyFile)
		if err != nil {
			log.Printf("Failed to load TLS certificate from files: %v", err)
			return
		}
		log.Printf("Loaded TLS certificate from %s and %s", self.opts.CertFile, self.opts.KeyFile)
	} else {
		// Use default self-signed certificate for development
		cert, err = tls.X509KeyPair([]byte(defaultServerCert), []byte(defaultServerKey))
		if err != nil {
			log.Printf("Failed to load default TLS certificate: %v", err)
			return
		}
		log.Println("Using default self-signed TLS certificate (for development only)")
	}
	config := &tls.Config{Certificates: []tls.Certificate{cert}}

	ln := tls.NewListener(listener, config)
	defer ln.Close()

	// Start background goroutines with WaitGroup
	self.wg.Add(2)
	go self.reportStatus()
	go self.cleanRoom()

	// Main loop
	go func() {
		<-self.ctx.Done()
		ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			// Check if this is due to shutdown
			select {
			case <-self.ctx.Done():
				log.Println("Listener closed due to shutdown")
				return
			default:
				log.Printf("Accept error: %s\n", err.Error())
				continue
			}
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
