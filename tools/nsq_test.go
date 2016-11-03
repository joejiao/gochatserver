package main

import (
	"github.com/jaehue/anyq"
	"io/ioutil"
	"log"
	"testing"
    "fmt"
)

var (
    testMsg = "hello world"
)

func BenchmarkNsqProduce(b *testing.B) {
	log.SetOutput(ioutil.Discard)

	q, err := anyq.New("nsq", "10.1.64.2:4150")
	q.SetLogger(log.New(ioutil.Discard, "", log.LstdFlags), anyq.LogLevelInfo)
	if err != nil {
		b.Error(err)
	}
	produceBenchmark(b, q, anyq.NsqProducerArgs{Topic: "test"})
}

func BenchmarkNsqConsume(b *testing.B) {
	log.SetOutput(ioutil.Discard)

	q, err := anyq.New("nsq", "10.1.64.2:4150")
	q.SetLogger(log.New(ioutil.Discard, "", log.LstdFlags), anyq.LogLevelInfo)
	if err != nil {
		b.Error(err)
	}
	pubsubBenchmark(b, q, anyq.NsqProducerArgs{Topic: "test"}, anyq.NsqConsumerArgs{Topic: "test", Channel: "anyq"})
}

func produceBenchmark(b *testing.B, q anyq.Queuer, args interface{}) {
    p, err := q.NewProducer(args)
    if err != nil {
        b.Error(err)
    }

    sendCh := make(chan []byte)
    err = p.BindSendChan(sendCh)
    if err != nil {
        b.Error(err)
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        body := fmt.Sprintf("[%d]%s", i, testMsg)
        sendCh <- []byte(body)
    }
}

func pubsubBenchmark(b *testing.B, q anyq.Queuer, producerArgs interface{}, consumerArgs interface{}) {
    // run producer
    p, err := q.NewProducer(producerArgs)
    if err != nil {
        b.Error(err)
    }
    sendCh := make(chan []byte)
    err = p.BindSendChan(sendCh)
    if err != nil {
        b.Error(err)
    }

    quit := make(chan struct{})
    go func() {
        i := 0
    produceloop:
        for {
            select {
            case <-quit:
                break produceloop
            default:
                body := fmt.Sprintf("[%d]%s", i, testMsg)
                sendCh <- []byte(body)
                i++
            }
        }
        // b.Logf("produce count: %d", i)
    }()

    // run consumer
    c, err := q.NewConsumer(consumerArgs)
    if err != nil {
        b.Error(err)
    }
    recvCh := make(chan *anyq.Message)
    err = c.BindRecvChan(recvCh)
    if err != nil {
        b.Error(err)
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        <-recvCh
    }
    quit <- struct{}{}
}
