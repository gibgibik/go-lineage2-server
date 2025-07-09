package main

import (
	"bytes"
	"fmt"
	"image/jpeg"
	"io"
	"log"
	"os"
)

func main() {
	f, _ := os.Open("frame2.jpg")
	buff, _ := io.ReadAll(f)
	img, err := jpeg.Decode(bytes.NewReader(buff))
	if err != nil {
		log.Fatalln(err)
	}
	statBounds := img.Bounds()
	delta := uint8(5)
	targetR, targetG, targetB := uint8(254), uint8(0), uint8(0)
	var result = map[string]int{
		"red":     0,
		"not_red": 0,
	}
	for y := statBounds.Min.Y; y < statBounds.Max.Y; y++ {
		for x := statBounds.Min.X; x < statBounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			r8 := uint8(r >> 8)
			g8 := uint8(g >> 8)
			b8 := uint8(b >> 8)

			if withinDelta(r8, targetR, delta) &&
				withinDelta(g8, targetG, delta) &&
				withinDelta(b8, targetB, delta) {
				result["red"] = result["red"] + 1
				// Detected: mark with bright green
			} else {
				result["not_red"] = result["not_red"] + 1
			}
			//gray := uint8((uint16(r8) + uint16(g8) + uint16(b8)) / 3)
		}
	}
	fmt.Println("result", result)
}
func withinDelta(val, target, delta uint8) bool {
	if val >= target {
		return val-target <= delta
	}
	return target-val <= delta
}
