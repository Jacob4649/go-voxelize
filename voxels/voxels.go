package voxels

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
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

// A height gradient of voxels
type HeightGradient struct {

	// the height gradient map
	Gradient map[int]int

}

// Converts a point to a voxel Coordinate
func PointToCoordinate(x float64, minX float64, y float64, minY float64, z float64, minZ float64, voxelSize float64) Coordinate {
	
	deltaX, deltaY, deltaZ := x - minX, y - minY, z - minZ
	
	return Coordinate{X: int(deltaX / voxelSize), Y: int(deltaY / voxelSize), Z: int(deltaZ / voxelSize)}
}

// The minimum heights at different coordinates
type MinimumHeights struct {

	// minimum heights
	Heights map[XYPair]int

	// voxels to use
	Voxels *VoxelSet

}

// Finds minimum heights in each column
type MinimumHeightFinder struct {

	// whether to output minimums
	OuptutMinimums bool

	// filename to output to
	OutputFile string

}

// hsvToRGB takes a color in HSV space with values hue(0.0 - 360.0),
// saturation (0 - 1.0) and value (0-1.0) and returns its representation
// in RGB color space, with values 0 - 0xFF.
// FROM github.com/redbo/gohsv/
func hsvToRGB(h, s, v float64) (r, g, b uint8) {
	h, f := math.Modf(h / 60.0)
	p := uint8(math.Round((v * (1.0 - s)) * 0xff))
	q := uint8(math.Round((v * (1.0 - (s * f))) * 0xff))
	t := uint8(math.Round((v * (1.0 - (s * (1.0 - f)))) * 0xff))
	vr := uint8(math.Round(v * 0xff))
	switch int(h) {
	default:
		return vr, t, p
	case 1:
		return q, vr, p
	case 2:
		return p, vr, t
	case 3:
		return p, q, vr
	case 4:
		return t, p, vr
	case 5:
		return vr, p, q
	}
}

// writes minimum heights to an image
func writeMinimumHeights(filename string, heights *MinimumHeights, status *lasProcessing.PipelineStatus, voxels *VoxelSet) {
	
	image := image.NewRGBA(image.Rect(0, 0, voxels.XVoxels, voxels.YVoxels))

	total := len(heights.Heights)

	current := 0
	
	for point, min := range heights.Heights {

		// hsv colors
		i := 200 + float64(min) / float64(voxels.ZVoxels) * -200
		r, g, b := hsvToRGB(i, 1, 1)
		color := color.RGBA{R: r, G: g, B: b, A: 255}

		// greyscale colors
		// i := 255 - uint8(float64(min) / float64(voxels.ZVoxels) * 255)
		// color := color.RGBA{R: i, G: i, B: i, A: 255}
		
		image.SetRGBA(point.X, point.Y, color)

		current += 1
		*status = lasProcessing.PipelineStatus{Step: "Write min", Progress: float64(current) / float64(total)}
	}

	outputFile, err := os.Create(filename)

	if err != nil {
		panic(err)
	}

	defer outputFile.Close()

	png.Encode(outputFile, image)
}

// finds minimum heights
func(heightFinder *MinimumHeightFinder) Process(voxelSet *VoxelSet, status *lasProcessing.PipelineStatus) *MinimumHeights {

	// map xy to minimum z value
	minHeights := make(map[XYPair]int)

	total := voxelSet.Voxels.Cardinality()

	current := 0

	*status = lasProcessing.PipelineStatus{Step: "Minimums", Progress: 0.0}

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
		*status = lasProcessing.PipelineStatus{Step: "Minimums", Progress: float64(current) / float64(total)}
	}

	heights := &MinimumHeights{Voxels: voxelSet, Heights: minHeights}

	*status = lasProcessing.PipelineStatus{Step: "Write min", Progress: 0.0}

	if heightFinder.OuptutMinimums {
		writeMinimumHeights(heightFinder.OutputFile, heights, status, voxelSet)
	}

	return heights
}

// Turns minimum heights back into plain voxels
type MinimumDegrouper struct {

}

// de groups minimum heights and voxels
func(degrouper *MinimumDegrouper) Process(heights *MinimumHeights, status *lasProcessing.PipelineStatus) *VoxelSet {
	return heights.Voxels
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
func(normalizer *LazyNormalizer) Process(voxelSet *MinimumHeights, status *lasProcessing.PipelineStatus) *VoxelSet {

	newVoxelSet := mapset.NewThreadUnsafeSet[Coordinate]()

	total := voxelSet.Voxels.Voxels.Cardinality()

	current := 0

	*status = lasProcessing.PipelineStatus{Step: "Normalizing", Progress: 0.0}

	for voxel := range voxelSet.Voxels.Voxels.Iterator().C {
		xy := XYPair{X: voxel.X, Y: voxel.Y}
		min := voxelSet.Heights[xy]
		
		voxel.Z -= min
		newVoxelSet.Add(voxel)

		current += 1
		*status = lasProcessing.PipelineStatus{Step: "Normalizing", Progress: float64(current) / float64(total)}
	}

	voxelSet.Voxels.Voxels = newVoxelSet
	
	return voxelSet.Voxels
}

// Converts voxels to a height gradient
type GradientProcessor struct {

}

// finds minimum heights
func(gradientProcessor *GradientProcessor) Process(voxelSet *VoxelSet, status *lasProcessing.PipelineStatus) *HeightGradient {

	// map height to minimum coxel count
	gradient := make(map[int]int)

	total := voxelSet.Voxels.Cardinality()

	current := 0

	*status = lasProcessing.PipelineStatus{Step: "Gradient", Progress: 0.0}

	for voxel := range voxelSet.Voxels.Iterator().C {
		count, contains := gradient[voxel.Z]
		
		if contains {
			gradient[voxel.Z] = count + 1
		} else {
			gradient[voxel.Z] = 1
		}

		current += 1
		*status = lasProcessing.PipelineStatus{Step: "Gradient", Progress: float64(current) / float64(total)}
	}

	return &HeightGradient{Gradient: gradient}
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

// Writes a gradient to a file
type GradientFileWriter struct {
	// The file name to write to
	FileName string
}

// Writes a set of voxels to a file
func(writer *GradientFileWriter) Process(gradient *HeightGradient, status *lasProcessing.PipelineStatus) error {
	file, err := os.Create(writer.FileName)

	if err != nil {
		return err
	}

	defer file.Close()

	file.WriteString("height,count\n")

	total := len(gradient.Gradient)

	current := 0

	*status = lasProcessing.PipelineStatus{Step: "Writing", Progress: 0.0}

	for height, count := range gradient.Gradient {
		_, err = file.WriteString(fmt.Sprint(height) + "," + fmt.Sprint(count) + "\n")

		if err != nil {
			return err
		}

		current += 1
		*status = lasProcessing.PipelineStatus{Step: "Writing", Progress: float64(current) / float64(total)}
	}

	return nil
}
