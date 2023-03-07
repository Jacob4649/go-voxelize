package voxels

import (
	"math"

	"github.com/Jacob4649/go-voxelize/go-voxelize/lasProcessing"
	"github.com/Jacob4649/go-voxelize/go-voxelize/lidarioMod"
	mapset "github.com/deckarep/golang-set/v2"
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

	// Min number of voxels in the x direction
	XMin int

	// Min number of voxels in the y direction
	YMin int

	// Min number of voxels in the z direction
	ZMin int

	// Set of voxels point densities
	Voxels map[Coordinate]int
}

// Processes LAS files into VoxelSets
type DensityVoxelSetProcessor struct {
	
	// Point density required for a voxel
	PointDensity int

	// Voxel size for this processor
	VoxelSize float64

}

// Processes a chunk of a LAS file into a VoxelSet
func(processor *DensityVoxelSetProcessor) Process(inputFile *lidarioMod.LasFile, chunk *lasProcessing.LASChunk, output chan<- *DensityVoxelSet, status *float64) {
	
	*status = 0.0
	
	minX, minY, minZ := inputFile.Header.MinX, inputFile.Header.MinY, inputFile.Header.MinZ

	minXVoxel, minYVoxel, minZVoxel := int(minX / processor.VoxelSize), int(minY / processor.VoxelSize), int(minZ / processor.VoxelSize)

	voxels := &DensityVoxelSet{Voxels: make(map[Coordinate]int), XMin: minXVoxel, YMin: minYVoxel, ZMin: minZVoxel}
	
	rawBytes := chunk.ReadOnFile(inputFile)

	for i := chunk.Start; i < chunk.End; i++ {
		x, y, z := lasProcessing.ReadPointData(inputFile, chunk, rawBytes, i)
		
		coordinate := PointToCoordinate(x, minX, y, minY, z, minZ, processor.VoxelSize, false)

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
func(processor *DensityVoxelSetProcessor) EmptyOutput(inputFile *lidarioMod.LasFile) *DensityVoxelSet {
	
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

// A post processor to turn point densities into voxels
type VoxelCondenser struct {
	// the density required for a voxel
	Density int
}

// Turns voxel density into voxels
func(condenser *VoxelCondenser) Process(densityVoxels *DensityVoxelSet, status *lasProcessing.PipelineStatus) *VoxelSet {
	voxelSet := mapset.NewThreadUnsafeSet[Coordinate]()

	total := len(densityVoxels.Voxels)

	current := 0

	*status = lasProcessing.PipelineStatus{Step: "Condensing", Progress: 0.0}

	for voxel, density := range densityVoxels.Voxels {

		if density >= densityVoxels.PointDensity {
			voxelSet.Add(voxel)
		}

		current += 1
		*status = lasProcessing.PipelineStatus{Step: "Condensing", Progress: float64(current) / float64(total)}
	}

	output := &VoxelSet{XSize: densityVoxels.XSize, 
		YSize: densityVoxels.YSize,
		ZSize: densityVoxels.ZSize,
		XVoxels: densityVoxels.XVoxels,
		YVoxels: densityVoxels.YVoxels,
		ZVoxels: densityVoxels.ZVoxels,
		Voxels: voxelSet}

	return output
}
