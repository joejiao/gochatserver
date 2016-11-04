# 聊天服务器
---

## 架构
```
SERVER|FILTER
    /  \
 ROOM  ROOM ------ MessageQueue
     /   |   \
   C1   C2   C3
```

Client 用户长连接管理
Room 聊天室信息交换处理
MessageQueue 分布式消息队列，为所有的room提供信息交换
Filter 信息过滤和黑名单处理
Server 聊天服务器全局管理

## 配置
1. 黑名单数据

```bash
mkdir ./filter
cp blacklist.json ./filter/
cat ./filter/blacklist.json
# 数据为json, 格式: uid:roomId, 数据类型: uid是string，roomId是int, 如果全局禁言，则roomId为0
{"1":1, "2":1, "3":0, "4":3}
```

## 运行
1. 启动MessageQueue集群
```bash
gnatsd -m 1234
```

2. 启动聊天服务器
```bash
./chatserver -filter_dir="./filter" -listen="0.0.0.0:9999" -nats_url="nats://127.0.0.1:4222"
```

## API
服务器默传输为TLS加密,连接代码如下:
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
join roomId
```
