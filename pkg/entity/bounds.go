package entity

import "image"

type GetBoundsConfig struct {
	Resolution    []int
	ExcludeBounds []image.Rectangle
	NpcThreshold  float32
	NpcNms        float32
}
