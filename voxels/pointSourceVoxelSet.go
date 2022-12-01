package voxels

import (
	"math"

	"github.com/Jacob4649/go-voxelize/go-voxelize/lasProcessing"
	"github.com/Jacob4649/go-voxelize/go-voxelize/lidarioMod"
)

// Collection of voxels divided by point source
type PointSourceDensityVoxelSet struct {
	
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
	VoxelsBySource map[int]map[Coordinate]int
}

// Processes LAS files into VoxelSets divided by point source
type PointSourceProcessor struct {
	
	// Point density required for a voxel
	PointDensity int

	// Voxel size for this processor
	VoxelSize float64

}

// Processes a chunk of a LAS file into a VoxelSet
func(processor *PointSourceProcessor) Process(inputFile *lidarioMod.LasFile, chunk *lasProcessing.LASChunk, output chan<- *PointSourceDensityVoxelSet, status *float64) {
	
	*status = 0.0
	
	voxels := &PointSourceDensityVoxelSet{VoxelsBySource: make(map[int]map[Coordinate]int)}
	
	minX, minY, minZ := inputFile.Header.MinX, inputFile.Header.MinY, inputFile.Header.MinZ

	rawBytes := chunk.ReadOnFile(inputFile)

	for i := chunk.Start; i < chunk.End; i++ {
		x, y, z := lasProcessing.ReadPointData(inputFile, chunk, rawBytes, i)
		source := lasProcessing.ReadPointSource(inputFile, chunk, rawBytes, i)

		coordinate := pointToCoordinate(x, minX, y, minY, z, minZ, processor.VoxelSize)

		*status = float64(i - chunk.Start) / float64(chunk.End - chunk.Start)

		_, containsSource := voxels.VoxelsBySource[source]

		if containsSource {
			val, contains := voxels.VoxelsBySource[source][coordinate]

			if contains {
				voxels.VoxelsBySource[source][coordinate] = val + 1
			} else {
				voxels.VoxelsBySource[source][coordinate] = 1
			}
		} else {
			newMap := make(map[Coordinate]int)
			newMap[coordinate] = 1
			voxels.VoxelsBySource[source] = newMap
		}
	}

	*status = 1.0

	output <- voxels
}

// Gets an empty VoxelSet
func(processor *PointSourceProcessor) EmptyOutput(inputFile *lidarioMod.LasFile) *PointSourceDensityVoxelSet {
	
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

	voxels := make(map[int]map[Coordinate]int)

	return &PointSourceDensityVoxelSet{XSize: xSize, YSize: ySize, ZSize: zSize, XVoxels: xVoxels, YVoxels: yVoxels, ZVoxels: zVoxels, VoxelsBySource: voxels, PointDensity: processor.PointDensity}
}

// Combines two VoxelSets
func(processor *PointSourceProcessor) CombineOutput(base *PointSourceDensityVoxelSet, incoming *PointSourceDensityVoxelSet) *PointSourceDensityVoxelSet {
	for source, voxels := range incoming.VoxelsBySource {
		baseVoxels, contains := base.VoxelsBySource[source]
		if contains {
			for coordinate, density := range voxels {
				baseDensity, contains := baseVoxels[coordinate]
				if contains {
					baseVoxels[coordinate] = baseDensity + density
				} else {
					baseVoxels[coordinate] = density
				}
			}
		} else {
			base.VoxelsBySource[source] = voxels
		}
	}

	return base
}