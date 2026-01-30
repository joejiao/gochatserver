package main

import (
	"fmt"
	"github.com/jaehue/anyq"
	"github.com/nats-io/nats.go"
	"io/ioutil"
	"log"
	"strconv"
	"testing"
	"time"
)

var (
	testMsg = "hello world"
)

func BenchmarkNatsProduce(b *testing.B) {
	log.SetOutput(ioutil.Discard)

	q, err := anyq.New("nats", "nats://10.1.64.2:4222")
	if err != nil {
		b.Error(err)
	}
	produceBenchmark(b, q, anyq.NatsProducerArgs{Subject: "test"})
}

func BenchmarkNatsConsume(b *testing.B) {
	log.SetOutput(ioutil.Discard)

	q, err := anyq.New("nats", "nats://10.1.64.2:4222")
	if err != nil {
		b.Error(err)
	}

	pubsubBenchmark(b, q, anyq.NatsProducerArgs{Subject: "test"}, anyq.NatsConsumerArgs{Subject: "test"})
}

/*
func BenchmarkNatsReply(b *testing.B) {
	log.SetOutput(ioutil.Discard)

	q, err := anyq.New("nats", "nats://10.1.64.2:4222")
	if err != nil {
		panic(err)
	}

	conn, err := q.Conn()
	if err != nil {
		b.Error(err)
	}
	natsConn, ok := conn.(*nats.Conn)
	if !ok {
		log.Fatalf("invalid conn type(%T)\n", conn)
	}

	// set consumer for reply
	natsConn.Subscribe("test", func(m *nats.Msg) {
		natsConn.Publish(m.Reply, m.Data)
		log.Println("[receive and reply]", string(m.Data))
	})

	// set producer for request
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		body := strconv.Itoa(i)

		m, err := natsConn.Request("test", []byte(body), 10*time.Second)
		if err != nil {
			log.Fatalln(err)
			return
		}
		log.Println("[replied]", string(m.Data))
	}
}
*/

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
