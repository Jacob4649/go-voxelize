package voxels

import (
	"math"

	"github.com/Jacob4649/go-voxelize/go-voxelize/lasProcessing"
	"github.com/Jacob4649/go-voxelize/go-voxelize/lidarioMod"

	mapset "github.com/deckarep/golang-set/v2"
)

// Processes LAS files into VoxelSets
type VoxelSetProcessor struct {
	

}

// Processes a chunk of a LAS file into a VoxelSet
func(processor *VoxelSetProcessor) Process(inputFile *lidarioMod.LasFile, chunk *lasProcessing.LASChunk, voxelSize float64, output chan<- *VoxelSet, status *float64) {
	
	*status = 0.0
	
	voxels := &VoxelSet{Voxels: mapset.NewThreadUnsafeSet[Coordinate]()}
	
	minX, minY, minZ := inputFile.Header.MinX, inputFile.Header.MinY, inputFile.Header.MinZ

	rawBytes := chunk.ReadOnFile(inputFile)

	for i := chunk.Start; i < chunk.End; i++ {
		x, y, z := lasProcessing.ReadPointData(inputFile, chunk, rawBytes, i)
		
		coordinate := PointToCoordinate(x, minX, y, minY, z, minZ, voxelSize)

		*status = float64(i - chunk.Start) / float64(chunk.End - chunk.Start)

		voxels.Voxels.Add(coordinate)
	}

	*status = 1.0

	output <- voxels
}

// Gets an empty VoxelSet
func(processor *VoxelSetProcessor) EmptyOutput(inputFile *lidarioMod.LasFile, voxelSize float64) *VoxelSet {
	
	header := inputFile.Header

	xSizeRaw, ySizeRaw, zSizeRaw := header.MaxX - header.MinX, header.MaxY - header.MinY, header.MaxZ - header.MinZ

	xRemainder, yRemainder, zRemainder := math.Mod(xSizeRaw, voxelSize), math.Mod(ySizeRaw, voxelSize), math.Mod(zSizeRaw, voxelSize)

	xVoxels, yVoxels, zVoxels := int(xSizeRaw / voxelSize), int(ySizeRaw / voxelSize), int(zSizeRaw / voxelSize)

	if (xRemainder != 0) {
		xVoxels += 1
	}

	if (yRemainder != 0) {
		yVoxels += 1
	}

	if (zRemainder != 0) {
		zVoxels += 1
	}

	xSize, ySize, zSize := float64(xVoxels) * voxelSize, float64(yVoxels) * voxelSize, float64(zVoxels) * voxelSize

	voxels := mapset.NewThreadUnsafeSet[Coordinate]()

	return &VoxelSet{XSize: xSize, YSize: ySize, ZSize: zSize, XVoxels: xVoxels, YVoxels: yVoxels, ZVoxels: zVoxels, Voxels: voxels}
}

// Combines two VoxelSets
func(processor *VoxelSetProcessor) CombineOutput(base *VoxelSet, incoming *VoxelSet) *VoxelSet {
	base.Voxels = base.Voxels.Union(incoming.Voxels)
	return base
}
