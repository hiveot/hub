package utils

import (
	"fmt"
	"os"
	"strings"
)

// simple indentation string
const spaces = "                                                                                 "
const indentSize = 4

// SL is a simple helper for building string lists
type SL struct {
	Indent int
	Lines  []string
}

// Add indented text to the string list
func (l *SL) Add(s string, args ...any) *SL {
	if l.Indent > 80/indentSize {
		l.Indent = 80 / indentSize
	}
	s2 := spaces[:l.Indent*indentSize] + fmt.Sprintf(s, args...)
	l.Lines = append(l.Lines, s2)
	return l
}

//// Remove removes a row from the list while maintaining order
//// This is slower than RemoveNoOrder but maintains order and does not modify
//// the original slice.
//// see also: https://stackoverflow.com/questions/37334119/how-to-delete-an-element-from-a-slice-in-golang
//func (l *SL) Remove(row int) {
//	Remove(l.Lines, row)
//}
//
//// RemoveFast is a fast way to remove a row from the list.
//// This does not maintain order and modifies the existing slice.
//func (l *SL) RemoveFast(row int) {
//	RemoveFast(l.Lines, row)
//}

func (l *SL) Size() int {
	return len(l.Lines)
}

// Write the Lines to file
func (l *SL) Write(outfile string) error {
	data := strings.Join(l.Lines, "\n")
	err := os.WriteFile(outfile, []byte(data), 0644)
	return err
}
