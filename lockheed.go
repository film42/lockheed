package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func execCommand(command string) string {
	output, err := exec.Command("bash", "-c", command).Output()
	if err != nil {
		fmt.Println("Command '", command, "' returned error:", err)
	}

	return string(output)
}

func execCommandAndReport(command string, reportChannel chan int) {
	execCommand(command)
	reportChannel <- 1
}

func isConnectedToVPN() bool {
	_, err := os.Stat("/proc/sys/net/ipv4/conf/tun0")
	if err != nil {
		return true
	}

	return false
}

func listenForAudioSource(inputChannel chan int, audioSources []string) {
	for {
		output := execCommand("pacmd list-sink-inputs")
		for _, audioSource := range audioSources {
			if strings.Contains(output, audioSource) {
				inputChannel <- 1
			}
		}

		// This check doesn't need to happen very often.
		time.Sleep(time.Second * 15)
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

func spawnForDevices(inputChannel chan int) {
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
}

func spawnForAudioSources(inputChannel chan int) {
	audioSources := []string{
		`application.name = "Chromium"`,
	}
	fmt.Println("Listening for the following audio sources:")
	for _, audioSource := range audioSources {
		fmt.Println("-", audioSource)
	}
	go listenForAudioSource(inputChannel, audioSources)
}

func main() {
	lockCommandPtr := flag.String("locker", "pgrep -f i3lock || i3lock", "Command to execute your screen lock.")
	lockTimePtr := flag.Uint("time", 5, "Minutes of idle time before locking.")
	notifierCommandPtr := flag.String("notifier", "notify-send -u critical -t 10000 -- 'Locking screen soon.'", "Command to execute your screen lock.")
	notifyTimePtr := flag.Uint("notify", 30, "Seconds before locking when a notification is sent.")
	suspendCommandPtr := flag.String("suspender", "systemctl suspend", "Command for suspending computer.")
	suspendTimePtr := flag.Uint("suspend", 15, "Minutes of idle time before suspending.")
	suspendDisabledPtr := flag.Bool("suspend-disabled", false, "Don't over suspend.")
	suspendDisabledWhileOnVPNPtr := flag.Bool("suspend-disabled-while-on-vpn", true, "Don't engage suspend if we're conected to a VPN.")
	flag.Parse()

	inputChannel := make(chan int)
	spawnForDevices(inputChannel)
	spawnForAudioSources(inputChannel)

	// Main loop
	lockTime := time.Duration(*lockTimePtr) * time.Minute
	notifyTime := time.Duration(lockTime - (time.Duration(*notifyTimePtr) * time.Second))
	lockTimer := time.NewTimer(lockTime)
	notifyTimer := time.NewTimer(notifyTime)
	suspendTime := time.Duration(*suspendTimePtr) * time.Minute
	suspendTimer := time.NewTimer(suspendTime)
	currentlyLocked := false
	lockFinishedChannel := make(chan int)

	fmt.Println("Timer settings:")
	fmt.Println("- Notify after", notifyTime)
	fmt.Println("- Lock after", lockTime)
	if *suspendDisabledPtr {
		fmt.Println("- Suspend disabled: true")
	} else {
		fmt.Println("- Suspend after", suspendTime)
		fmt.Println("- Suspend disabled while connected to VPN:", *suspendDisabledWhileOnVPNPtr)
	}

	fmt.Println("Starting wait/lock/notify/suspend loop...")
	for {
		select {
		case <- inputChannel:
			// Await any device or sound input.
			if !currentlyLocked {
				lockTimer.Reset(lockTime)
				notifyTimer.Reset(notifyTime)
			}
			suspendTimer.Reset(suspendTime)
		case <- lockFinishedChannel:
			// Reset lock state when lock exec completes.
			fmt.Println("Locker finished!")
			currentlyLocked = false
			lockTimer.Reset(lockTime)
			notifyTimer.Reset(notifyTime)
			suspendTimer.Reset(suspendTime)
		case <- notifyTimer.C:
			// Notfication timer has fired.
			fmt.Println("Notifying about locking soon!")
			go execCommand(*notifierCommandPtr)
		case <- lockTimer.C:
			// Lock timer has fired.
			fmt.Println("Locking computer!")
			currentlyLocked = true
			go execCommandAndReport(*lockCommandPtr, lockFinishedChannel)
		case <- suspendTimer.C:
			// Suspend timer has fired.
			if *suspendDisabledPtr {
				continue
			}

			// If we're connected to a VPN then don't suspend.
			if *suspendDisabledWhileOnVPNPtr && isConnectedToVPN() {
				fmt.Println("Skipping suspend because VPN is connected")
				continue
			}

			fmt.Println("Suspending computer!")
			execCommand(*suspendCommandPtr)
			suspendTimer.Reset(suspendTime)
		}
	}
}
