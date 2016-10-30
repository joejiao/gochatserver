package main

import (
    "log"
    //"time"
    "strconv"

    "github.com/nsqio/go-nsq"
)

var (
    nsqdServer = "10.1.64.2:4150"
    exitChan = make(chan struct{})
)

func main() {
    go startConsumer()
    startProducer()
}

// 生产者
func startProducer() {
    cfg := nsq.NewConfig()
    cfg.MaxInFlight = 8

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
    cfg.MaxInFlight = 8

    consumer, err := nsq.NewConsumer("testtopic", "channel01", cfg)
    if err != nil {
        log.Fatal("NewConsumer:", err)
    }

    i := 1
    // 设置消息处理函数
    consumer.AddHandler(nsq.HandlerFunc(func(message *nsq.Message) error {
        message.DisableAutoResponse()
        log.Printf("[%s]consumer resp: %s",strconv.Itoa(i), string(message.Body))
        message.Finish()
        i++
        return nil
    }))

    // 连接到单例nsqd
    //err := consumer.ConnectToNSQLookupd("127.0.0.1:4161")
    err = consumer.ConnectToNSQD(nsqdServer)
    if err != nil {
        log.Fatal(err)
    }

    <-exitChan
    log.Println("Consumer quit")
}
