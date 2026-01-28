package main

import "fmt"

func main() {
    a := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
    b := make([]int, 7)

    fmt.Println(a[3:9])

    tmp := copy(b, a[3:])
    fmt.Println(tmp)
    fmt.Println(b)
}
