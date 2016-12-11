package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"
	"path/filepath"
	"flag"
)

func execCommand(command string) {
	err := exec.Command("bash", "-c", command).Run()
	if err != nil {
		fmt.Println("Command '", command, "' returned error:", err)
	}
}

func listenForEvents(inputDeviceChannel chan int, path string, size int) {
	file, err := os.Open(path)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for {
		bytes := make([]byte, size)
		_, err := file.Read(bytes)
		if err != nil {
			fmt.Println("Error attempting to read N bytes from", path, ":", err)
			continue
		}

		inputDeviceChannel <- 1
	}
}

func main() {
	lockCommandPtr := flag.String("locker", "i3lock", "Command to execute your screen lock.")
	waitTimePtr := flag.Uint("time", 5, "Minutes of idle time before locking.")
	flag.Parse()


	inputDeviceChannel := make(chan int)
	// Add all devices under /dev/input/by-id because these seem to only list
	// the input devices I'm looking for. Maybe this should look for "mouse" or
	// "keyboard" as well? Maybe make this a cli flag?
	// The only error if for a bad format.
	paths, _ := filepath.Glob("/dev/input/by-id/*")
	fmt.Println("Listening to the following input devices:")
	for _, path := range paths {
		fmt.Println("-", path)
		go listenForEvents(inputDeviceChannel, path, 24)
	}


	fmt.Println("Starting wait/lock loop...")
	waitTime := time.Duration(*waitTimePtr)
	timeout := time.NewTimer(time.Second * waitTime)
	for {
		select {
		case <- inputDeviceChannel:
			timeout.Reset(time.Second * waitTime)
		case <- timeout.C:
			// lock screen
			fmt.Println("Locking computer!")
			execCommand(*lockCommandPtr)
		}
	}
}
