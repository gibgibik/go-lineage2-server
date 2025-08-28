package entity

import "image"

type GetBoundsConfig struct {
	ExcludeBounds []image.Rectangle
	NpcThreshold  float32
	NpcNms        float32
}
