package main

import (
	"flag"
	"os"

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

	// whether to normalize
	normalize bool

	// whether to convert to a gradient
	gradient bool

	// where to output minimum heights
	minimumImagePath string
}

// parses the specified arguments
func parseArgs() executionArgs {
	
	destName := flag.String("output", "output.csv", "the file to output results to")

	concurrency := flag.Int("concurrency", 32, "how many concurrent reads and processes to use")

	chunkNumber := flag.Int("chunks", 256, "how many chunks to split the file into")

	density := flag.Int("density", 20, "point density in a voxel to be filled")

	voxelSize := flag.Float64("voxel", 0.1, "side length for a voxel")

	normalize := flag.Bool("normalize", false, "whether to normalize output")

	gradient := flag.Bool("gradient", false, "whether to convert output to a height gradient")

	minimumImagePath := flag.String("minimum-output", "", "file path to output the minimums image")

	flag.Parse()

	fileName := flag.Arg(0)

	if fileName == "" {
		print("must define an output file")
		os.Exit(0)
	}

	return executionArgs{fileName: fileName, destName: *destName, 
		concurrency: *concurrency, chunkNumber: *chunkNumber, density: *density, voxelSize: *voxelSize,
		normalize: *normalize, gradient: *gradient, minimumImagePath: *minimumImagePath}
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

/// selects a post processing pipeline to use
func choosePipeline(config executionArgs) lasProcessing.PostProcessingPipeline[*voxels.DensityVoxelSet, error] {
	var finalPipeline lasProcessing.PostProcessingPipeline[*voxels.DensityVoxelSet, error]
	
	var voxelPipeline lasProcessing.PostProcessingPipeline[*voxels.DensityVoxelSet, *voxels.VoxelSet] = 
		&voxels.VoxelCondenser{Density: config.density}

	outputMinimums := config.minimumImagePath != ""

	minimumPipeline := &voxels.MinimumHeightFinder{
		OuptutMinimums: outputMinimums,
		OutputFile: config.minimumImagePath,
	}

	if config.normalize {
		heightPipeline := lasProcessing.ChainPipeline[*voxels.DensityVoxelSet, *voxels.VoxelSet, *voxels.MinimumHeights](
			voxelPipeline, minimumPipeline)
	
		voxelPipeline = lasProcessing.ChainPipeline[*voxels.DensityVoxelSet, *voxels.MinimumHeights, *voxels.VoxelSet](
			heightPipeline, &voxels.LazyNormalizer{})
	} else if outputMinimums {
		heightPipeline := lasProcessing.ChainPipeline[*voxels.DensityVoxelSet, *voxels.VoxelSet, *voxels.MinimumHeights](
			voxelPipeline, minimumPipeline)
	
		voxelPipeline = lasProcessing.ChainPipeline[*voxels.DensityVoxelSet, *voxels.MinimumHeights, *voxels.VoxelSet](
			heightPipeline, &voxels.MinimumDegrouper{})
	}

	finalPipeline = lasProcessing.ChainPipeline[*voxels.DensityVoxelSet, *voxels.VoxelSet, error](
		voxelPipeline, &voxels.VoxelFileWriter{FileName: config.destName})

	if config.gradient {
		gradientPipline := lasProcessing.ChainPipeline[*voxels.DensityVoxelSet, *voxels.VoxelSet, *voxels.HeightGradient](
			voxelPipeline, &voxels.GradientProcessor{})

		finalPipeline = lasProcessing.ChainPipeline[*voxels.DensityVoxelSet, *voxels.HeightGradient, error](
			gradientPipline, &voxels.GradientFileWriter{FileName: config.destName})
	}

	return finalPipeline
}

// Main function
func main() {

	config := parseArgs()

	file, err := lidarioMod.NewLasFile(config.fileName, "rh")

	if err != nil {
		println("Error accessing LAS file")
		os.Exit(1)
	}

	// main processing

	processor := voxels.DensityVoxelSetProcessor{PointDensity: config.density, VoxelSize: config.voxelSize}

	output := mainProcessing[voxels.DensityVoxelSet](file, &processor, config)

	// post processing

	pipeline := choosePipeline(config)

	err = postProcessing(output, pipeline, config)

	// processing finished
	
	if err != nil {
		println("Error writing to csv")
		os.Exit(1)
	}

	println("Complete")
}