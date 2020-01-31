package main

import (
	"log"

	"github.com/alexej-v/grpc_cli/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
