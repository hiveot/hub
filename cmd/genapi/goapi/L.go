package goapi

import (
	"fmt"
	"os"
	"strings"
)

// L is a type for building source files out of a TD
type L []string

func (l L) Add(s string, args ...any) L {
	s2 := fmt.Sprintf(s, args...)
	l2 := append(l, s2)
	return l2
}

// Write the lines to file
func (l L) Write(outfile string) error {
	data := strings.Join(l, "\n")
	err := os.WriteFile(outfile, []byte(data), 0644)
	return err
}
