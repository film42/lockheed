package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"
	"path/filepath"
	"flag"
	"strings"
)

func execCommand(command string) string {
	output, err := exec.Command("bash", "-c", command).Output()
	if err != nil {
		fmt.Println("Command '", command, "' returned error:", err)
	}

	return string(output)
}

func listenForAudioSource(inputChannel chan int, audioSources []string) {
	for {
		output := execCommand("pacmd list-sink-inputs")
		for _, audioSource := range audioSources {
			if strings.Contains(output, audioSource) {
				inputChannel <- 1
			}
		}

		// This check doesn't need to happen at least once per minute.
		time.Sleep(time.Second * 1)
	}
}

func listenForEvents(inputChannel chan int, path string, size int) {
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

		inputChannel <- 1
	}
}

func main() {
	lockCommandPtr := flag.String("locker", "i3lock", "Command to execute your screen lock.")
	waitTimePtr := flag.Uint("time", 5, "Minutes of idle time before locking.")
	flag.Parse()


	inputChannel := make(chan int)
	// Add all devices under /dev/input/by-id because these seem to only list
	// the input devices I'm looking for. Maybe this should look for "mouse" or
	// "keyboard" as well? Maybe make this a cli flag?
	// The only error if for a bad format.
	paths, _ := filepath.Glob("/dev/input/by-id/*")
	fmt.Println("Listening to the following input devices:")
	for _, path := range paths {
		fmt.Println("-", path)
		go listenForEvents(inputChannel, path, 24)
	}

	audioSources := []string{
		`application.name = "Chromium"`,
	}
	fmt.Println("Listening for the following audio sources:")
	for _, audioSource := range audioSources {
		fmt.Println("-", audioSource)
	}
	go listenForAudioSource(inputChannel, audioSources)

	fmt.Println("Starting wait/lock loop...")
	waitTime := time.Duration(*waitTimePtr)
	timeout := time.NewTimer(time.Second * waitTime)
	for {
		select {
		case <- inputChannel:
			timeout.Reset(time.Second * waitTime)
		case <- timeout.C:
			// lock screen
			fmt.Println("Locking computer!")
			execCommand(*lockCommandPtr)
		}
	}
}
