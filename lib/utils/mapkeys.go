package utils

import (
	"golang.org/x/exp/constraints"
	"sort"
)

// MapKeys returns an slice of keys
func MapKeys[K comparable, V any](mm map[K]V) []K {
	res := make([]K, 0, len(mm))
	for k, _ := range mm {
		res = append(res, k)
	}
	return res
}

// OrderedMapKeys returns an ordered slice of keys
func OrderedMapKeys[K constraints.Ordered, V any](mm map[K]V) []K {
	res := make([]K, 0, len(mm))
	for k, _ := range mm {
		res = append(res, k)
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i] < res[j]
	})
	return res
}
