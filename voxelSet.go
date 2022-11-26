package main

import (
	"math"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/jblindsay/lidario"
)

// Voxel coordinate type
type Coordinate struct {

	// X coordinate
	X int

	// Y coordinate
	Y int

	// Z coordinate
	Z int

}

// Type of a collection of voxels
type VoxelSet struct {
	
	// Size of this VoxelSet in the X direction
	XSize float64

	// Size of this VoxelSet in the Y direction
	YSize float64

	// Size of this VoxelSet in the Z direction
	ZSize float64

	// Number of voxels in the X direction
	XVoxels int

	// Number of voxels in the Y direction
	YVoxels int

	// Number of voxels in the Z direction
	ZVoxels int

	// Set of voxels in this VoxelSet
	Voxels mapset.Set[Coordinate]
}

// Converts a point to a voxel Coordinate
func pointToCoordinate(x float64, minX float64, y float64, minY float64, z float64, minZ float64, voxelSize float64) Coordinate {
	
	deltaX, deltaY, deltaZ := x - minX, y - minY, z - minZ
	
	return Coordinate{X: int(deltaX / voxelSize), Y: int(deltaY / voxelSize), Z: int(deltaZ / voxelSize)}
}

// Processes LAS files into VoxelSets
type VoxelSetProcessor struct {
	

}

// Processes a chunk of a LAS file into a VoxelSet
func(processor *VoxelSetProcessor) Process(inputFile *lidario.LasFile, chunk *LASChunk, voxelSize float64, output chan<- *VoxelSet) {
	voxels := &VoxelSet{Voxels: mapset.NewThreadUnsafeSet[Coordinate]()}
	
	minX, minY, minZ := inputFile.Header.MinX, inputFile.Header.MinY, inputFile.Header.MinZ

	for i := chunk.start; i < chunk.end; i++ {
		point, err := inputFile.LasPoint(i)
		
		// exit on error
		if err != nil {
			panic(err)
		}

		pointData := point.PointData()
		x, y, z := pointData.X, pointData.Y, pointData.Z
		
		coordinate := pointToCoordinate(x, minX, y, minY, z, minZ, voxelSize)

		voxels.Voxels.Add(coordinate)
	}

	output <- voxels
}

// Gets an empty VoxelSet
func(processor *VoxelSetProcessor) EmptyOutput(inputFile *lidario.LasFile, voxelSize float64) *VoxelSet {
	
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