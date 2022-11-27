package voxels

import mapset "github.com/deckarep/golang-set/v2"

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
func PointToCoordinate(x float64, minX float64, y float64, minY float64, z float64, minZ float64, voxelSize float64) Coordinate {
	
	deltaX, deltaY, deltaZ := x - minX, y - minY, z - minZ
	
	return Coordinate{X: int(deltaX / voxelSize), Y: int(deltaY / voxelSize), Z: int(deltaZ / voxelSize)}
}
