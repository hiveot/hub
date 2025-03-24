package main

import (
	"crypto/md5"
	"fmt"
)

// Generate a hex key from a password
func main() {
	pass := "My name is groot"
	data := []byte(pass)
	key := md5.Sum(data)
	keyStr := ""
	for i, val := range key {
		if i < 15 {
			keyStr = keyStr + fmt.Sprintf("%02X", val)
		} else {
			keyStr = keyStr + fmt.Sprintf("%02X", val)
		}
	}
	fmt.Printf("OpenZWaveAdapter.LoadingConfiguration: Adding network key: %s\n", keyStr)

}
