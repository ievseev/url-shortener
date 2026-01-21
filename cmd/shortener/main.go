package main

import (
	"os"

	"github.com/ievseev/url-shortener/internal/app"
)

const fail = 1

func main() {
	if err := app.Run(); err != nil {
		os.Exit(fail)
	}
}
