package main

import (
	"crypto/sha256"
	"os"
)

const (
	charset   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789@"
	maxDeepth = 8
)

func main() {
	inputWord := os.Args[1]
	targetz3DHash := sha256.Sum256([]byte(inputWord))
	BruteForceIterative(targetz3DHash, maxDeepth)
	BruteForceRecursive(targetz3DHash, maxDeepth)
}

func BruteForceIterative(hash [32]byte, depth int) string {
	for d := 1; d <= depth; d++ {
		indices := make([]int, d)
		buf := make([]byte, d)

		for {
			for i, idx := range indices {
				buf[i] = charset[idx]
			}

			if sha256.Sum256(buf) == hash {
				return string(buf)
			}

			pos := d - 1
			for pos >= 0 {
				indices[pos]++
				if indices[pos] < len(charset) {
					break
				}
				indices[pos] = 0
				pos--
			}

			if pos < 0 {
				break
			}
		}
	}

	return ""
}

func BruteForceRecursive(hash [32]byte, depth int) string {
	for d := 1; d <= depth; d++ {
		res := search("", d, hash)
		if res != "" {
			return res
		}
	}
	return ""
}

func search(current string, targetLen int, hash [32]byte) string {
	if len(current) == targetLen {
		if sha256.Sum256([]byte(current)) == hash {
			return current
		}
		return ""
	}

	for i := 0; i < len(charset); i++ {
		res := search(current+string(charset[i]), targetLen, hash)
		if res != "" {
			return res
		}
	}

	return ""
}
