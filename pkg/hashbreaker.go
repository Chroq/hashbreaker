package pkg

import (
	"crypto/sha256"
	"math"
)

const (
	Charset  = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789@"
	MaxDepth = 8
)

type Candidate struct {
	Hash [32]byte
	Word string
}

type ArrayReferential struct {
	HashedPasswords []Candidate
}

func NewArrayReferential(depth int) ArrayReferential {
	capacity := int(math.Pow(float64(len(Charset)), float64(depth)))
	referential := ArrayReferential{
		HashedPasswords: make([]Candidate, 0, capacity),
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
			referential.HashedPasswords = append(referential.HashedPasswords, Candidate{
				Hash: hashedPassword,
				Word: string(buf),
			})
		}
	}

	return referential
}

func (r ArrayReferential) Get(hash [32]byte) string {
	for i := range r.HashedPasswords {
		if r.HashedPasswords[i].Hash == hash {
			return r.HashedPasswords[i].Word
		}
	}
	return ""
}

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
