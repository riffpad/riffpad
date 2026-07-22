package idgen

import (
	"fmt"
	"math/rand"
)

// ID returns a 12-char random identifier (lowercase alphanumeric).
func ID() string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 12)
	for i := range b {
		b[i] = alphabet[rand.Intn(len(alphabet))]
	}
	return string(b)
}

// Slug returns a human-friendly random slug, e.g. "amber-spark-42".
func Slug() string {
	adjectives := []string{"amber", "velvet", "silent", "brisk", "lunar", "solar", "neon", "quiet", "bold", "swift"}
	nouns := []string{"spark", "breeze", "wave", "pulse", "orbit", "flame", "drift", "echo", "glint", "nova"}
	adj := adjectives[rand.Intn(len(adjectives))]
	noun := nouns[rand.Intn(len(nouns))]
	return fmt.Sprintf("%s-%s-%d", adj, noun, rand.Intn(100))
}
