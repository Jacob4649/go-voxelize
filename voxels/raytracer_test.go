package voxels

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"testing"

	mapset "github.com/deckarep/golang-set/v2"
)

// Helper function to compare two slices of Coordinates.
func compareCoordinateSlices(a, b []Coordinate) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Helper to convert coordinate slice to string for easier comparison in test output
func coordsToString(coords []Coordinate) string {
	strs := make([]string, len(coords))
	for i, c := range coords {
		strs[i] = fmt.Sprintf("{%d,%d,%d}", c.X, c.Y, c.Z)
	}
	sort.Strings(strs)
	return strings.Join(strs, ", ")
}


func TestCastRay_DDA(t *testing.T) {
	voxelSize := 1.0
	tests := []struct {name string; ray Ray; maxDistance float64; expectedPath []Coordinate}{
		{"Horiz D(1,0,0) mD=3", Ray{Origin:Coordinate{X:0,Y:0,Z:0}, Direction:Vector3D{X:1,Y:0,Z:0}}, 3.0*voxelSize, []Coordinate{{0,0,0},{1,0,0},{2,0,0},{3,0,0}}},
		{"Vert D(0,0,-1) mD=2", Ray{Origin:Coordinate{X:1,Y:1,Z:0}, Direction:Vector3D{X:0,Y:0,Z:-1}}, 2.0*voxelSize, []Coordinate{{1,1,0},{1,1,-1},{1,1,-2}}},
		{"Diag D(1,1,0) mD=~2.12", Ray{Origin:Coordinate{X:0,Y:0,Z:0}, Direction:Vector3D{X:1,Y:1,Z:0}}, math.Sqrt(2.0)*1.5*voxelSize, []Coordinate{{0,0,0},{1,1,0}}},
		{"Zero D mD=5", Ray{Origin:Coordinate{X:0,Y:0,Z:0}, Direction:Vector3D{X:0,Y:0,Z:0}}, 5.0*voxelSize, []Coordinate{}},
		{"Short mD stays origin", Ray{Origin:Coordinate{X:0,Y:0,Z:0}, Direction:Vector3D{X:1,Y:0,Z:0}}, 0.4*voxelSize, []Coordinate{{0,0,0}}},
		{"NonNorm D(2,0,0) mD=2.5", Ray{Origin:Coordinate{X:0,Y:0,Z:0}, Direction:Vector3D{X:2,Y:0,Z:0}}, 2.5*voxelSize, []Coordinate{{0,0,0},{1,0,0}}},
		{"Ray S1 ApplyT1", Ray{Origin:Coordinate{X:0,Y:5,Z:0}, Direction:Vector3D{X:2,Y:0,Z:0}}, 14.142, []Coordinate{{0,5,0},{1,5,0},{2,5,0},{3,5,0},{4,5,0},{5,5,0},{6,5,0},{7,5,0}}},
		{"Ray S2 ApplyT1", Ray{Origin:Coordinate{X:10,Y:5,Z:0}, Direction:Vector3D{X:-8,Y:0,Z:0}}, 14.142, []Coordinate{{10,5,0},{9,5,0},{8,5,0}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualPath := castRay_DDA(tt.ray, tt.maxDistance, voxelSize)
			if !compareCoordinateSlices(actualPath, tt.expectedPath) {
				t.Errorf("castRay_DDA() path = %v, want %v", actualPath, tt.expectedPath)
			}
		})
	}
}

func TestGenerateScannerRays(t *testing.T) {
	tests := []struct {name string; scannerPos Coordinate; radius int; expectedNumRays int}{
		{name: "r0", scannerPos: Coordinate{0,0,0}, radius: 0, expectedNumRays: 0},
		{name: "r1", scannerPos: Coordinate{0,0,0}, radius: 1, expectedNumRays: 26},
		{name: "r2", scannerPos: Coordinate{5,5,5}, radius: 2, expectedNumRays: 98},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rays := generateScannerRays(tt.scannerPos, tt.radius)
			if len(rays) != tt.expectedNumRays {
				t.Errorf("generateScannerRays() num_rays = %d, want %d", len(rays), tt.expectedNumRays)
			}
			// Further checks for target uniqueness and shell property could be added here if needed
		})
	}
}

func TestIsVoxelFilled(t *testing.T) {
	vsEmpty := &VoxelSet{Voxels: mapset.NewSet[Coordinate]()}
	c1, c2 := Coordinate{1,2,3}, Coordinate{4,5,6}
	vsFilled := &VoxelSet{Voxels: mapset.NewSet[Coordinate](c1)}
	tests := []struct{name string; c Coordinate; vs *VoxelSet; expected bool}{
		{"empty",c1,vsEmpty,false}, {"filled, present",c1,vsFilled,true},
		{"filled, absent",c2,vsFilled,false}, {"nil VS",c1,nil,false},
		{"VS nil map",c1,&VoxelSet{Voxels:nil},false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isVoxelFilled(tt.c, tt.vs); got != tt.expected {
				t.Errorf("isVoxelFilled() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// mockApplyScannerOcclusion is a simplified version for testing ApplyScannerOcclusionProcessing's second stage.
// It takes pre-calculated allRayHitInfos.
func mockApplyScannerOcclusion(initialVoxelSet *VoxelSet, allRayHitInfos []RayHitInfo) *VoxelSet {
	voxelsToAdd := []Coordinate{}
	for _, currentRayInfo := range allRayHitInfos {
		if currentRayInfo.HitVoxel == nil || currentRayInfo.HitVoxelPathIndex == -1 { continue }
		for i := currentRayInfo.HitVoxelPathIndex + 1; i < len(currentRayInfo.Path); i++ {
			voxelToFill := currentRayInfo.Path[i]
			isEventuallyVisible := false
			for _, otherRayInfo := range allRayHitInfos {
				if otherRayInfo.ScannerOriginIdx == currentRayInfo.ScannerOriginIdx { continue }
				limit := otherRayInfo.HitVoxelPathIndex
				if limit == -1 { limit = len(otherRayInfo.Path) }
				for j := 0; j < limit; j++ {
					if otherRayInfo.Path[j] == voxelToFill { isEventuallyVisible = true; break }
				}
				if isEventuallyVisible { break }
			}
			if !isEventuallyVisible {
				if !initialVoxelSet.Voxels.Contains(voxelToFill) {
					alreadyPending := false
					for _, vC := range voxelsToAdd { if vC == voxelToFill { alreadyPending = true; break } }
					if !alreadyPending { voxelsToAdd = append(voxelsToAdd, voxelToFill) }
				}
			} else { break }
		}
	}
	for _, v := range voxelsToAdd { initialVoxelSet.Voxels.Add(v) }
	return initialVoxelSet
}

func TestApplyScannerOcclusionProcessingLogic(t *testing.T) {
	// Tests the second stage (filling logic) of ApplyScannerOcclusionProcessing
	// by providing manually crafted RayHitInfo slices.

	c000, c100, c200, c300, c400 := Coordinate{0,0,0}, Coordinate{1,0,0}, Coordinate{2,0,0}, Coordinate{3,0,0}, Coordinate{4,0,0} // Removed c500
	c050, c150, c250, c350, c450, c550 := Coordinate{0,5,0}, Coordinate{1,5,0}, Coordinate{2,5,0}, Coordinate{3,5,0}, Coordinate{4,5,0}, Coordinate{5,5,0}
	c1050, c950, c850 := Coordinate{10,5,0}, Coordinate{9,5,0}, Coordinate{8,5,0}


	tests := []struct {
		name            string
		initialVoxels   []Coordinate
		allRayHitInfos  []RayHitInfo
		expectedFinal   []Coordinate
	}{
		{
			name: "Single Ray, Hit, Fill Behind",
			initialVoxels: []Coordinate{c200},
			allRayHitInfos: []RayHitInfo{
				{RayIndex:0, Path:[]Coordinate{c000,c100,c200,c300,c400}, HitVoxel:&c200, HitVoxelPathIndex:2, ScannerOriginIdx:0},
			},
			expectedFinal: []Coordinate{c200,c300,c400},
		},
		{
			name: "S1 blocked by S2 clear path",
			initialVoxels: []Coordinate{c250},
			allRayHitInfos: []RayHitInfo{
				{RayIndex:0, Path:[]Coordinate{c050,c150,c250,c350,c450}, HitVoxel:&c250, HitVoxelPathIndex:2, ScannerOriginIdx:0},
				{RayIndex:1, Path:[]Coordinate{c1050,c950,c850,c350,c250}, HitVoxel:&c250, HitVoxelPathIndex:4, ScannerOriginIdx:1},
			},
			expectedFinal: []Coordinate{c250},
		},
		{
			name: "S1 fills some, then S2 clear path stops",
			initialVoxels: []Coordinate{c250},
			allRayHitInfos: []RayHitInfo{
				{RayIndex:0, Path:[]Coordinate{c050,c150,c250,c350,c450,c550}, HitVoxel:&c250, HitVoxelPathIndex:2, ScannerOriginIdx:0},
				{RayIndex:1, Path:[]Coordinate{c1050,c950,c850,c450}, HitVoxel:nil, HitVoxelPathIndex:-1, ScannerOriginIdx:1},
			},
			expectedFinal: []Coordinate{c250,c350},
		},
		{
			name: "S2 view blocked, S1 fills more",
			initialVoxels: []Coordinate{c250, c850},
			allRayHitInfos: []RayHitInfo{
				{RayIndex:0, Path:[]Coordinate{c050,c150,c250,c350,c450,c550}, HitVoxel:&c250, HitVoxelPathIndex:2, ScannerOriginIdx:0},
				{RayIndex:1, Path:[]Coordinate{c1050,c950,c850}, HitVoxel:&c850, HitVoxelPathIndex:2, ScannerOriginIdx:1},
			},
			expectedFinal: []Coordinate{c250,c850,c350,c450,c550},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initialVS := &VoxelSet{ Voxels: mapset.NewSet[Coordinate]() }
			for _, c := range tt.initialVoxels { initialVS.Voxels.Add(c) }

			finalVS := mockApplyScannerOcclusion(initialVS, tt.allRayHitInfos)

			finalSetExpected := mapset.NewSet[Coordinate]()
			for _, c := range tt.expectedFinal { finalSetExpected.Add(c) }

			if !finalVS.Voxels.Equal(finalSetExpected) {
				gotSlice, expectedSlice := finalVS.Voxels.ToSlice(), finalSetExpected.ToSlice()
				customSort := func(s []Coordinate) {
					sort.Slice(s, func(i,j int) bool {
						if s[i].X!=s[j].X { return s[i].X < s[j].X }
						if s[i].Y!=s[j].Y { return s[i].Y < s[j].Y }
						return s[i].Z < s[j].Z
					})
				}
				customSort(gotSlice); customSort(expectedSlice)
				t.Errorf("mockApplyScannerOcclusion() %s: \nGot:  %s\nWant: %s", tt.name, coordsToString(gotSlice), coordsToString(expectedSlice))
			}
		})
	}
}
