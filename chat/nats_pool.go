package chat

import (
	"log"
	"sync"
	"time"

	"github.com/nats-io/nats"
)

// NATSConnection 包装 NATS 连接，增加健康状态管理
type NATSConnection struct {
	conn           *nats.Conn
	encodedConn    *nats.Conn
	healthy        bool
	lastUsed       time.Time
	reconnectCount int
	mu             sync.RWMutex
	url            string
}

// NATSConnectionPool 管理 NATS 连接池
type NATSConnectionPool struct {
	connections    []*NATSConnection
	currentIndex   int
	mu             sync.RWMutex
	url            string
	maxConnections int
	healthCheck    bool
}

// NewNATSConnectionPool 创建新的连接池
func NewNATSConnectionPool(url string, maxConnections int) *NATSConnectionPool {
	if maxConnections < 1 {
		maxConnections = 2 // 默认至少 2 个连接
	}

	pool := &NATSConnectionPool{
		url:            url,
		maxConnections: maxConnections,
		healthCheck:    true,
		currentIndex:   0,
		connections:    make([]*NATSConnection, 0, maxConnections),
	}

	// 初始化连接
	for i := 0; i < maxConnections; i++ {
		conn, err := pool.createConnection()
		if err != nil {
			log.Printf("Failed to create initial NATS connection %d: %v\n", i, err)
			continue
		}
		pool.connections = append(pool.connections, conn)
	}

	// 启动健康检查 goroutine
	if pool.healthCheck && len(pool.connections) > 0 {
		go pool.startHealthCheck()
	}

	log.Printf("NATS connection pool initialized with %d connections\n", len(pool.connections))
	return pool
}

// createConnection 创建新的 NATS 连接
func (pool *NATSConnectionPool) createConnection() (*NATSConnection, error) {
	opts := []nats.Option{
		nats.ReconnectWait(2 * time.Second),
		nats.MaxReconnects(5),
		nats.PingInterval(20 * time.Second),
		nats.MaxPingsOutstanding(5),
		nats.ClosedHandler(func(nc *nats.Conn) {
			log.Printf("NATS connection closed: %s\n", nc.Status().String())
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Printf("NATS connection reconnected: %s\n", nc.ConnectedUrl())
		}),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			log.Printf("NATS connection disconnected: %v\n", err)
		}),
	}

	nc, err := nats.Connect(pool.url, opts...)
	if err != nil {
		return nil, err
	}

	conn := &NATSConnection{
		conn:           nc,
		encodedConn:    nc, // EncodedConn 在使用时创建
		healthy:        true,
		lastUsed:       time.Now(),
		reconnectCount: 0,
		url:            pool.url,
	}

	return conn, nil
}

// GetConnection 获取一个可用的连接（轮询方式）
func (pool *NATSConnectionPool) GetConnection() *NATSConnection {
	pool.mu.RLock()
	if len(pool.connections) == 0 {
		pool.mu.RUnlock()
		return nil
	}

	// 轮询获取连接
	conn := pool.connections[pool.currentIndex]
	pool.currentIndex = (pool.currentIndex + 1) % len(pool.connections)
	pool.mu.RUnlock()

	// 检查连接健康状态
	if !conn.IsHealthy() {
		log.Printf("Connection unhealthy, attempting reconnect...\n")
		conn = pool.reconnectConnection(conn)
	}

	if conn != nil {
		conn.UpdateLastUsed()
	}

	return conn
}

// reconnectConnection 重连不健康的连接
func (pool *NATSConnectionPool) reconnectConnection(oldConn *NATSConnection) *NATSConnection {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	// 查找连接索引
	index := -1
	for i, c := range pool.connections {
		if c == oldConn {
			index = i
			break
		}
	}

	if index == -1 {
		return nil
	}

	// 关闭旧连接
	oldConn.Close()

	// 创建新连接
	newConn, err := pool.createConnection()
	if err != nil {
		log.Printf("Failed to reconnect: %v\n", err)
		// 保留旧连接在池中，下次重试
		newConn = oldConn
	} else {
		log.Printf("Successfully reconnected NATS connection\n")
		newConn.reconnectCount = oldConn.reconnectCount + 1
		pool.connections[index] = newConn
	}

	return newConn
}

// GetEncodedConn 获取编码连接（兼容现有代码）
func (conn *NATSConnection) GetEncodedConn() (*nats.EncodedConn, error) {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	if conn.encodedConn == nil || !conn.healthy {
		return nil, nats.ErrBadConnection
	}

	return nats.NewEncodedConn(conn.encodedConn, nats.JSON_ENCODER)
}

// IsHealthy 检查连接是否健康
func (conn *NATSConnection) IsHealthy() bool {
	conn.mu.RLock()
	defer conn.mu.RUnlock()

	if conn.conn == nil {
		return false
	}

	// NATS 连接状态检查
	status := conn.conn.Status()
	return status == nats.CONNECTED && conn.healthy
}

// UpdateLastUsed 更新最后使用时间
func (conn *NATSConnection) UpdateLastUsed() {
	conn.mu.Lock()
	defer conn.mu.Unlock()
	conn.lastUsed = time.Now()
}

// Close 关闭连接
func (conn *NATSConnection) Close() {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	if conn.conn != nil {
		conn.conn.Close()
		conn.healthy = false
	}
}

// startHealthCheck 启动健康检查
func (pool *NATSConnectionPool) startHealthCheck() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		pool.checkAllConnections()
	}
}

// checkAllConnections 检查所有连接的健康状态
func (pool *NATSConnectionPool) checkAllConnections() {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	for i, conn := range pool.connections {
		if !conn.IsHealthy() {
			log.Printf("Connection %d is unhealthy, scheduling reconnect...\n", i)
			// 注意：这里不能直接调用 reconnectConnection，因为已经持有 RLock
			// 实际重连会在 GetConnection 时触发
		}
	}
}

// Close 关闭连接池
func (pool *NATSConnectionPool) Close() {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	for _, conn := range pool.connections {
		if conn != nil {
			conn.Close()
		}
	}
	pool.connections = nil
	log.Printf("NATS connection pool closed\n")
}

// Stats 获取连接池统计信息
func (pool *NATSConnectionPool) Stats() map[string]interface{} {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	healthyCount := 0
	for _, conn := range pool.connections {
		if conn != nil && conn.IsHealthy() {
			healthyCount++
		}
	}

	return map[string]interface{}{
		"total_connections":   len(pool.connections),
		"healthy_connections": healthyCount,
		"url":                 pool.url,
	}
}
