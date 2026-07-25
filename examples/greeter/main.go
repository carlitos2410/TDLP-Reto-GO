package main

import (
	"fmt"
	"os"
)

func main() {
	greeting := os.Getenv("GREETING")
	if greeting == "" {
		greeting = "Hola"
	}

	for i := 0; i < 5; i++ {
		fmt.Printf("%s (%d/5)\n", greeting, i+1)
	}
}
