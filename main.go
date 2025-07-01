package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

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

	// whether to convert to measurements
	measurements bool

	// where to output minimum heights
	minimumImagePath string

	// whether to split into sources
	splitSources bool

	// scanner locations
	scannerLocations string
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

	measurements := flag.Bool("measurements", false, "whether to convert output to measurements")

	minimumImagePath := flag.String("minimum-output", "", "file path to output the PNG minimums image")

	splitSources := flag.Bool("split-sources", false, "whether to split input by source before processing")

	scannerLocations := flag.String("scanner-locations", "", "comma-separated list of scanner x,y,z coordinates (e.g., \"x1,y1,z1,x2,y2,z2\")")

	flag.Parse()

	fileName := flag.Arg(0)

	if fileName == "" {
		print("must define an input file")
		os.Exit(0)
	}

	return executionArgs{fileName: fileName, destName: *destName, 
		concurrency: *concurrency, chunkNumber: *chunkNumber, density: *density, voxelSize: *voxelSize,
		normalize: *normalize, gradient: *gradient, minimumImagePath: *minimumImagePath, splitSources: *splitSources,
		measurements: *measurements, scannerLocations: *scannerLocations}
}

// helper function to parse scanner locations
func parseScannerLocations(locationsString string, voxelSize float64) ([]voxels.Coordinate, error) {
	locations := []voxels.Coordinate{}
	if locationsString == "" {
		return locations, nil
	}

	parts := strings.Split(locationsString, ",")
	if len(parts)%3 != 0 {
		return nil, fmt.Errorf("scanner locations string must have a number of values divisible by 3 (x,y,z triplets)")
	}

	for i := 0; i < len(parts); i += 3 {
		x, errX := strconv.ParseFloat(parts[i], 64)
		y, errY := strconv.ParseFloat(parts[i+1], 64)
		z, errZ := strconv.ParseFloat(parts[i+2], 64)

		if errX != nil || errY != nil || errZ != nil {
			return nil, fmt.Errorf("error parsing scanner location coordinates: %v, %v, %v", errX, errY, errZ)
		}
		// Convert to voxel coordinates. Assuming 0,0,0 min for now, this might need adjustment
		// based on how global vs local coordinates are handled for scanners.
		// For now, directly convert.
		coord := voxels.PointToCoordinate(x, 0, y, 0, z, 0, voxelSize, false)
		locations = append(locations, coord)
	}
	return locations, nil
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

/// selects a post processing pipeline to use for density voxels
func chooseDensityVoxelPipeline(config executionArgs, scannerCoords []voxels.Coordinate) lasProcessing.PostProcessingPipeline[*voxels.DensityVoxelSet, error] {
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

	// Insert OcclusionProcessor after VoxelCondenser (which is the last step in current voxelPipeline)
	// The scannerCoords are parsed in main and passed to processDensityVoxels, then to here via config.
	// We need to get scannerCoords into this scope.
	// For now, let's assume scannerCoords can be accessed via config or passed differently.
	// Modifying chooseDensityVoxelPipeline to accept scannerCoords.
	// scannerCoords are now passed as a parameter.

	occlusionAwarePipeline := lasProcessing.ChainPipeline[*voxels.DensityVoxelSet, *voxels.VoxelSet, *voxels.VoxelSet](
		voxelPipeline,
		&voxels.OcclusionProcessor{
			ScannerLocations: scannerCoords, // Use the passed-in scannerCoords
			VoxelSize:        config.voxelSize,
		},
	)

	// The rest of the pipeline now chains from occlusionAwarePipeline
	finalPipeline = lasProcessing.ChainPipeline[*voxels.DensityVoxelSet, *voxels.VoxelSet, error](
		occlusionAwarePipeline, &voxels.VoxelFileWriter{FileName: config.destName})

	if config.gradient {
		gradientPipline := lasProcessing.ChainPipeline[*voxels.DensityVoxelSet, *voxels.VoxelSet, *voxels.HeightGradient](
			occlusionAwarePipeline, &voxels.GradientProcessor{})

		finalPipeline = lasProcessing.ChainPipeline[*voxels.DensityVoxelSet, *voxels.HeightGradient, error](
			gradientPipline, &voxels.GradientFileWriter{FileName: config.destName})
	} else if config.measurements {
		measurementPipeline := lasProcessing.ChainPipeline[*voxels.DensityVoxelSet, *voxels.VoxelSet, *voxels.Measurements](
			occlusionAwarePipeline, &voxels.MeasurementFinder{})

		finalPipeline = lasProcessing.ChainPipeline[*voxels.DensityVoxelSet, *voxels.Measurements, error](
			measurementPipeline, &voxels.MeasurementsFileWriter{FileName: config.destName})
	}

	return finalPipeline
}

// processes some density voxels and outputs an error
func processDensityVoxels(file *lidarioMod.LasFile, config executionArgs, scannerCoords []voxels.Coordinate) error {
		// main processing

		processor := voxels.DensityVoxelSetProcessor{
			PointDensity:     config.density,
			VoxelSize:        config.voxelSize,
			ScannerLocations: scannerCoords, // Pass parsed scanner coordinates
		}

		output := mainProcessing[voxels.DensityVoxelSet](file, &processor, config)
	
		// post processing
	
		pipeline := chooseDensityVoxelPipeline(config, scannerCoords) // Pass scannerCoords here
	
		return postProcessing(output, pipeline, config)
}

// makes pipelines for processing density voxel sets from different sources
// Also needs scannerCoords if chooseDensityVoxelPipeline is called within.
func makeSourcesPipelines(sets []*voxels.DensityVoxelSet, config executionArgs, scannerCoords []voxels.Coordinate) ([]executionArgs, []lasProcessing.PostProcessingPipeline[*voxels.DensityVoxelSet, error]) {
	configs := make([]executionArgs, 0)

	for i, _ := range sets {
		copy := config
		copy.destName = fmt.Sprint(i) + "-" + copy.destName
		if copy.minimumImagePath != "" {
			copy.minimumImagePath = fmt.Sprint(i) + "-" + copy.minimumImagePath
		}
		configs = append(configs, copy)
	}

	pipelines := make([]lasProcessing.PostProcessingPipeline[*voxels.DensityVoxelSet, error], 0)

	for _, pipelineConfig := range configs {
		// Each source might have its own scanner context if that was a feature,
		// but for now, use the global scannerCoords for all.
		pipelines = append(pipelines, chooseDensityVoxelPipeline(pipelineConfig, scannerCoords))
	}

	return configs, pipelines
}

// processes voxels by source and outputs an error
func processSources(file *lidarioMod.LasFile, config executionArgs, scannerCoords []voxels.Coordinate) {
	// main processing
	
	processor := voxels.PointSourceProcessor{
		PointDensity:     config.density,
		VoxelSize:        config.voxelSize,
		ScannerLocations: scannerCoords, // Pass parsed scanner coordinates
	}

	output := mainProcessing[voxels.PointSourceDensityVoxelSet](file, &processor, config)

	// split into sets
	
	splitPipeline := voxels.PointSourceSplitter{}

	sets := postProcessing[*voxels.PointSourceDensityVoxelSet, []*voxels.DensityVoxelSet](output, &splitPipeline, config)

	// concurrent post processing

	configs, pipelines := makeSourcesPipelines(sets, config, scannerCoords) // Pass scannerCoords here

	for i, pipeline := range pipelines {
		set := sets[i]
		pipelineConfig := configs[i]
		println("Processing source " + fmt.Sprint(i))

		postProcessing(set, pipeline, pipelineConfig)

		println("Completed processing source " + fmt.Sprint(i))
	}

}

// Main function
func main() {

	config := parseArgs()

	scannerCoords, err := parseScannerLocations(config.scannerLocations, config.voxelSize)
	if err != nil {
		fmt.Printf("Error parsing scanner locations: %v\n", err)
		os.Exit(1)
	}

	// For debugging, print the parsed scanner coordinates
	if len(scannerCoords) > 0 {
		fmt.Printf("Parsed scanner locations (voxel coordinates): %+v\n", scannerCoords)
	}

	file, err := lidarioMod.NewLasFile(config.fileName, "rh")

	if err != nil {
		println("Error accessing LAS file")
		os.Exit(1)
	}

	if config.splitSources {
		processSources(file, config, scannerCoords)
	} else {
		err = processDensityVoxels(file, config, scannerCoords)
	}

	// processing finished
	
	if err != nil {
		println("Error writing to csv")
		os.Exit(1)
	}

	println("Complete")
}