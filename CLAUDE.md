# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run Commands

```bash
# Build the server
go build -o bin/chatserver ./cmd/chatserver

# Run directly with flags
go run ./cmd/chatserver -nats_url="nats://127.0.0.1:4222" -listen="0.0.0.0:9999" -filter_dir="./filter"

# Run tests
go test ./test/benchmark/... -bench=. -benchmem
go test ./test/demo/...

# Stress test client (requires running server)
go run ./examples/client.go
```

**Flags:**
- `-nats_url`: NATS server URL (default: `nats://10.1.64.2:4222`)
- `-listen`: Server listen address (default: `0.0.0.0:9999`)
- `-filter_dir`: Blacklist config directory (default: `./filter`)

## High-Level Architecture

The server uses a distributed message queue architecture with NATS as the message bus:

```
ChatServer (manages rooms)
    ├── Room (per-chatroom router) [1:N]
    │   ├── Client (TCP connection) [1:N]
    │   │   └── 3 goroutines: listen(), read(), getFromRingBuffer()
    │   ├── RingBuffer (lock-free queue, LMAX Disruptor pattern)
    │   └── 2 goroutines: writeToNATS(), readFromNATS()
    ├── NATSConnectionPool (shared connections, default size: 3)
    └── Filter (blacklist/禁言 system)
```

**Message Flow:**
```
Client.incoming → Room.incoming → writeToNATS → NATS bus
                                              ↓
Room.outgoing ← readFromNATS ← ← ← ← ← ← ← ← ←
       ↓
RingBuffer.Put() → consumer.Get() → Client.writeMsg()
```

## Key Design Patterns

### 1. Embedded Mutex Pattern
All major structs embed `sync.RWMutex` for thread-safe operations:
```go
type ChatServer struct {
    sync.RWMutex
    rooms map[string]*Room
    // ...
}

// Usage: self.RLock()/self.RUnlock() for reads, self.Lock()/self.Unlock() for writes
```

### 2. Lock-Free Ring Buffer (chat/ring.go)
Based on LMAX Disruptor pattern with CAS operations:
- Size must be power of 2 (uses `pos & mask` instead of modulo)
- CPU cache-line padding to prevent false sharing
- Each Consumer has independent Sequence cursor

**Critical:** Buffer overflow **silently overwrites** old data (logs but doesn't block).

### 3. NATS Connection Pool (chat/nats_pool.go)
Refactored from per-Room connections to shared pool:
- Round-robin connection allocation
- Health check every 30 seconds
- Auto-reconnect (max 5 times, 2s interval)
- Call `GetEncodedConn()` to get connection, **always defer Close()`

### 4. Room Lifecycle (chat/room.go)
- **Creation:** Lazy initialization in `ChatServer.GetRoom()` (double-check locking pattern - has known race)
- **Cleanup:** `cleanRoom()` goroutine deletes empty rooms every 120s
- **Shutdown:** `room.quit()` closes channels: `quiting`, `incoming`, `outgoing`

### 5. Client Protocol (chat/client.go)
Connection handshake sequence:
```
1. auth <password>  (hardcoded: "pw")
2. uid <user_id>
3. join <room_name>
```

After auth: `Client.handler()` spawns 3 goroutines:
- `listen()`: multiplexes `incoming` and `quiting` channels
- `read()`: reads socket, applies filter, writes to `incoming`
- `getFromRingBuffer()`: consumes from RingBuffer, writes to TCP

## Important Implementation Details

### TLS Certificates
**Hardcoded** in `chat/server.go:11-34` (self-signed certificate). Clients must set `InsecureSkipVerify: true`.

### Filter/Blacklist System
- Loads from `./filter/blacklist.json` every 120 seconds
- Format: `{"uid": roomId}` where `roomId=0` means global ban
- Type inconsistency: `rid` can be `int` or `string` (see `filter.go:93-99`)

### Double-Check Locking Bug
`ChatServer.GetRoom()` (server.go:66-82) has a race condition between the RUnlock and Lock calls - multiple goroutines can create duplicate rooms.

### Fatal Error Handling
Multiple places use `log.Fatal()` which **terminates the entire process**:
- `server.go:136` (TCP listen failure)
- `server.go:143` (TLS cert parse failure)
- `room.go:119` (NATS connection failure)

### Goroutine Management
**No graceful shutdown** - `Close()` methods don't wait for goroutines to exit. Goroutine count scales linearly with clients (no worker pools).

## Code Style Conventions

- **Method receiver:** Always use `self` (not `this` or abbreviated names)
- **Constructor pattern:** `NewChatServer()`, `NewClient()`, `NewRoom()`
- **Panic recovery:** Use `defer func() { if r := recover(); ... }()` in goroutines
- **Channel closing:** Only close in owner goroutine, never in receiver
- **Logging:** `log.Printf()` with context, `log.Println()` for simple cases

## Module Structure

- **gochatserver** is the module name (not a domain path)
- Import core package as: `"gochatserver/chat"`
- Dependencies cached in `~/go/pkg/mod/` (Go Modules mode)
