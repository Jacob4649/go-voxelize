package main

import (
	"os"
	"strconv"

	"github.com/Jacob4649/go-voxelize/go-voxelize/lasProcessing"
	"github.com/Jacob4649/go-voxelize/go-voxelize/lidarioMod"
	"github.com/Jacob4649/go-voxelize/go-voxelize/voxels"
)

// Arguments passed to run the program
type executionArgs struct {
	
	// input file name
	fileName string

	// output file name
	destName string

	// concurrency to use
	concurrency int

	// number of chunks
	chunkNumber int

	// density to use
	density int

	// voxel size to use
	voxelSize float64
}

// parses the specified arguments
func parseArgs(args []string) executionArgs {

	if len(args) < 3 || len(args) > 3 && len(args) < 7 {
		println("Usage: voxelize <input LAS path> <output CSV path> <optional chunk number> <optional concurrency> " +
			"<optional density> <optional voxel size>")
		println("Must specify chunk number, concurrency, density, and voxel size to use any")
		println("Default chunk number: 256")
		println("Default concurrency: 32")
		println("Default density: 20")
		println("Default voxel size: 0.1")
		os.Exit(1)
	}

	fileName := args[1]

	destName := args[2]

	concurrency := 32

	chunkNumber := 256

	density := 20

	voxelSize := 0.1

	if len(args) >= 7 {
		chunkString := args[3]
		concurrencyString := args[4]
		densityString := args[5]
		voxelSizeString := args[6]

		var err1 error
		var err2 error
		var err3 error
		var err4 error

		chunkNumber, err1 = strconv.Atoi(chunkString)
		concurrency, err2 = strconv.Atoi(concurrencyString)
		density, err3 = strconv.Atoi(densityString)
		voxelSize, err4 = strconv.ParseFloat(voxelSizeString, 64)

		if err1 != nil || err2 != nil || err3 != nil || err4 != nil || chunkNumber <= 0 || concurrency <= 2 ||
			density <= 1 || voxelSize <= 0 {
			print("Error parsing chunk number, concurrency, or density. " +
				"All must be integers (chunk number > 0, concurrency > 2, density > 1, voxel size > 0)")
			os.Exit(1)
		}
	}

	return executionArgs{fileName: fileName, destName: destName, 
		concurrency: concurrency, chunkNumber: chunkNumber, density: density, voxelSize: voxelSize}
}

// performs the main processing of the LAS file
func mainProcessing[O any](file *lidarioMod.LasFile, processor lasProcessing.LASProcessor[O], config executionArgs) *O {
	chunks := lasProcessing.ChunkFile(file, config.chunkNumber)

	status := lasProcessing.NewConcurrentStatus()

	quit := false

	uiDone := make(chan bool)

	go lasProcessing.CLIStatus(status, &quit, uiDone)

	output := lasProcessing.ConcurrentProcess(file, chunks, processor, config.concurrency, status)

	quit = true

	<- uiDone

	return output
}

// post processes the resulting voxels
func postProcessing[I any, O any](voxels I, pipeline lasProcessing.PostProcessingPipeline[I, O], config executionArgs) O {
	pipelineStatus := &lasProcessing.PipelineStatus{}

	quit := false

	uiDone := make(chan bool)

	go lasProcessing.PostProcessingStatus(pipelineStatus, &quit, uiDone)

	err := lasProcessing.ProcessWithPipeline(voxels, pipeline, pipelineStatus)

	quit = true

	<- uiDone

	return err
}

// Main function
func main() {

	config := parseArgs(os.Args)

	file, err := lidarioMod.NewLasFile(config.fileName, "rh")

	if err != nil {
		println("Error accessing LAS file")
		os.Exit(1)
	}

	// main processing

	processor := voxels.DensityVoxelSetProcessor{PointDensity: config.density, VoxelSize: config.voxelSize}

	output := mainProcessing[voxels.DensityVoxelSet](file, &processor, config)

	// post processing

	pipeline := lasProcessing.ChainPipeline[*voxels.DensityVoxelSet, *voxels.VoxelSet, error](
		&voxels.VoxelCondenser{Density: config.density}, &voxels.VoxelFileWriter{FileName: config.destName})

	err = postProcessing(output, pipeline, config)

	// processing finished
	
	if err != nil {
		println("Error writing to csv")
		os.Exit(1)
	}

	println("Complete")
}