package main
import (
    "fmt"
)
func main() {
    var n int64 = 43
    var max int64 = 16
    mask := max - 1
    fmt.Println(n % max)
    fmt.Println(n & mask)
    //fmt.Println(roundUp(n))
}
func roundUp(v uint64) uint64 {
    fmt.Printf("v= %064b\n", v)
    v--
    fmt.Printf("-1 %064b\n", v)
    var tmp uint64
    tmp = v >> 1
    fmt.Printf(">> %064b\n", tmp)
    v |= tmp
    fmt.Printf("v| %064b\n", v)
    tmp = v >> 2
    fmt.Printf(">> %064b\n", tmp)
    v |= tmp
    fmt.Printf("v| %064b\n", v)
    tmp = v >> 4
    fmt.Printf(">> %064b\n", tmp)
    v |= tmp
    fmt.Printf("v| %064b\n", v)
    tmp = v >> 8
    fmt.Printf(">> %064b\n", tmp)
    v |= tmp
    fmt.Printf("v| %064b\n", v)
    tmp = v >> 16
    fmt.Printf(">> %064b\n", tmp)
    v |= tmp
    fmt.Printf("v| %064b\n", v)
    tmp = v >> 32
    fmt.Printf(">> %064b\n", tmp)
    v |= tmp
    fmt.Printf("v| %064b\n", v)
    v++
    fmt.Printf("+1 %064b\n", v)
    return v
}
