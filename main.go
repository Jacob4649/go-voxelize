package main

import (
	"github.com/Jacob4649/go-voxelize/go-voxelize/lasProcessing"
	"github.com/Jacob4649/go-voxelize/go-voxelize/lidarioMod"
	"github.com/Jacob4649/go-voxelize/go-voxelize/voxels"
)

func main() {
	fileName := "DemoFile.las"

	file, err := lidarioMod.NewLasFile(fileName, "rh")

	if err != nil {
		panic(err)
	}

	chunks := lasProcessing.ChunkFile(file, 256)

	processor := voxels.VoxelSetProcessor{}

	test := lasProcessing.ConcurrentProcess[voxels.VoxelSet](file, chunks, &processor, 32, 1)

	if test == nil {
		println("Error with test")
	} else {
		print("Success")
	}
}