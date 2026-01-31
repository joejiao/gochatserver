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
go run ./cmd/chatserver \
  -nats_url="nats://127.0.0.1:4222" \
  -listen="0.0.0.0:9999" \
  -filter_dir="./filter" \
  -auth_password="your-password"
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
# 使用默认配置（内置自签名证书，默认密码 "pw"）
./bin/chatserver \
  -filter_dir="./filter" \
  -listen="0.0.0.0:9999" \
  -nats_url="nats://127.0.0.1:4222"

# 使用自定义密码和 TLS 证书
./bin/chatserver \
  -filter_dir="./filter" \
  -listen="0.0.0.0:9999" \
  -nats_url="nats://127.0.0.1:4222" \
  -auth_password="secure-password" \
  -cert_file="/path/to/cert.pem" \
  -key_file="/path/to/key.pem"
```

**启动参数说明：**
- `-nats_url`: NATS 服务器地址（默认: `nats://10.1.64.2:4222`）
- `-listen`: 监听地址（默认: `0.0.0.0:9999`）
- `-filter_dir`: 黑名单配置目录（默认: `./filter`）
- `-auth_password`: 客户端认证密码（默认: `pw`）
- `-cert_file`: TLS 证书文件路径（留空使用内置自签名证书）
- `-key_file`: TLS 私钥文件路径（留空使用内置自签名证书）

## API

### 服务器说明
- 服务器默认使用 TLS 加密传输
- 内置自签名证书用于开发环境（生产环境请使用 `-cert_file` 和 `-key_file` 指定正式证书）
- 支持优雅关闭（`Ctrl+C` 或 `SIGTERM` 信号）

### 客户端连接
由于默认使用自签名证书，客户端需要设置跳过证书校验：
```go
conf := &tls.Config{
    InsecureSkipVerify: true,
}

conn, err := tls.Dial("tcp", *hostAndPort, conf)
```

**Go 客户端示例：**
```go
conf := &tls.Config{
    InsecureSkipVerify: true,  // 开发环境跳过证书验证
}

conn, err := tls.Dial("tcp", "127.0.0.1:9999", conf)
```

**命令行连接：**
```bash
ncat --ssl 127.0.0.1 9999
```

**连接协议：**
```bash
auth <password>      # 认证，默认密码 "pw"
uid <user_id>        # 设置用户 ID
join <room_id>       # 加入房间
```
