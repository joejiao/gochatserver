#!/bin/bash

echo "=========================================="
echo "Go 聊天服务器 - 架构优化测试"
echo "=========================================="
echo ""

export GOPATH="/Users/jiaoshengqiang/GitLab/gochatserver"
export GO111MODULE=off

echo "1. 检查代码编译..."
echo "----------------------------------------"
cd "$GOPATH"
if go build -o chatserver chatserver.go 2>&1; then
    echo "✓ 代码编译成功"
else
    echo "✗ 代码编译失败"
    exit 1
fi
echo ""

echo "2. 运行基准测试..."
echo "----------------------------------------"
echo "运行 Ring Buffer 基准测试..."
go test -bench=BenchmarkRingBuffer ./test/ -benchmem
echo ""

echo "3. 验证连接池功能..."
echo "----------------------------------------"
cat > /tmp/test_nats_pool.go << 'EOF'
package main

import (
	"fmt"
	"testing"
	"time"

	"./chat"
)

func TestConnectionPoolCreation(t *testing.T) {
	url := "nats://127.0.0.1:4222"
	pool := chat.NewNATSConnectionPool(url, 3)

	if pool == nil {
		t.Fatal("Failed to create connection pool")
	}

	stats := pool.Stats()
	fmt.Printf("Connection Pool Stats: %+v\n", stats)
}

func TestGetConnection(t *testing.T) {
	url := "nats://127.0.0.1:4222"
	pool := chat.NewNATSConnectionPool(url, 2)

	conn1 := pool.GetConnection()
	conn2 := pool.GetConnection()

	if conn1 == nil || conn2 == nil {
		t.Fatal("Failed to get connections from pool")
	}

	fmt.Printf("✓ Successfully got connections from pool\n")
}

func TestConnectionHealthCheck(t *testing.T) {
	url := "nats://127.0.0.1:4222"
	pool := chat.NewNATSConnectionPool(url, 2)

	time.Sleep(2 * time.Second) // Wait for health check

	conn := pool.GetConnection()
	if conn == nil {
		t.Fatal("Failed to get connection")
	}

	if !conn.IsHealthy() {
		t.Fatal("Connection is not healthy")
	}

	fmt.Printf("✓ Connection health check passed\n")
}

func BenchmarkGetConnection(b *testing.B) {
	url := "nats://127.0.0.1:4222"
	pool := chat.NewNATSConnectionPool(url, 3)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn := pool.GetConnection()
		if conn != nil {
			conn.UpdateLastUsed()
		}
	}
}

func main() {
	testing.Main(func(pat, str string) (bool, error) { return true, nil },
		[]testing.InternalTest{
			{"TestConnectionPoolCreation", TestConnectionPoolCreation},
			{"TestGetConnection", TestGetConnection},
			{"TestConnectionHealthCheck", TestConnectionHealthCheck},
		},
		nil,
		nil,
	)
}
EOF

if go run /tmp/test_nats_pool.go 2>&1; then
    echo "✓ 连接池功能测试通过"
else
    echo "✗ 连接池功能测试失败"
    echo "  注意: 需要 NATS 服务器运行在 nats://127.0.0.1:4222"
fi
echo ""

echo "4. 架构优化总结"
echo "----------------------------------------"
echo "✓ 创建了 NATS 连接池 (chat/nats_pool.go)"
echo "✓ 实现了连接复用和健康检查"
echo "✓ 集成到 ChatServer，移除重复连接"
echo "✓ 重构 Room 的 writeToNATS 和 readFromNATS 方法"
echo "✓ 移除 room.go 中的 nats 直接导入"
echo ""
echo "优化效果:"
echo "  - 从每个 Room 2 个 NATS 连接 -> 3 个共享连接"
echo "  - 实现了连接池管理和负载均衡"
echo "  - 添加了自动重连和健康检查"
echo "  - 提高了资源利用率和系统稳定性"
echo ""

echo "=========================================="
echo "测试完成"
echo "=========================================="
