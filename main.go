package main

import (
	"crypto/sha256"
	"math"
	"os"
)

const (
	charset   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789@"
	maxDeepth = 8
)

type Candidate struct {
	hash [32]byte
	word string
}

type ArrayReferential struct {
	hashedPasswords []Candidate
}

func NewArrayReferential(depth int) ArrayReferential {
	capacity := int(math.Pow(float64(len(charset)), float64(depth)))
	referental := ArrayReferential{
		hashedPasswords: make([]Candidate, 0, capacity),
	}

	for d := 1; d <= depth; d++ {
		indices := make([]int, d)
		buf := make([]byte, d)

		for {
			for i, idx := range indices {
				buf[i] = charset[idx]
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

			hashedPassword := sha256.Sum256(buf)
			referental.hashedPasswords = append(referental.hashedPasswords, Candidate{
				hash: hashedPassword,
				word: string(buf),
			})
		}
	}

	return referental
}

func (r ArrayReferential) Get(hash [32]byte) string {
	for i := range r.hashedPasswords {
		if r.hashedPasswords[i].hash == hash {
			return r.hashedPasswords[i].word
		}
	}
	return ""
}

type MapReferential struct {
	hashedPasswords map[[32]byte]string
}

func NewMapReferential(depth int) MapReferential {
	referental := MapReferential{
		hashedPasswords: make(map[[32]byte]string),
	}

	for d := 1; d <= depth; d++ {
		indices := make([]int, d)
		buf := make([]byte, d)

		for {
			for i, idx := range indices {
				buf[i] = charset[idx]
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

			hashedPassword := sha256.Sum256(buf)
			referental.hashedPasswords[hashedPassword] = string(buf)
		}
	}

	return referental
}

func (r MapReferential) Get(hash [32]byte) (string, bool) {
	word, ok := r.hashedPasswords[hash]
	return word, ok
}

func main() {
	inputHash := GetHash(os.Args[1])

	mapReferential := NewMapReferential(maxDeepth)
	arrayReferential := NewArrayReferential(maxDeepth)

	mapReferential.Get(inputHash)
	arrayReferential.Get(inputHash)

}

func GetHash(word string) [32]byte {
	return sha256.Sum256([]byte(word))
}
