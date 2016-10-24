package main

import "fmt"

func main() {
        c := make(chan int, 100)
        fmt.Println(len(c))
        for i := 0; i < int(100 / 3); i++ {
                c <- i
        }
        close(c)
        for n := len(c); n >= 0; n-- {
            s, ok := <-c
            fmt.Println(n, s, ok)
        }
        fmt.Println(len(c))
        fmt.Println(cap(c))
}
