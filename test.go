package main

/*
#cgo pkg-config: lept tesseract
#cgo CXXFLAGS: -std=c++0x
#cgo CPPFLAGS: -Wno-unused-result
#include <stdlib.h>
#include <stdbool.h>
*/
import "C"
import (
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"strconv"
)

func main() {
	// Open the existing PNG file
	file, err := os.Open("1.jpg")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	// Decode PNG
	jpegImg, err := jpeg.Decode(file)
	if err != nil {
		panic(err)
	}
	// Create an RGBA image from the decoded image
	statsPointers := []struct {
		delta []uint8
		rest  image.Rectangle
	}{
		{delta: []uint8{192, 152, 45}, rest: image.Rectangle{image.Point{33, 49}, image.Point{238, 49}}},
		{delta: []uint8{195, 64, 44}, rest: image.Rectangle{image.Point{33, 66}, image.Point{238, 66}}},
		{delta: []uint8{41, 123, 194}, rest: image.Rectangle{image.Point{33, 84}, image.Point{238, 84}}},
	}

	//targetR, targetG, targetB := uint8(254), uint8(0), uint8(0)
	colors := map[int]struct {
		match     int
		not_match int
	}{
		0: {match: 0, not_match: 0},
		1: {match: 0, not_match: 0},
		2: {match: 0, not_match: 0},
	}
	targetDelta := uint8(5)
	tt := make(map[string]int, 0)
	for idx, point := range statsPointers {
		for x := point.rest.Min.X; x < point.rest.Max.X; x++ {
			r, g, b, _ := jpegImg.At(x, point.rest.Min.Y).RGBA()
			r8 := uint8(r >> 8)
			g8 := uint8(g >> 8)
			b8 := uint8(b >> 8)
			key := fmt.Sprintf("%s_%s_%s", strconv.Itoa(int(r8)), strconv.Itoa(int(g8)), strconv.Itoa(int(b8)))
			if _, ok := tt[key]; !ok {
				tt[key] = 0
			}
			tt[key] += 1
			if withinDelta(r8, point.delta[0], targetDelta) &&
				withinDelta(g8, point.delta[1], targetDelta) &&
				withinDelta(b8, point.delta[2], targetDelta) {
				//fmt.Println("match")
				colors[idx] = struct {
					match     int
					not_match int
				}{match: colors[idx].match + 1, not_match: colors[idx].not_match}
			} else {
				colors[idx] = struct {
					match     int
					not_match int
				}{match: colors[idx].match, not_match: colors[idx].not_match + 1}
			}
			//Detected: mark with bright green
			//}
			//// Grayscale average
			//gray := uint8((uint16(r8) + uint16(g8) + uint16(b8)) / 3)
			//fmt.Println(r8, g8, b8)
			//if _, ok := colors[cAt]; !ok {
			//	colors[cAt] = 0
			//}
			//colors[cAt]++
		}
		fmt.Println(float32(colors[idx].match) / 2.5)
		fmt.Println(tt)
		return
		//return
	}
	fmt.Println(colors)
	return

}

func withinDelta(val, target, delta uint8) bool {
	if val >= target {
		return val-target <= delta
	}
	return target-val <= delta
}
