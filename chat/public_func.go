package chat

import (
	"os"
)

func IsFileExist(name string) bool {
    if _, err := os.Stat(name); os.IsNotExist(err) {
        return false
    }
    return true
}

// roundUp takes a uint64 greater than 0 and rounds it up to the next
// power of 2.
func roundUp(v uint64) uint64 {
    v--
    v |= v >> 1
    v |= v >> 2
    v |= v >> 4
    v |= v >> 8
    v |= v >> 16
    v |= v >> 32
    v++
    return v
}
