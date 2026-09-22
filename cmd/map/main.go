package main

import (
	"os"
	"strconv"

	"github.com/Chroq/HashBreaker/pkg"
)

func main() {
	targetWord := os.Args[1]

	// inputHash := pkg.GetHash(targetWord)
	depth := len(targetWord)
	if len(os.Args) > 2 {
		if d, err := strconv.Atoi(os.Args[2]); err == nil && d > 0 {
			depth = d
		}
	}

	pkg.NewMapReferential(depth)

	/*
		mapRef := pkg.NewMapReferential(depth)

		found, ok := mapRef.Get(inputHash)
				if ok {
					fmt.Printf("Trouvé : %s\n", found)
				} else {
					fmt.Println("Non trouvé")
				}
	*/
}
