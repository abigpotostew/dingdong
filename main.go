package main

import (
	"log"

	"github.com/abigpotostew/stewstats/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
