package main

import (
	"net/http"
	"os"

	"github.com/ievseev/url-shortener/internal/config"
)

const fail = 1

func main() {
	if err := run(); err != nil {
		os.Exit(fail)
	}
}

func run() error {
	appConfig := config.Must()

	err := http.ListenAndServe(appConfig.Host+":"+appConfig.Port, nil)
	if err != nil {
		return err
	}
	// init configs

	// make repo, service, handler

	// run server
	return nil
}
