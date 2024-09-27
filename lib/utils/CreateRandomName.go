package utils

import "github.com/dchest/uniuri"

// Create a random text
func CreateRandomName(prefix string, length int) string {
	if length > 0 {
		return prefix + uniuri.NewLen(length)
	}
	return prefix + uniuri.New()
}
