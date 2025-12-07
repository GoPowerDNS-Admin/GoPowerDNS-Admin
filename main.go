package main

import (
	"os"

	"github.com/GoPowerDNS-Admin/GoPowerDNS-Admin/app"
)

func main() {
	err := app.Execute()
	if err != nil {
		os.Exit(1)
	}
}
