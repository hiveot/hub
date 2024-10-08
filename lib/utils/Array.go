package utils

// helper functions to work with arrays

// Map2Array makes an array from a map
func Map2Array[T comparable](m map[string]T) []T {
	arr := make([]T, 0, len(m))
	for _, v := range m {
		arr = append(arr, v)
	}
	return arr
}

// Remove removes a row from the slice while maintaining order.
// This is slower than RemoveNoOrder but maintains order and does not modify
// the original slice.
// see also: https://stackoverflow.com/questions/37334119/how-to-delete-an-element-from-a-slice-in-golang
//
// Note: it should be part of the go standard library
func Remove[T comparable](arr []T, row int) []T {
	if row >= len(arr) {
		return arr
	}
	l2 := make([]T, 0, len(arr))
	l2 = append(l2, arr[:row]...)
	if row < len(arr)-1 {
		l2 = append(l2, arr[row+1:]...)
	}
	// zero out the old value to release memory (in case of pointers)
	var zero T
	arr[row] = zero
	return l2
}

// RemoveFast is a fast way to remove a row from the slice.
// This does not maintain order and modifies the existing slice.
//
// Note: it should be part of the go standard library
func RemoveFast[T comparable](arr []T, row int) []T {
	rem := len(arr) - 1
	if row < rem {
		arr[row] = arr[rem]
		// https://stackoverflow.com/questions/70585852/return-default-value-for-generic-type
		var zero T
		arr[rem] = zero
	}
	arr = arr[:rem]
	return arr
}
