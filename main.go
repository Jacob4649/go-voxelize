package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/Jacob4649/go-voxelize/go-voxelize/lasProcessing"
	"github.com/Jacob4649/go-voxelize/go-voxelize/lidarioMod"
	"github.com/Jacob4649/go-voxelize/go-voxelize/voxels"
	mapset "github.com/deckarep/golang-set/v2"
)

// Post processes a DensityVoxelSet into a VoxelSet
func postProcess(densityVoxels *voxels.DensityVoxelSet) *voxels.VoxelSet {

	voxelSet := mapset.NewThreadUnsafeSet[voxels.Coordinate]()

	total := len(densityVoxels.Voxels)

	current := 0

	println(lasProcessing.ProgressBarInt("Post", current, total))

	for voxel, density := range densityVoxels.Voxels {

		if density >= densityVoxels.PointDensity {
			voxelSet.Add(voxel)
		}

		current += 1
		if current % 500 == 0 {
			print("\033[A\r")
			println(lasProcessing.ProgressBarInt("Post", current, total))
		}
	}

	print("\033[A\033[2K\r")
	println("Finished postprocessing")

	output := &voxels.VoxelSet{XSize: densityVoxels.XSize, 
		YSize: densityVoxels.YSize,
		ZSize: densityVoxels.ZSize,
		XVoxels: densityVoxels.XVoxels,
		YVoxels: densityVoxels.YVoxels,
		ZVoxels: densityVoxels.ZVoxels,
		Voxels: voxelSet}

	return output
}

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

	if len(os.Args) < 3 || len(os.Args) > 3 && len(os.Args) < 7 {
		println("Usage: voxelize <input LAS path> <output CSV path> <optional chunk number> <optional concurrency> <optional density> <optional voxel size>")
		println("Must specify chunk number, concurrency, density, and voxel size to use any")
		println("Default chunk number: 256")
		println("Default concurrency: 32")
		println("Default density: 20")
		println("Default voxel size: 0.1")
		return
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

		chunkNumber, err1 := strconv.Atoi(chunkString)
		concurrency, err2 := strconv.Atoi(concurrencyString)
		density, err3 := strconv.Atoi(densityString)
		voxelSize, err4 := strconv.ParseFloat(voxelSizeString, 64)

		if err1 != nil || err2 != nil || err3 != nil || err4 != nil || chunkNumber <= 0 || concurrency <= 2 || density <= 1 || voxelSize <= 0 {
			print("Error parsing chunk number, concurrency, or density. All must be integers (chunk number > 0, concurrency > 2, density > 1, voxel size > 0)")
			return
		}
	}

	file, err := lidarioMod.NewLasFile(fileName, "rh")

	if err != nil {
		println("Error accessing LAS file")
		return
	}

	chunks := lasProcessing.ChunkFile(file, chunkNumber)

	processor := voxels.DensityVoxelSetProcessor{PointDensity: density}

	status := lasProcessing.NewConcurrentStatus()

	quit := false

	uiDone := make(chan bool)

	go lasProcessing.CLIStatus(status, &quit, uiDone)

	output := lasProcessing.ConcurrentProcess[voxels.DensityVoxelSet](file, chunks, &processor, concurrency, voxelSize, status)

	quit = true

	<- uiDone

	voxelSetOutput := postProcess(output)

	err = writeToFile(voxelSetOutput, destName)

	if err != nil {
		println("Error writing to csv")
		return
	}

	println("Written to csv")
}