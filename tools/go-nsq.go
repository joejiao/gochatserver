package main

import (
    "log"
    //"time"
    "strconv"

    "github.com/nsqio/go-nsq"
)

var (
    nsqdServer = "10.1.64.2:4150"
)

func main() {
    go startConsumer()
    startProducer()
}

// 生产者
func startProducer() {
    cfg := nsq.NewConfig()

    producer, err := nsq.NewProducer(nsqdServer, cfg)
    if err != nil {
        log.Fatal(err)
    }

    // 发布消息
    i := 0
    for {
        msg := "test message " + strconv.Itoa(i)
        if err := producer.Publish("testtopic", []byte(msg)); err != nil {
            log.Fatal("publish error: " + err.Error())
        }

        //time.Sleep(1 * time.Second)
        i++
    }
}

// 消费者
func startConsumer() {
    cfg := nsq.NewConfig()

    consumer, err := nsq.NewConsumer("testtopic", "channel01", cfg)
    if err != nil {
        log.Fatal("NewConsumer:", err)
    }

    i := 1
    // 设置消息处理函数
    consumer.AddHandler(nsq.HandlerFunc(func(message *nsq.Message) error {
        log.Printf("[%s]consumer resp: %s",strconv.Itoa(i), string(message.Body))
        i++
        return nil
    }))

    // 连接到单例nsqd
    if err := consumer.ConnectToNSQD(nsqdServer); err != nil {
        log.Fatal(err)
    }
}
