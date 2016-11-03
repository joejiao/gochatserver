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
