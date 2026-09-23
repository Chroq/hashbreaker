package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
)

func main() {
	word := flag.String("word", "", "Mot à hasher (ou passé en premier argument)")
	flag.Parse()

	target := *word
	if target == "" && len(flag.Args()) > 0 {
		target = flag.Arg(0)
	}
	if target == "" {
		target = "@kAl1"
	}

	h := sha256.Sum256([]byte(target))
	fmt.Printf("%x\n", h)
}
