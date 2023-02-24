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

// A column of voxels
type Column struct {

	// a set of all the voxel heights in this column
	Heights mapset.Set[int]

	// the minimum height of a voxel in this column
	MinHeight int

	// the maximum height of a voxel in this column
	MaxHeight int

	// the approximate ground height, can be at most min height
	GroundHeight int

}

// Creates a column of voxels with one voxel at the specified height
func createColumn(height int) *Column {
	return &Column{GroundHeight: height, MinHeight: height, MaxHeight: height, Heights: mapset.NewThreadUnsafeSet[int](height)}
}

// Adds a height to a column
func(column *Column) addVoxel(height int) {
	column.Heights.Add(height)

	// ground height should really be independently set
	if height < column.GroundHeight {
		column.GroundHeight = height
	}

	if height < column.MinHeight {
		column.MinHeight = height
	}

	if height > column.MaxHeight {
		column.MaxHeight = height
	}
}

// Gets the start and end of the longest empty sequence of voxels in a column
func(column *Column) getLongestEmptySequence() (int, int) {
	
	// best start and end for interval, start is a filled voxel, end is the last empty
	// end can be equal to start if no good intervals exist
	bestStart, bestEnd := column.MinHeight, column.MinHeight
	
	// currently considered start of interval, should be a filled voxel
	curStart := column.MinHeight

	// in case the minimum is far above the ground
	if column.GroundHeight < column.MinHeight {
		bestStart, bestEnd = column.GroundHeight, column.MinHeight - 1
	}

	for i := column.MinHeight + 1; i < column.MaxHeight; i++ {
		if column.Heights.Contains(i) {
			// filled voxel
			curStart = i
		} else if i - curStart > bestEnd - bestStart {
			// empty voxel longer than best interval
			bestEnd = i
			bestStart = curStart
		}
	}

	return bestStart, bestEnd
}

// Finds measurements about each column
type MeasurementFinder struct {
	
}

// finds measurements
func(finder *MeasurementFinder) Process(voxelSet *VoxelSet, status *lasProcessing.PipelineStatus) *Measurements {

	// map xy to column of voxels
	columns := make(map[XYPair]*Column)

	// height of the ground to set for all columns
	groundHeight := math.MaxInt64

	total := voxelSet.Voxels.Cardinality()

	current := 0

	*status = lasProcessing.PipelineStatus{Step: "Columns", Progress: 0.0}

	for voxel := range voxelSet.Voxels.Iterator().C {
		xy := XYPair{X: voxel.X, Y: voxel.Y}
		column, contains := columns[xy]
		if contains {
			column.addVoxel(voxel.Z)
		} else {
			columns[xy] = createColumn(voxel.Z)
		}

		if voxel.Z < groundHeight {
			groundHeight = voxel.Z
		}

		current += 1
		*status = lasProcessing.PipelineStatus{Step: "Columns", Progress: float64(current) / float64(total)}
	}

	*status = lasProcessing.PipelineStatus{Step: "Measuring", Progress: 0.0}

	measurements := createMeasurements()
	current = 0
	total = len(columns)

	for coords, column := range columns {

		// measure the specified column
		measurements.addColumn(column, coords)

		current += 1
		*status = lasProcessing.PipelineStatus{Step: "Measuring", Progress: float64(current) / float64(total)}
	}

	return measurements
}

// A collection of measurements over the 2D ground plane.
// Measurements are calculated as per https://doi.org/10.1016/j.foreco.2021.119037.
// All measurements are in voxels.
type Measurements struct {

	// map of the canopy base height at different points
	CanopyBaseHeight map[XYPair]int

	// map of the fuel strata gap at different points
	FuelStrataGap map[XYPair]int

	// map of the canopy height at different points
	CanopyHeight map[XYPair]int

	// map of the understory height at different points
	UnderstoryHeight map[XYPair]int

}

// creates a new set of measurements
func createMeasurements() *Measurements {
	cbh := make(map[XYPair]int)
	fsg := make(map[XYPair]int)
	ch := make(map[XYPair]int)
	uh := make(map[XYPair]int)
	return &Measurements{CanopyBaseHeight: cbh, FuelStrataGap: fsg, CanopyHeight: ch, UnderstoryHeight: uh}
}

// adds the specified column to some measurements for the specified coordinates
// that are not already contained in the measurements
func(measurements *Measurements) addColumn(column *Column, coords XYPair) {
	ch := column.MaxHeight + 1 // to account for voxel heights being measured from bottom left
	uh, cbh := column.getLongestEmptySequence() // uh is filled, cbh is empty, both need to be increased by 1
	uh += 1
	cbh += 1
	fsg := cbh - uh

	measurements.CanopyHeight[coords] = ch
	measurements.UnderstoryHeight[coords] = uh
	measurements.CanopyBaseHeight[coords] = cbh
	measurements.FuelStrataGap[coords] = fsg
}

// Converts a point to a voxel Coordinate
func PointToCoordinate(x float64, minX float64, y float64, minY float64, z float64, minZ float64, voxelSize float64, zeroCoords bool) Coordinate {
	
	var deltaX, deltaY, deltaZ float64

	if (zeroCoords) {
		deltaX, deltaY, deltaZ = x - minX, y - minY, z - minZ
	} else {
		deltaX, deltaY, deltaZ = x, y, z
	}
	
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

// writes measurements to a file
type MeasurementsFileWriter struct {
	// the name of the file to write to
	FileName string
}

// Writes a set of measurements to a file
func(writer *MeasurementsFileWriter) Process(measurements *Measurements, status *lasProcessing.PipelineStatus) error {
	file, err := os.Create(writer.FileName)

	if err != nil {
		return err
	}

	defer file.Close()

	file.WriteString("x,y,understory_height,canopy_base_height,fuel_strata_gap,canopy_height\n")

	total := len(measurements.CanopyBaseHeight)

	current := 0

	*status = lasProcessing.PipelineStatus{Step: "Writing", Progress: 0.0}

	for coords := range measurements.CanopyBaseHeight {

		x := coords.X
		y := coords.Y
		uh := measurements.UnderstoryHeight[coords]
		cbh := measurements.CanopyBaseHeight[coords]
		ch := measurements.CanopyHeight[coords]
		fsg := measurements.FuelStrataGap[coords]

		_, err = file.WriteString(fmt.Sprint(x) + "," + 
			fmt.Sprint(y) + "," +
			fmt.Sprint(uh) + "," +
			fmt.Sprint(cbh) + "," +
			fmt.Sprint(fsg) + "," +
			fmt.Sprint(ch) + "\n")

		if err != nil {
			return err
		}

		current += 1
		*status = lasProcessing.PipelineStatus{Step: "Writing", Progress: float64(current) / float64(total)}
	}

	return nil
}