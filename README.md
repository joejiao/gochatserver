# 聊天服务器
---

## 架构

```
SERVER|FILTER
    /  \
 ROOM  ROOM ------ MessageQueue
     /  |  \
   C1   C2  C3
```

Client 用户长连接管理  
Room 聊天室信息交换处理  
MessageQueue 分布式消息队列，为所有的room提供信息交换  
Filter 信息过滤和黑名单处理  
Server 聊天服务器全局管理  

## 配置
- 黑名单数据

```bash
mkdir ./filter
cp blacklist.json ./filter/
cat ./filter/blacklist.json
# 数据为json, 格式: uid:roomId, 数据类型: uid是string，roomId是int, 如果全局禁言，则roomId为0
{"1":1, "2":1, "3":0, "4":3}
```

## 项目结构

```
gochatserver/
├── cmd/
│   └── chatserver/     # 主服务器入口
│       └── main.go
├── chat/               # 核心包
│   ├── server.go       # 服务器核心逻辑
│   ├── room.go         # 聊天室管理
│   ├── client.go       # 客户端连接管理
│   ├── ring.go         # 环形缓冲区实现
│   ├── nats_pool.go    # NATS 连接池
│   ├── filter.go       # 消息过滤
│   └── ...
├── examples/           # 示例程序
│   ├── client.go       # 压力测试客户端
│   └── README.md
├── test/
│   ├── benchmark/      # 性能测试
│   │   └── ring_benchmark_test.go
│   └── demo/           # 演示程序
│       └── ring_demo.go
├── filter/             # 黑名单配置目录
└── go.mod              # Go Modules 配置
```

## 运行

### 构建

项目使用 Go Modules 管理依赖：

```bash
# 构建主服务器
go build -o bin/chatserver ./cmd/chatserver

# 或直接运行
go run ./cmd/chatserver -nats_url="nats://127.0.0.1:4222" -listen="0.0.0.0:9999" -filter_dir="./filter"
```

### 启动

- 启动MessageQueue集群

```bash
wget https://github.com/nats-io/gnatsd/releases/download/v0.9.4/gnatsd-v0.9.4-linux-amd64.zip
unzip gnatsd-v0.9.4-linux-amd64.zip
cp gnatsd-v0.9.4-linux-amd64/gnatsd /usr/local/sbin/
chmod a+x /usr/local/sbin/gnatsd
nohup gnatsd -m 1234 > /tmp/gnatsd.log 2>&1 &
```

- 启动聊天服务器

```bash
./bin/chatserver -filter_dir="./filter" -listen="0.0.0.0:9999" -nats_url="nats://127.0.0.1:4222"
```

## API
服务器默认传输为TLS加密，由于使用了自签名证书，需要设置client将不再对服务端的证书进行校验，连接代码如下:
```go
conf := &tls.Config{
    InsecureSkipVerify: true,
}

conn, err := tls.Dial("tcp", *hostAndPort, conf)
```

客户端连接方法：
```bash
ncat --ssl 127.0.0.1 9999
auth password
uid 1111
join roomId
```
