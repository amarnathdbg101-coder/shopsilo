package imghash_test

import (
	"fmt"

	"shopMe/internal/pkg/imghash"
)

func ExampleSimilar() {
	// Two perceptual hashes with a single bit difference
	hashA := "8f00ff00ff00ff00"
	hashB := "8f00ff00ff00ff01"

	dist, _ := imghash.Distance(hashA, hashB)
	isSimilar := imghash.Similar(hashA, hashB, 5)

	fmt.Printf("Hamming Distance: %d\n", dist)
	fmt.Printf("Is Similar: %v\n", isSimilar)

	// Output:
	// Hamming Distance: 1
	// Is Similar: true
}
