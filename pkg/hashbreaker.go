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

func NewMapReferential(depth int) MapReferential {
	referential := MapReferential{
		HashedPasswords: make(map[[32]byte]string),
	}

	for d := 1; d <= depth; d++ {
		indices := make([]int, d)
		buf := make([]byte, d)

		for {
			for i, idx := range indices {
				buf[i] = Charset[idx]
			}

			pos := d - 1
			for pos >= 0 {
				indices[pos]++
				if indices[pos] < len(Charset) {
					break
				}
				indices[pos] = 0
				pos--
			}

			if pos < 0 {
				break
			}

			hashedPassword := sha256.Sum256(buf)
			referential.HashedPasswords[hashedPassword] = string(buf)
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
