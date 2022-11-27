package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/Jacob4649/go-voxelize/go-voxelize/lasProcessing"
	"github.com/Jacob4649/go-voxelize/go-voxelize/lidarioMod"
	"github.com/Jacob4649/go-voxelize/go-voxelize/voxels"
)

// Writes a set of voxels to a file
func writeToFile(voxels *voxels.VoxelSet, fileName string) error {
	file, err := os.Create(fileName)

	if err != nil {
		return err
	}

	defer file.Close()

	file.WriteString("x,y,z\n")

	total := voxels.Voxels.Cardinality()

	current := 0

	println(lasProcessing.ProgressBarInt("Writing", current, total))

	for voxel := range voxels.Voxels.Iterator().C {
		_, err = file.WriteString(fmt.Sprint(voxel.X) + "," +fmt.Sprint(voxel.Y) + "," + fmt.Sprint(voxel.Z) + "\n")

		if err != nil {
			return err
		}

		current += 1
		if current % 500 == 0 {
			print("\033[A\r")
			println(lasProcessing.ProgressBarInt("Writing", current, total))
		}
	}

	print("\033[A\033[2K\r")
	println("Finished writing")

	return nil
}

// Main function
func main() {

	if len(os.Args) < 3 {
		println("Usage: voxelize <input LAS path> <output CSV path> <optional chunk number> <optional concurrency>")
		println("Must specify both chunk number and concurrency to use either")
		println("Default chunk number: 256")
		println("Default concurrency: 32")
		return
	}

	fileName := os.Args[1]

	destName := os.Args[2]

	concurrency := 32

	chunkNumber := 256

	if len(os.Args) >= 5 {
		chunkString := os.Args[3]
		concurrencyString := os.Args[4]

		var err error
		chunkNumber, err = strconv.Atoi(chunkString)
		concurrency, err = strconv.Atoi(concurrencyString)

		if err != nil || chunkNumber <= 0 || concurrency <= 2 {
			print("Error parsing chunk number or concurrency, both must be integers (chunk number > 0, concurrency > 2)")
			return
		}
	}

	file, err := lidarioMod.NewLasFile(fileName, "rh")

	if err != nil {
		println("Error accessing LAS file")
		return
	}

	chunks := lasProcessing.ChunkFile(file, chunkNumber)

	processor := voxels.VoxelSetProcessor{}

	status := lasProcessing.NewConcurrentStatus()

	quit := false

	uiDone := make(chan bool)

	go lasProcessing.CLIStatus(status, &quit, uiDone)

	output := lasProcessing.ConcurrentProcess[voxels.VoxelSet](file, chunks, &processor, concurrency, 0.1, status)

	quit = true

	<- uiDone

	err = writeToFile(output, destName)

	if err != nil {
		println("Error writing to csv")
		return
	}

	println("Written to csv")
}