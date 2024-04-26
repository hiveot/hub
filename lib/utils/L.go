package utils

import (
	"fmt"
	"os"
	"strings"
)

// L is a simple helper for building string lists
type L struct {
	Lines []string
}

func (l *L) Add(s string, args ...any) *L {
	s2 := fmt.Sprintf(s, args...)
	l.Lines = append(l.Lines, s2)
	return l
}

// Remove removes a row from the list while maintaining order
// This is slower than RemoveNoOrder but maintains order and does not modify
// the original slice.
// see also: https://stackoverflow.com/questions/37334119/how-to-delete-an-element-from-a-slice-in-golang
func (l *L) Remove(row int) {
	if row >= len(l.Lines) {
		return
	}
	l2 := make([]string, 0, len(l.Lines))
	l2 = append(l2, l.Lines[:row]...)
	if row < len(l.Lines)-1 {
		l2 = append(l2, l.Lines[row+1:]...)
	}
}

// RemoveFast is a fast way to remove a row from the list.
// This does not maintain order and modifies the existing slice.
func (l *L) RemoveFast(row int) {
	rem := len(l.Lines) - 1
	if row < rem {
		l.Lines[row] = l.Lines[rem]
	}
	l.Lines = l.Lines[:rem]
}

func (l *L) Size() int {
	return len(l.Lines)
}

// Write the Lines to file
func (l *L) Write(outfile string) error {
	data := strings.Join(l.Lines, "\n")
	err := os.WriteFile(outfile, []byte(data), 0644)
	return err
}
