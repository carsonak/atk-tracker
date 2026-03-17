package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/carsonak/atk-tracker/internal/utils"
)

func main() {
	files, err := filepath.Glob("/dev/input/event*")
	if err != nil {
		panic(err)
	}

	var validDevices []string

	for _, filename := range files {
		file, err := os.Open(filename)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}

		res, err := utils.IsActivityDevice(file)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}

		file.Close()
		if res {
			validDevices = append(validDevices, filename)
		}
	}

	for _, dev := range validDevices {
		fmt.Println(dev)
	}
}
