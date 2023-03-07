package voxels

import (
	"math"

	"github.com/Jacob4649/go-voxelize/go-voxelize/lasProcessing"
	"github.com/Jacob4649/go-voxelize/go-voxelize/lidarioMod"

	mapset "github.com/deckarep/golang-set/v2"
)

// Processes LAS files into VoxelSets
type VoxelSetProcessor struct {
	
	// Voxel size to use with this processor
	VoxelSize float64

}

// Processes a chunk of a LAS file into a VoxelSet
func(processor *VoxelSetProcessor) Process(inputFile *lidarioMod.LasFile, chunk *lasProcessing.LASChunk, output chan<- *VoxelSet, status *float64) {
	
	*status = 0.0
	
	minX, minY, minZ := inputFile.Header.MinX, inputFile.Header.MinY, inputFile.Header.MinZ

	minXVoxel, minYVoxel, minZVoxel := int(minX / processor.VoxelSize), int(minY / processor.VoxelSize), int(minZ / processor.VoxelSize)

	voxels := &VoxelSet{Voxels: mapset.NewThreadUnsafeSet[Coordinate](), XMin: minXVoxel, YMin: minYVoxel, ZMin: minZVoxel}
	
	rawBytes := chunk.ReadOnFile(inputFile)

	for i := chunk.Start; i < chunk.End; i++ {
		x, y, z := lasProcessing.ReadPointData(inputFile, chunk, rawBytes, i)
		
		coordinate := PointToCoordinate(x, minX, y, minY, z, minZ, processor.VoxelSize, false)

		*status = float64(i - chunk.Start) / float64(chunk.End - chunk.Start)

		voxels.Voxels.Add(coordinate)
	}

	*status = 1.0

	output <- voxels
}

// Gets an empty VoxelSet
func(processor *VoxelSetProcessor) EmptyOutput(inputFile *lidarioMod.LasFile) *VoxelSet {
	
	header := inputFile.Header

	xSizeRaw, ySizeRaw, zSizeRaw := header.MaxX - header.MinX, header.MaxY - header.MinY, header.MaxZ - header.MinZ

	xRemainder, yRemainder, zRemainder := math.Mod(xSizeRaw, processor.VoxelSize), math.Mod(ySizeRaw, processor.VoxelSize), math.Mod(zSizeRaw, processor.VoxelSize)

	xVoxels, yVoxels, zVoxels := int(xSizeRaw / processor.VoxelSize), int(ySizeRaw / processor.VoxelSize), int(zSizeRaw / processor.VoxelSize)

	if (xRemainder != 0) {
		xVoxels += 1
	}

	if (yRemainder != 0) {
		yVoxels += 1
	}

	if (zRemainder != 0) {
		zVoxels += 1
	}

	xSize, ySize, zSize := float64(xVoxels) * processor.VoxelSize, float64(yVoxels) * processor.VoxelSize, float64(zVoxels) * processor.VoxelSize

	minX, minY, minZ := inputFile.Header.MinX, inputFile.Header.MinY, inputFile.Header.MinZ

	minXVoxel, minYVoxel, minZVoxel := int(minX / processor.VoxelSize), int(minY / processor.VoxelSize), int(minZ / processor.VoxelSize)

	voxels := mapset.NewThreadUnsafeSet[Coordinate]()

	return &VoxelSet{XSize: xSize, YSize: ySize, ZSize: zSize, XVoxels: xVoxels, YVoxels: yVoxels, ZVoxels: zVoxels, Voxels: voxels, XMin: minXVoxel, YMin: minYVoxel, ZMin: minZVoxel}
}

// Combines two VoxelSets
func(processor *VoxelSetProcessor) CombineOutput(base *VoxelSet, incoming *VoxelSet) *VoxelSet {
	base.Voxels = base.Voxels.Union(incoming.Voxels)
	return base
}
