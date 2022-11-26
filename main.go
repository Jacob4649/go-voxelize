package main

import (
	"github.com/Jacob4649/go-voxelize/go-voxelize/lasProcessing"
	"github.com/Jacob4649/go-voxelize/go-voxelize/voxels"
	"github.com/jblindsay/lidario"
)

func main() {
	fileName := "DemoFile.las"

	file, err := lidario.NewLasFile(fileName, "r")

	if err != nil {
		panic(err)
	}

	chunks := lasProcessing.ChunkFile(file, 36)

	processor := voxels.VoxelSetProcessor{}

	test := lasProcessing.ConcurrentProcess[voxels.VoxelSet](file, chunks, &processor, 8, 1)

	if test == nil {
		println("Error with test")
	} else {
		print("Success")
	}
}