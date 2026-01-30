# 示例程序

本目录包含 gochatserver 的示例程序，帮助理解和使用聊天服务器。

## client.go

压力测试客户端，用于模拟多个连接测试服务器性能。

### 功能特性

- 模拟多个并发客户端连接
- 支持配置最大连接数
- 支持配置消息发送速率（QPS）
- 自动进行认证、注册UID、加入房间
- 发送测试消息并丢弃服务器响应

### 运行方式

```bash
# 基本用法
go run examples/client.go

# 自定义配置
go run examples/client.go -host=127.0.0.1:9999 -maxconn=2000 -qps=10
```

### 命令行参数

- `-host`: 服务器地址和端口（默认：127.0.0.1:9999）
- `-maxconn`: 最大连接数（默认：2000）
- `-qps`: 每秒消息数（默认：10）

### 注意事项

- 此程序使用 TLS 连接服务器，默认跳过证书验证
- 每个连接发送100条消息后自动断开
- 建议在测试环境使用，避免在生产环境运行

## 协议

客户端连接流程：

1. 建立TLS连接
2. 发送认证命令：`auth pw\n`
3. 发送UID注册：`uid <uid>\n`
4. 加入房间：`join room<roomId>\n`
5. 发送消息：`room: [<roomId>] <uid>-<msgId>\n`

## 示例输出

```
connn to server: 127.0.0.1:xxxxx
connn to server: 127.0.0.1:xxxxx
...
disconnect: 127.0.0.1:xxxxx
disconnect: 127.0.0.1:xxxxx
...
```
