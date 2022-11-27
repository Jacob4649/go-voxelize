package lasProcessing

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// Displays a progress bar at the specified progress
func progressBarRaw(width int, progress float64) string {
	active := int(float64(width) * progress)
	inactive := width - active
	return "|" + strings.Repeat("=", active) + strings.Repeat("-", inactive) + "|"
}

// Displays a progress bar using an int
func ProgressBarInt(name string, progress int, maxProgress int) string {
	return name + ":\t" + progressBarRaw(80, float64(progress) / float64(maxProgress)) + fmt.Sprintf("(%d / %d)", progress, maxProgress)
}

// Displays a progress bar using a float
func ProgressBarFloat(name string, progress float64) string {
	return name + ":\t" + progressBarRaw(80, progress) + fmt.Sprintf("(%f", math.Round(progress * 10000) / 100) + "%)"
}

// Prints the CLI status to the console
func cliStatusPanel(status *ConcurrentStatus) {
	println(ProgressBarInt("CKS", *(status.CurrentChunk), *(status.TotalChunks)));
	for i, progress := range status.ChunkProgress {
		println(ProgressBarFloat("P" + fmt.Sprint(i), *progress))
	}
}

// Displays the status of a ConcurrentStatus in the console
func CLIStatus(status *ConcurrentStatus, quit *bool, uiDone chan<- bool) {
	cliStatusPanel(status)
	for !*quit {
		time.Sleep(200 * time.Millisecond)
		print(strings.Repeat("\033[A", *(status.Concurrency) + 1) + "\r")
		cliStatusPanel(status)
	}
	print(strings.Repeat("\033[A\033[2K\r", *(status.Concurrency) + 1))
	println("Finished processing")
	uiDone <- true
}