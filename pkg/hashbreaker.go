package pkg

import (
	"crypto/sha256"
)

const (
	Charset  = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789@"
	MaxDepth = 8
)

type MapReferential struct {
	HashedPasswords map[[32]byte]string
}

func TotalCombinations(depth int) int {
	total := 0
	curr := 1
	n := len(Charset)
	for d := 1; d <= depth; d++ {
		curr *= n
		total += curr
	}
	return total
}

func NewMapReferential(depth int) MapReferential {
	referential := MapReferential{
		HashedPasswords: make(map[[32]byte]string, TotalCombinations(depth)),
	}

	for d := 1; d <= depth; d++ {
		indices := make([]int, d)
		buf := make([]byte, d)
		for i := range buf {
			buf[i] = Charset[0]
		}

		for {
			hashedPassword := sha256.Sum256(buf)
			referential.HashedPasswords[hashedPassword] = string(buf)

			pos := d - 1
			for pos >= 0 {
				indices[pos]++
				if indices[pos] < len(Charset) {
					buf[pos] = Charset[indices[pos]]
					break
				}
				indices[pos] = 0
				buf[pos] = Charset[0]
				pos--
			}

			if pos < 0 {
				break
			}
		}
	}

	return referential
}

func (r MapReferential) Get(hash [32]byte) (string, bool) {
	word, ok := r.HashedPasswords[hash]
	return word, ok
}

func GetHash(word string) [32]byte {
	return sha256.Sum256([]byte(word))
}
