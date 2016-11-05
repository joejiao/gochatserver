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
    topic   = "test"
    nsqServer = "127.0.0.1:4150"
)

func BenchmarkNsqProduce(b *testing.B) {
	log.SetOutput(ioutil.Discard)

	q, err := anyq.New("nsq", nsqServer)
	q.SetLogger(log.New(ioutil.Discard, "", log.LstdFlags), anyq.LogLevelInfo)
	if err != nil {
		b.Error(err)
	}
	produceBenchmark(b, q, anyq.NsqProducerArgs{Topic: topic})
}

func BenchmarkNsqConsume(b *testing.B) {
	log.SetOutput(ioutil.Discard)

	q, err := anyq.New("nsq", nsqServer)
	q.SetLogger(log.New(ioutil.Discard, "", log.LstdFlags), anyq.LogLevelInfo)
	if err != nil {
		b.Error(err)
	}
	pubsubBenchmark(b, q, anyq.NsqProducerArgs{Topic: topic}, anyq.NsqConsumerArgs{Topic: topic, Channel: "anyq#ephemeral"})
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
    sendCh := make(chan []byte, 5000)
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
    recvCh := make(chan *anyq.Message, 5000)
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
