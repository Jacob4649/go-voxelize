package voxels

import (
	"fmt"
	"os"

	"github.com/Jacob4649/go-voxelize/go-voxelize/lasProcessing"
	mapset "github.com/deckarep/golang-set/v2"
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
func PointToCoordinate(x float64, minX float64, y float64, minY float64, z float64, minZ float64, voxelSize float64) Coordinate {
	
	deltaX, deltaY, deltaZ := x - minX, y - minY, z - minZ
	
	return Coordinate{X: int(deltaX / voxelSize), Y: int(deltaY / voxelSize), Z: int(deltaZ / voxelSize)}
}

// Normalizes heights lazily
type LazyNormalizer struct {

}

// Pair of x and y coordinates
type XYPair struct {
	
	// X coordinate value
	X int

	// Y coordinate value
	Y int
}

// normalizes heights after the fact (z is up)
func(normalizer *LazyNormalizer) Process(voxelSet *VoxelSet, status *lasProcessing.PipelineStatus) *VoxelSet {

	// map xy to minimum z value
	minHeights := make(map[XYPair]int)

	total := voxelSet.Voxels.Cardinality()

	current := 0

	*status = lasProcessing.PipelineStatus{Step: "Finding minimums", Progress: 0.0}

	for voxel := range voxelSet.Voxels.Iterator().C {
		xy := XYPair{X: voxel.X, Y: voxel.Y}
		min, contains := minHeights[xy]
		if contains {
			if voxel.Z < min {
				minHeights[xy] = voxel.Z
			}
		} else {
			minHeights[xy] = voxel.Z
		}

		current += 1
		*status = lasProcessing.PipelineStatus{Step: "Finding minimums", Progress: float64(current) / float64(total)}
	}

	newVoxelSet := mapset.NewThreadUnsafeSet[Coordinate]()

	current = 0

	*status = lasProcessing.PipelineStatus{Step: "Normalizing", Progress: 0.0}

	for voxel := range voxelSet.Voxels.Iterator().C {
		xy := XYPair{X: voxel.X, Y: voxel.Y}
		min := minHeights[xy]
		
		voxel.Z -= min
		newVoxelSet.Add(voxel)

		current += 1
		*status = lasProcessing.PipelineStatus{Step: "Normalizing", Progress: float64(current) / float64(total)}
	}

	voxelSet.Voxels = newVoxelSet
	
	return voxelSet
}

// Writes the output to a file
type VoxelFileWriter struct {

	// Filename to write to
	FileName string

}

// Writes a set of voxels to a file
func(writer *VoxelFileWriter) Process(voxels *VoxelSet, status *lasProcessing.PipelineStatus) error {
	file, err := os.Create(writer.FileName)

	if err != nil {
		return err
	}

	defer file.Close()

	file.WriteString("x,y,z\n")

	total := voxels.Voxels.Cardinality()

	current := 0

	*status = lasProcessing.PipelineStatus{Step: "Writing", Progress: 0.0}

	for voxel := range voxels.Voxels.Iterator().C {
		_, err = file.WriteString(fmt.Sprint(voxel.X) + "," +fmt.Sprint(voxel.Y) + "," + fmt.Sprint(voxel.Z) + "\n")

		if err != nil {
			return err
		}

		current += 1
		*status = lasProcessing.PipelineStatus{Step: "Writing", Progress: float64(current) / float64(total)}
	}

	return nil
}
