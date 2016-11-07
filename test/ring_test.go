package chat

import (
    "testing"
    "strconv"
    "../chat/"
)

func BenchmarkRingBufferGet(b *testing.B) {
    var size int64 = 1024
    rb := chat.NewRingBuffer(size)

    for i := int64(0); i < size + 139; i++ {
        str := strconv.Itoa(int(i))
        //fmt.Println(str)
        rb.Put(str)
    }

    b.ResetTimer()
    c1 := chat.NewConsumer(rb)
    for i := 0; i < b.N; i++ {
        _, err := c1.Get()
        if err != nil {
            
            //b.Printf(err)
        }
    }
}

func BenchmarkRingBufferPut(b *testing.B) {
    var size int64 = 1024
    rb := chat.NewRingBuffer(size)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        str := strconv.Itoa(int(i))
        //fmt.Println(str)
        rb.Put(str)
    }
}
