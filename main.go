package main

import (
	"os"
	"strconv"

	"github.com/Jacob4649/go-voxelize/go-voxelize/lasProcessing"
	"github.com/Jacob4649/go-voxelize/go-voxelize/lidarioMod"
	"github.com/Jacob4649/go-voxelize/go-voxelize/voxels"
)

// Main function
func main() {

	if len(os.Args) < 3 || len(os.Args) > 3 && len(os.Args) < 7 {
		println("Usage: voxelize <input LAS path> <output CSV path> <optional chunk number> <optional concurrency> <optional density> <optional voxel size>")
		println("Must specify chunk number, concurrency, density, and voxel size to use any")
		println("Default chunk number: 256")
		println("Default concurrency: 32")
		println("Default density: 20")
		println("Default voxel size: 0.1")
		os.Exit(1)
	}

	fileName := os.Args[1]

	destName := os.Args[2]

	concurrency := 32

	chunkNumber := 256

	density := 20

	voxelSize := 0.1

	if len(os.Args) >= 7 {
		chunkString := os.Args[3]
		concurrencyString := os.Args[4]
		densityString := os.Args[5]
		voxelSizeString := os.Args[6]

		var err1 error
		var err2 error
		var err3 error
		var err4 error

		chunkNumber, err1 = strconv.Atoi(chunkString)
		concurrency, err2 = strconv.Atoi(concurrencyString)
		density, err3 = strconv.Atoi(densityString)
		voxelSize, err4 = strconv.ParseFloat(voxelSizeString, 64)

		if err1 != nil || err2 != nil || err3 != nil || err4 != nil || chunkNumber <= 0 || concurrency <= 2 || density <= 1 || voxelSize <= 0 {
			print("Error parsing chunk number, concurrency, or density. All must be integers (chunk number > 0, concurrency > 2, density > 1, voxel size > 0)")
			os.Exit(1)
		}
	}

	file, err := lidarioMod.NewLasFile(fileName, "rh")

	if err != nil {
		println("Error accessing LAS file")
		os.Exit(1)
	}

	chunks := lasProcessing.ChunkFile(file, chunkNumber)

	processor := voxels.DensityVoxelSetProcessor{PointDensity: density, VoxelSize: voxelSize}

	status := lasProcessing.NewConcurrentStatus()

	quit := false

	uiDone := make(chan bool)

	go lasProcessing.CLIStatus(status, &quit, uiDone)

	output := lasProcessing.ConcurrentProcess[voxels.DensityVoxelSet](file, chunks, &processor, concurrency, status)

	quit = true

	<- uiDone

	// post processing

	pipelineStatus := &lasProcessing.PipelineStatus{}

	quit = false

	uiDone = make(chan bool)

	go lasProcessing.PostProcessingStatus(pipelineStatus, &quit, uiDone)

	pipeline := lasProcessing.ChainPipeline[*voxels.DensityVoxelSet, *voxels.VoxelSet, error](&voxels.VoxelCondenser{Density: density}, &voxels.VoxelFileWriter{FileName: destName})

	err = lasProcessing.ProcessWithPipeline(output, pipeline, pipelineStatus)

	quit = true

	<- uiDone

	// processing finished
	
	if err != nil {
		println("Error writing to csv")
		os.Exit(1)
	}

	println("Complete")
}