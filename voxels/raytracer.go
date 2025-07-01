package voxels

import (
	"math"

	"github.com/Jacob4649/go-voxelize/go-voxelize/lasProcessing"
)

// Vector3D represents a 3D vector or point.
type Vector3D struct {
	X float64
	Y float64
	Z float64
}

// Ray represents a ray with an origin and a direction vector.
type Ray struct {
	Origin    Coordinate
	Direction Vector3D
}

// castRay_DDA casts a ray using a Digital Differential Analyzer (DDA) algorithm
// and returns the list of voxel coordinates it passes through up to a maximum distance.
func castRay_DDA(ray Ray, maxDistance float64, voxelSize float64) []Coordinate {
	path := []Coordinate{}

	currentX_float := (float64(ray.Origin.X) + 0.5) * voxelSize
	currentY_float := (float64(ray.Origin.Y) + 0.5) * voxelSize
	currentZ_float := (float64(ray.Origin.Z) + 0.5) * voxelSize

	dirLength := math.Sqrt(ray.Direction.X*ray.Direction.X + ray.Direction.Y*ray.Direction.Y + ray.Direction.Z*ray.Direction.Z)
	if dirLength == 0 {
		return path
	}

	dx_norm := ray.Direction.X / dirLength
	dy_norm := ray.Direction.Y / dirLength
	dz_norm := ray.Direction.Z / dirLength

	currentVoxel := ray.Origin
	path = append(path, currentVoxel)

	var stepX, stepY, stepZ int
	var tMaxX, tMaxY, tMaxZ float64
	var deltaTx, deltaTy, deltaTz float64

	if dx_norm > 0 {
		stepX = 1
		tMaxX = (((float64(ray.Origin.X) + 1.0) * voxelSize) - currentX_float) / dx_norm
	} else if dx_norm < 0 {
		stepX = -1
		tMaxX = ((float64(ray.Origin.X) * voxelSize) - currentX_float) / dx_norm
	} else {
		stepX = 0
		tMaxX = math.MaxFloat64
	}
	if dx_norm != 0 {
		deltaTx = voxelSize / math.Abs(dx_norm)
	} else {
		deltaTx = math.MaxFloat64
	}

	if dy_norm > 0 {
		stepY = 1
		tMaxY = (((float64(ray.Origin.Y) + 1.0) * voxelSize) - currentY_float) / dy_norm
	} else if dy_norm < 0 {
		stepY = -1
		tMaxY = ((float64(ray.Origin.Y) * voxelSize) - currentY_float) / dy_norm
	} else {
		stepY = 0
		tMaxY = math.MaxFloat64
	}
	if dy_norm != 0 {
		deltaTy = voxelSize / math.Abs(dy_norm)
	} else {
		deltaTy = math.MaxFloat64
	}

	if dz_norm > 0 {
		stepZ = 1
		tMaxZ = (((float64(ray.Origin.Z) + 1.0) * voxelSize) - currentZ_float) / dz_norm
	} else if dz_norm < 0 {
		stepZ = -1
		tMaxZ = ((float64(ray.Origin.Z) * voxelSize) - currentZ_float) / dz_norm
	} else {
		stepZ = 0
		tMaxZ = math.MaxFloat64
	}
	if dz_norm != 0 {
		deltaTz = voxelSize / math.Abs(dz_norm)
	} else {
		deltaTz = math.MaxFloat64
	}

	distTraveledParametric := 0.0

	for distTraveledParametric * dirLength < maxDistance {
		prevVoxel := currentVoxel
		updated := false

		if tMaxX < tMaxY {
			if tMaxX < tMaxZ {
				distTraveledParametric = tMaxX
				currentVoxel.X += stepX
				tMaxX += deltaTx
				updated = true
			} else if tMaxZ < tMaxX {
				distTraveledParametric = tMaxZ
				currentVoxel.Z += stepZ
				tMaxZ += deltaTz
				updated = true
			} else {
				distTraveledParametric = tMaxX
				currentVoxel.X += stepX
				currentVoxel.Z += stepZ
				tMaxX += deltaTx
				tMaxZ += deltaTz
				updated = true
			}
		} else if tMaxY < tMaxX {
			if tMaxY < tMaxZ {
				distTraveledParametric = tMaxY
				currentVoxel.Y += stepY
				tMaxY += deltaTy
				updated = true
			} else if tMaxZ < tMaxY {
				distTraveledParametric = tMaxZ
				currentVoxel.Z += stepZ
				tMaxZ += deltaTz
				updated = true
			} else {
				distTraveledParametric = tMaxY
				currentVoxel.Y += stepY
				currentVoxel.Z += stepZ
				tMaxY += deltaTy
				tMaxZ += deltaTz
				updated = true
			}
		} else {
			if tMaxX < tMaxZ {
				distTraveledParametric = tMaxX
				currentVoxel.X += stepX
				currentVoxel.Y += stepY
				tMaxX += deltaTx
				tMaxY += deltaTy
				updated = true
			} else if tMaxZ < tMaxX {
				distTraveledParametric = tMaxZ
				currentVoxel.Z += stepZ
				tMaxZ += deltaTz
				updated = true
			} else {
				distTraveledParametric = tMaxX
				currentVoxel.X += stepX
				currentVoxel.Y += stepY
				currentVoxel.Z += stepZ
				tMaxX += deltaTx
				tMaxY += deltaTy
				tMaxZ += deltaTz
				updated = true
			}
		}

		if !updated {
			break
		}

		if distTraveledParametric * dirLength >= maxDistance {
			break
		}

		if currentVoxel != prevVoxel {
			path = append(path, currentVoxel)
		}
	}
	return path
}

func isVoxelFilled(coord Coordinate, vs *VoxelSet) bool {
	if vs == nil || vs.Voxels == nil {
		return false
	}
	return vs.Voxels.Contains(coord)
}

type RayHitInfo struct {
	RayIndex         int
	Path             []Coordinate
	HitVoxel         *Coordinate
	HitVoxelPathIndex int
	ScannerOriginIdx int
}

func generateScannerRays(scannerPos Coordinate, radius int) []Ray {
	rays := []Ray{}
	if radius <= 0 {
		return rays
	}

	processedTargets := make(map[Coordinate]bool)

	for x_rel := -radius; x_rel <= radius; x_rel++ {
		for y_rel := -radius; y_rel <= radius; y_rel++ {
			for z_rel := -radius; z_rel <= radius; z_rel++ {
				if math.Abs(float64(x_rel)) == float64(radius) ||
				   math.Abs(float64(y_rel)) == float64(radius) ||
				   math.Abs(float64(z_rel)) == float64(radius) {

					target := Coordinate{
						X: scannerPos.X + x_rel,
						Y: scannerPos.Y + y_rel,
						Z: scannerPos.Z + z_rel,
					}

					if target == scannerPos {
						continue
					}

					if _, exists := processedTargets[target]; !exists {
						direction := Vector3D{
							X: float64(x_rel),
							Y: float64(y_rel),
							Z: float64(z_rel),
						}
						rays = append(rays, Ray{Origin: scannerPos, Direction: direction})
						processedTargets[target] = true
					}
				}
			}
		}
	}
	return rays
}

func ApplyScannerOcclusionProcessing(initialVoxelSet *VoxelSet, scannerLocations []Coordinate, voxelSize float64, sceneMinBound, sceneMaxBound Coordinate) *VoxelSet {
	if len(scannerLocations) == 0 {
		return initialVoxelSet
	}

	allRayHitInfos := []RayHitInfo{}

	sceneDx := float64(sceneMaxBound.X - sceneMinBound.X)
	sceneDy := float64(sceneMaxBound.Y - sceneMinBound.Y)
	sceneDz := float64(sceneMaxBound.Z - sceneMinBound.Z)

	maxSceneExtent := math.Max(sceneDx, math.Max(sceneDy, sceneDz))
	rayGenerationRadius := int(math.Ceil(maxSceneExtent / 2.0))
	if rayGenerationRadius == 0 {
		if maxSceneExtent > 0 {
			rayGenerationRadius = 1
		} else {
			rayGenerationRadius = 10
		}
	}

	maxDdaDistance := math.Sqrt(sceneDx*sceneDx + sceneDy*sceneDy + sceneDz*sceneDz) * voxelSize
	if maxDdaDistance == 0 {
		maxDdaDistance = float64(rayGenerationRadius) * voxelSize * 1.5
	}

	rayGlobalIndex := 0
	for scannerIdx, scannerPos := range scannerLocations {
		rays := generateScannerRays(scannerPos, rayGenerationRadius)

		for _, ray := range rays {
			path := castRay_DDA(ray, maxDdaDistance, voxelSize)
			hitInfo := RayHitInfo{
				RayIndex:         rayGlobalIndex,
				Path:             path,
				HitVoxel:         nil,
				HitVoxelPathIndex: -1,
				ScannerOriginIdx: scannerIdx,
			}

			for pathIdx, voxelCoord := range path {
				if pathIdx == 0 && voxelCoord == ray.Origin {
					continue
				}
				if isVoxelFilled(voxelCoord, initialVoxelSet) {
					hitCopy := voxelCoord
					hitInfo.HitVoxel = &hitCopy
					hitInfo.HitVoxelPathIndex = pathIdx
					break
				}
			}
			allRayHitInfos = append(allRayHitInfos, hitInfo)
			rayGlobalIndex++
		}
	}

	voxelsToAdd := []Coordinate{}

	for _, currentRayInfo := range allRayHitInfos {
		if currentRayInfo.HitVoxel == nil || currentRayInfo.HitVoxelPathIndex == -1 {
			continue
		}

		for i := currentRayInfo.HitVoxelPathIndex + 1; i < len(currentRayInfo.Path); i++ {
			voxelToFill := currentRayInfo.Path[i]
			isEventuallyVisibleToOtherScanner := false
			for _, otherRayInfo := range allRayHitInfos {
				if otherRayInfo.ScannerOriginIdx == currentRayInfo.ScannerOriginIdx {
					continue
				}

				clearPathLimit := otherRayInfo.HitVoxelPathIndex
				if clearPathLimit == -1 {
					clearPathLimit = len(otherRayInfo.Path)
				}

				for j := 0; j < clearPathLimit; j++ {
					if otherRayInfo.Path[j] == voxelToFill {
						isEventuallyVisibleToOtherScanner = true
						break
					}
				}
				if isEventuallyVisibleToOtherScanner {
					break
				}
			}

			if !isEventuallyVisibleToOtherScanner {
				if !initialVoxelSet.Voxels.Contains(voxelToFill){
					alreadyPending := false
					for _, vCheck := range voxelsToAdd {
						if vCheck == voxelToFill {
							alreadyPending = true
							break
						}
					}
					if !alreadyPending {
						voxelsToAdd = append(voxelsToAdd, voxelToFill)
					}
				}
			} else {
				break
			}
		}
	}

	for _, v := range voxelsToAdd {
		initialVoxelSet.Voxels.Add(v)
	}

	return initialVoxelSet
}

type OcclusionProcessor struct {
	ScannerLocations []Coordinate
	VoxelSize        float64
}

func (op *OcclusionProcessor) Process(inputVoxelSet *VoxelSet, status *lasProcessing.PipelineStatus) *VoxelSet {
	if op == nil || len(op.ScannerLocations) == 0 {
		if status != nil {
			*status = lasProcessing.PipelineStatus{Step: "OcclusionFill", Progress: 1.0} // Removed Message
		}
		return inputVoxelSet
	}

	if status != nil {
		*status = lasProcessing.PipelineStatus{Step: "OcclusionFill", Progress: 0.0} // Removed Message
	}

	sceneMinBound := Coordinate{X: inputVoxelSet.XMin, Y: inputVoxelSet.YMin, Z: inputVoxelSet.ZMin}
	sceneMaxBound := Coordinate{
		X: inputVoxelSet.XMin + inputVoxelSet.XVoxels - 1,
		Y: inputVoxelSet.YMin + inputVoxelSet.YVoxels - 1,
		Z: inputVoxelSet.ZMin + inputVoxelSet.ZVoxels - 1,
	}

	processedVoxelSet := ApplyScannerOcclusionProcessing(inputVoxelSet, op.ScannerLocations, op.VoxelSize, sceneMinBound, sceneMaxBound)

	if status != nil {
		*status = lasProcessing.PipelineStatus{Step: "OcclusionFill", Progress: 1.0} // Removed Message
	}
	return processedVoxelSet
}
