package main

import (
	"fmt"
	"os"
)

func main() {
	mode := os.Getenv("FAIL_MODE")
	if mode == "once" {
		fmt.Println("flaky: fallo simulado")
		os.Exit(1)
	}
	fmt.Println("flaky: ejecución correcta")
}
