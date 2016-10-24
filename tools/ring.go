package main

import (
	"fmt"
	"strconv"
	"sync/atomic"
)

const (
	cpuCacheLinePadding        = 7
	InitialSequenceValue int64 = -1
)

type Sequence struct {
	cursor  int64
	padding [cpuCacheLinePadding]int64
}

func NewSequence() *Sequence {
	return &Sequence{cursor: InitialSequenceValue}
}
func (seq *Sequence) get() int64 {
	return atomic.LoadInt64(&seq.cursor)
}
func (seq *Sequence) set(cursor int64) {
	atomic.StoreInt64(&seq.cursor, cursor)
}
func (seq *Sequence) add(i int64) {
	atomic.AddInt64(&seq.cursor, i)
}
func (seq *Sequence) cas(old, new int64) bool {
	return atomic.CompareAndSwapInt64(&seq.cursor, old, new)
}

type RingBuffer struct {
	size             int64
	mask             int64
	producerSequence *Sequence
	buffer           []interface{}
}

func NewRingBuffer(size int64) *RingBuffer {
	return &RingBuffer{
		size:             size,
		mask:             size - 1,
		producerSequence: NewSequence(),
		buffer:           make([]interface{}, size),
	}
}
func (rb *RingBuffer) put(item interface{}) error {
	producerPos := rb.producerSequence.get()
	nextPos := producerPos + 1

	for {
		if rb.producerSequence.cas(producerPos, nextPos) {
			break
		}
	}

	//fmt.Println(producerPos, item)
	rb.buffer[nextPos&rb.mask] = item
	return nil
}

type Consumer struct {
	sequence   *Sequence
	ringBuffer *RingBuffer
}

func NewConsumer(rb *RingBuffer) *Consumer {
	c := &Consumer{sequence: NewSequence(), ringBuffer: rb}
	//c.sequence.set(0)
	return c
}

func (this *Consumer) get() (interface{}, error) {
    consumerPos := this.sequence.get()

    // producerPos 为目前最大写入的pos值
    producerPos := this.ringBuffer.producerSequence.get()
    minConsumerPos := producerPos - this.ringBuffer.size + 1
    //fmt.Println(producerPos, consumerPos, minConsumerPos)

    // 要取的pos还没写到
    if consumerPos >= producerPos {
        return nil, fmt.Errorf("no new data, pos: %d", consumerPos)
        //continue
    }

    // 如果要取的值已经被覆盖，取最小的有效数据
    if consumerPos < minConsumerPos {
        this.sequence.set(minConsumerPos - 1)
        consumerPos = minConsumerPos - 1
        //return "", fmt.Errorf("consumerPos too old %d %d %d", producerPos, consumerPos, minConsumerPos)
    }

    nextPos := consumerPos + 1
    item := this.ringBuffer.buffer[nextPos&this.ringBuffer.mask]
    this.sequence.add(1)
    return item, nil
}

func main() {
    var size int64 = 512
    rb := NewRingBuffer(size)

    for i := int64(0); i < size + 13; i++ {
		str := strconv.Itoa(int(i))
		//fmt.Println(str)
		rb.put(str)
	}

	fmt.Printf("%+v\n", rb.buffer)
	c1 := NewConsumer(rb)
	for i := int64(0); i < size + 10; i++ {
		str, err := c1.get()
		fmt.Println(i, str, err)
	}
	/*
		c2 := NewConsumer(rb)
		for i := int64(0); i < size; i++ {
			str, _ := c2.get()
			fmt.Println(i, str)
		}
	*/
}
