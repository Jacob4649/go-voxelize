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
		if *progress != 1.0 {
			println("\033[2K\rP" + ProgressBarFloat("P" + fmt.Sprint(i), *progress))
		} else {
			println("\033[2K\rP" + fmt.Sprint(i) + ":\tMerging")
		}
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

// Writes pipeline status to the screen
func pipelineStatus(status *PipelineStatus, prevStep string) bool {
	if status.Step != prevStep && prevStep != "" {
		println("Finished " + strings.ToLower(prevStep))
	}
	
	if status.Step == "" {
		if prevStep != "" {
			// finished
			println("Finished post processing")	
		}
		return false
	}

	println(ProgressBarFloat(status.Step, status.Progress))
	return true
}

// Displays the status of a PostProcessingStatus in the console
func PostProcessingStatus(status *PipelineStatus, quit *bool, uiDone chan<- bool) {
	prevStep := status.Step
	prevWrite := pipelineStatus(status, prevStep)
	for !*quit {
		time.Sleep(200 * time.Millisecond)
		if prevWrite {
			print("\033[A\033[2K\r")
		}
		prevWrite = pipelineStatus(status, prevStep)
		prevStep = status.Step
	}
	pipelineStatus(status, prevStep)
	uiDone <- true
}