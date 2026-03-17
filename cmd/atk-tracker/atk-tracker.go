package main

import (
	"fmt"
	"log"
	"os"

	"github.com/carsonak/atk-tracker/internal/HIDevent"
)

func main() {
	validDevices, errs := HIDevent.GetHIDHandlers()
	if len(errs) > 0 {
		for _, e := range errs {
			log.Println(e)
		}

		os.Exit(1)
	}

	for _, dev := range validDevices {
		fmt.Println(dev)
	}
}
