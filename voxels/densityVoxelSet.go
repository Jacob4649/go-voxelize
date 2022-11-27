package voxels

import (
	"math"

	"github.com/Jacob4649/go-voxelize/go-voxelize/lasProcessing"
	"github.com/Jacob4649/go-voxelize/go-voxelize/lidarioMod"
)

// Type of a collection of voxels
type DensityVoxelSet struct {
	
	// Minimum point density to be considered a filled voxel
	PointDensity int

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

	// Set of voxels point densities
	Voxels map[Coordinate]int
}

// Converts a point to a voxel Coordinate
func pointToCoordinate(x float64, minX float64, y float64, minY float64, z float64, minZ float64, voxelSize float64) Coordinate {
	
	deltaX, deltaY, deltaZ := x - minX, y - minY, z - minZ
	
	return Coordinate{X: int(deltaX / voxelSize), Y: int(deltaY / voxelSize), Z: int(deltaZ / voxelSize)}
}

// Processes LAS files into VoxelSets
type DensityVoxelSetProcessor struct {
	
	// Point density required for a voxel
	PointDensity int

}

// Processes a chunk of a LAS file into a VoxelSet
func(processor *DensityVoxelSetProcessor) Process(inputFile *lidarioMod.LasFile, chunk *lasProcessing.LASChunk, voxelSize float64, output chan<- *DensityVoxelSet, status *float64) {
	
	*status = 0.0
	
	voxels := &DensityVoxelSet{Voxels: make(map[Coordinate]int)}
	
	minX, minY, minZ := inputFile.Header.MinX, inputFile.Header.MinY, inputFile.Header.MinZ

	rawBytes := chunk.ReadOnFile(inputFile)

	for i := chunk.Start; i < chunk.End; i++ {
		x, y, z := lasProcessing.ReadPointData(inputFile, chunk, rawBytes, i)
		
		coordinate := pointToCoordinate(x, minX, y, minY, z, minZ, voxelSize)

		*status = float64(i - chunk.Start) / float64(chunk.End - chunk.Start)

		val, contains := voxels.Voxels[coordinate]

		if contains {
			voxels.Voxels[coordinate] = val + 1
		} else {
			voxels.Voxels[coordinate] = 1
		}
	}

	*status = 1.0

	output <- voxels
}

// Gets an empty VoxelSet
func(processor *DensityVoxelSetProcessor) EmptyOutput(inputFile *lidarioMod.LasFile, voxelSize float64) *DensityVoxelSet {
	
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

	voxels := make(map[Coordinate]int)

	return &DensityVoxelSet{XSize: xSize, YSize: ySize, ZSize: zSize, XVoxels: xVoxels, YVoxels: yVoxels, ZVoxels: zVoxels, Voxels: voxels, PointDensity: processor.PointDensity}
}

// Combines two VoxelSets
func(processor *DensityVoxelSetProcessor) CombineOutput(base *DensityVoxelSet, incoming *DensityVoxelSet) *DensityVoxelSet {
	for coordinate, density := range incoming.Voxels {
		baseDensity, contains := base.Voxels[coordinate]
		if contains {
			base.Voxels[coordinate] = baseDensity + density
		} else {
			base.Voxels[coordinate] = density
		}
	}
	return base
}
