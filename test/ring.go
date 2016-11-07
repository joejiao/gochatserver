package main

import (
	"fmt"
    "strconv"
    "../chat/"
)

func main() {
    var size int64 = 512
    rb := chat.NewRingBuffer(size)

    for i := int64(0); i < size + 139; i++ {
		str := strconv.Itoa(int(i))
		//fmt.Println(str)
		rb.Put(str)
	}

	//fmt.Printf("%+v\n", rb.nodes)
	c1 := chat.NewConsumer(rb)
	for i := int64(0); i < size; i++ {
		str, err := c1.Get()
		fmt.Println(i, str, err)
	}
    c2 := chat.NewConsumer(rb)
    str, _ := c2.BatchGet()
    fmt.Printf("%+v\n", str)
}
