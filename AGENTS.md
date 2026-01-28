# Agent Guidelines for gochatserver

## Project Overview

This is a Go-based chat server using NATS message queue for distributed messaging. The codebase follows older Go conventions (GOPATH mode) with relative imports and vendor directory for dependencies.

## Build Commands

### Build the server
```bash
# GOPATH mode (required due to relative imports)
export GOPATH=/path/to/parent/GitLab
export GO111MODULE=off
go run chatserver.go -nats_url="nats://127.0.0.1:4222" -listen="0.0.0.0:9999" -filter_dir="./filter"

# Or build binary
go build -o chatserver chatserver.go
```

### Run with flags
```bash
./chatserver \
  -nats_url="nats://127.0.0.1:4222" \
  -listen="0.0.0.0:9999" \
  -filter_dir="./filter"
```

## Test Commands

### Run all tests
```bash
# Individual test files
go test -v ./test/ring_test.go
go test -v ./tools/nats_test.go

# Run benchmarks
go test -bench=. -benchmem ./test/
go test -bench=. -benchmem ./tools/
```

### Run a specific test
```bash
# Specific benchmark
go test -bench=BenchmarkRingBufferGet ./test/
go test -bench=BenchmarkNatsProduce ./tools/

# With verbose output
go test -v -bench=BenchmarkRingBufferPut ./test/
```

## Code Style Guidelines

### Import Organization
```go
// Order: standard library → third-party → local packages
import (
    "crypto/tls"
    "log"
    "net"
    "sync"
    "time"

    "github.com/nats-io/nats"

    "./chat"
)
```

### Struct and Type Definitions
```go
// Use sync.RWMutex embedded for thread-safe structures
type ChatServer struct {
    sync.RWMutex
    rooms  map[string]*Room
    opts   *Options
    filter *Filter
}

// Constructor pattern
func NewChatServer(opts *Options) *ChatServer {
    rooms := make(map[string]*Room)
    filter := NewFilter(opts)
    filter.StartAndServe()

    server := &ChatServer{
        rooms: rooms,
        opts: opts,
        filter: filter,
    }
    return server
}
```

### Method Receivers
```go
// Use "self" as receiver name (NOT "this" or "c")
func (self *ChatServer) GetRoom(name string) *Room {
    self.RLock()
    defer self.RUnlock()

    // Method implementation
    return room
}
```

### Concurrency Patterns
```go
// Use channels for goroutine communication
type Client struct {
    incoming    chan *Message
    quiting     chan struct{}
}

// Use RWMutex for read/write locking
func (self *Room) addClient(conn net.Conn, client *Client) {
    self.Lock()
    self.clients[conn] = client
    self.Unlock()
}

// Goroutine management with defer recover
func (self *Room) writeToNATS() {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("runtime panic: room.writeToNATS: %s\n", r)
        }
    }()
    // Implementation
}
```

### Error Handling
```go
// Fatal errors use log.Fatal
if err != nil {
    log.Fatal("error description: ", err)
}

// Recoverable errors use log.Println
if err != nil {
    log.Println("non-fatal error: ", err)
    return
}

// Type assertions with error checking
rid, ok := rid.(string)
if !ok {
    log.Printf("type assertion error: %+v\n", rid)
    return false
}
```

### Constants and Configuration
```go
const (
    ringBufferMaxSize = 512
    InitialSequenceValue int64 = -1
)

// Flag definitions in main
var (
    natsUrl   = flag.String("nats_url", "nats://10.1.64.2:4222", "Cluster Gnats URL")
    filterDir = flag.String("filter_dir", "./filter", "Msg Filter File Dir")
    listen    = flag.String("listen", "0.0.0.0:9999", "Server Listen Address:Port")
)
```

### File and Package Structure
- **Root**: `chatserver.go` (main entry point), `client.go`, `filter_api.go`
- **chat/**: Core package with `server.go`, `room.go`, `client.go`, `ring.go`, `message.go`, `options.go`, `filter.go`, `public_func.go`
- **test/**: Test files using `_test.go` suffix
- **tools/**: Utility programs and additional benchmarks
- **filter/**: Blacklist JSON configuration directory

### Important Constraints
- Project uses GOPATH mode with relative imports (`"./chat"`)
- Does NOT use Go modules (set `GO111MODULE=off`)
- Dependencies managed in `pkg/mod/` and `vendor/` directories
- Mixed English and Chinese comments in the codebase
- Preserves commented-out code blocks
- Uses hardcoded password "pw" for authentication (should be configurable)

### Testing Style
```go
func BenchmarkRingBufferGet(b *testing.B) {
    var size int64 = 1024
    rb := chat.NewRingBuffer(size)

    // Setup code
    b.ResetTimer()

    // Benchmark loop
    c1 := chat.NewConsumer(rb)
    for i := 0; i < b.N; i++ {
        _, err := c1.Get()
        if err != nil {
            // Handle error
        }
    }
}
```

### Logging
- Use `log.Printf()` for formatted logging
- Use `log.Println()` for simple logging
- Prefix context: `log.Printf("Status: %s: online %d\n", room.name, len(room.clients))`
- Include error details: `log.Printf("Accept error: %s\n", err.Error())`

### Message Format
- Messages use JSON encoding via NATS: `nats.JSON_ENCODER`
- Message struct: `{Sender, Receiver, Data string}` with JSON tags
- Protocol commands: `auth password`, `uid 1111`, `join roomId`

## Architecture Notes

The server follows a producer-consumer pattern with:
- Ring buffers for message queuing (disruptor pattern)
- Multiple NATS subscribers for distributed messaging
- Rooms as message routing units
- Filter/blacklist system for message control
- TLS encryption with embedded certificates

## Environment Setup

```bash
# Required environment variables (from .vscode/settings.json)
export GOPATH="/Users/jiaoshengqiang/GitLab/gochatserver"
export GO111MODULE="auto"
export GOFLAGS="-mod=vendor"
```

## Development Notes

- Codebase predates Go 1.11 modules
- Uses older Go patterns (before modules became standard)
- Heavy use of goroutines and channels
- Embedded mutex pattern for thread safety
- Ring buffer implementation based on LMAX Disruptor
