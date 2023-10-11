package utils

// Clone the given byte array.
// Use this is the source holds a transient value and its buffer can be reused elsewhere.
// Eg, all capnp byte arrays
// Golang will eventually get a bytes.clone() method but it isn't in go-19
//
// returns nil if b is nil
func Clone(b []byte) (c []byte) {
	if b != nil {
		c = []byte(string(b))
	}
	return c
}
