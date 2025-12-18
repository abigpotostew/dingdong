package main

import (
	"log"

	"github.com/abigpotostew/dingdong/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
