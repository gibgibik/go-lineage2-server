package main

import (
	"fmt"
	"github.com/otiai10/gosseract/v2"
	"gocv.io/x/gocv"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
)

/*
#cgo pkg-config: lept tesseract
#cgo CXXFLAGS: -std=c++0x
#cgo CPPFLAGS: -Wno-unused-result
#include <stdlib.h>
#include <stdbool.h>
*/
import "C"

var (
	excludeBoundsArea = []image.Rectangle{
		image.Rect(0, 0, 247, 104),
		image.Rect(0, 590, 370, 1074),
		image.Rect(697, 915, 1273, 1074),
	}
)

func main() {
	threshold := 0.9
	nms := 0.4
	resizeWidth := 1920
	resizeHeight := 1088
	f, _ := os.Open("3.png")
	//img, _ := png.Decode(f)
	imgConfig, _ := png.DecodeConfig(f)
	mat := gocv.IMRead("3.png", gocv.IMReadColor)
	imgHeight := imgConfig.Height
	imgWidth := imgConfig.Width
	rW := float64(imgWidth) / float64(resizeWidth)
	rH := float64(imgHeight) / float64(resizeHeight)

	client := gosseract.NewClient()
	//client.SetLanguage("eng")
	defer client.Close()
	net := gocv.ReadNet("frozen_east_text_detection1.pb", "")
	defer net.Close()
	if net.Empty() {
		log.Fatal("❌ Failed to load EAST model")
	}
	// Prepare blob
	blob := gocv.BlobFromImage(mat, 1.0, image.Pt(int(resizeWidth), int(resizeHeight)), gocv.NewScalar(123.68, 116.78, 103.94, 0), true, false)
	defer blob.Close()
	net.SetInput(blob, "")
	// Define output layers
	outputNames := []string{"feature_fusion/Conv_7/Sigmoid", "feature_fusion/concat_3"}
	outputBlobs := net.ForwardLayers(outputNames)
	//
	//Decode results (this part can be tricky, depends on your task)
	scores := outputBlobs[0]
	geometry := outputBlobs[1]

	rotatedBoxes, confidences := decodeBoundingBoxes(scores, geometry, float32(threshold))
	boxes := []image.Rectangle{}
	for _, rotatedBox := range rotatedBoxes {
		//if !checkExcludeBox(rotatedBox.BoundingRect) {
		//	continue
		//}
		boxes = append(boxes, rotatedBox.BoundingRect)
	}
	// Only Apply NMS when there are at least one box
	indices := make([]int, len(boxes))
	if len(boxes) > 0 {
		indices = gocv.NMSBoxes(boxes, confidences, float32(threshold), float32(nms))
	}
	// Resize indices to only include those that have values other than zero
	var numIndices int = 0
	for _, value := range indices {
		if value != 0 {
			numIndices++
		}
	}
	indices = indices[0:numIndices]
	//return
	for i := 0; i < len(indices); i++ {
		// get 4 corners of the rotated rect
		verticesMat := gocv.NewMat()
		if err := gocv.BoxPoints(rotatedBoxes[indices[i]], &verticesMat); err != nil {
			log.Fatal(err)
		}

		//
		//	// scale the bounding box coordinates based on the respective ratios
		vertices := []image.Point{}
		var minX, minY, maxX, maxY int
		for j := 0; j < 4; j++ {
			p1 := image.Pt(
				int(verticesMat.GetFloatAt(j, 0)*float32(rW)),
				int(verticesMat.GetFloatAt(j, 1)*float32(rH)),
			)

			//p2 := image.Pt(
			//	int(verticesMat.GetFloatAt((j+1)%4, 0)*float32(rW)),
			//	int(verticesMat.GetFloatAt((j+1)%4, 1)*float32(rH)),
			//)
			if minX == 0 || minX > p1.X {
				minX = p1.X
			}
			if minY == 0 || minY > p1.Y {
				minY = p1.Y
			}
			if maxX == 0 || maxX < p1.X {
				maxX = p1.X
			}
			if maxY == 0 || maxY < p1.Y {
				maxY = p1.Y
			}
			vertices = append(vertices, p1)
			//gocv.Line(&mat, p1, p2, color.RGBA{0, 255, 0, 0}, 1)
		}
		rect := image.Rect(minX, minY, maxX, maxY)
		if !checkExcludeBox(rect) {
			continue
		}
		gocv.Rectangle(&mat, rect, color.RGBA{0, 255, 0, 0}, 1)
		cropped := fourPointsTransform(mat, gocv.NewPointVectorFromPoints(vertices))
		//_ = gocv.IMWrite("2.png", cropped)
		//return
		//gocv.CvtColor(mat, &cropped, gocv.ColorBGRToGray)

		// Create a 4D blob from cropped image
		blob = gocv.BlobFromImage(cropped, 1/127.5, image.Pt(128, 32), gocv.NewScalar(127.5, 0, 0, 0), false, false) //120?
		buf, _ := gocv.IMEncode(gocv.PNGFileExt, cropped)
		client.SetImageFromBytes(buf.GetBytes())
		fmt.Println(client.Text())
		//
		//// Run the recognition model
		////startTime = time.Now()
		//result := net.Forward("")
		////inferenceTime += time.Since(startTime)
		//
		// decode the result into text
		//wordRecognized := decodeText(result)
		//gocv.PutText(&img, wordRecognized, vertices[1], gocv.FontHersheySimplex, 0.5, color.RGBA{0, 0, 255, 0}, 1)

	}
	ok := gocv.IMWrite("2.png", mat)
	if !ok {
		log.Fatalf("Failed to write image")
	}
}

func checkExcludeBox(box image.Rectangle) bool {
	for _, excludeBox := range excludeBoundsArea {
		if excludeBox.Min.X <= box.Min.X && excludeBox.Min.Y <= box.Min.Y && excludeBox.Max.X >= box.Max.X && excludeBox.Max.Y >= box.Max.Y {
			return false
		}
	}
	return true
}

//func decodeText(scores gocv.Mat) string {
//	text := ""
//	alphabet := "0123456789abcdefghijklmnopqrstuvwxyz"
//
//	for i := 0; i < scores.Size()[0]; i++ {
//		scoresChannel := gocv.GetBlobChannel(scores, 0, i)
//		var c int = 0
//		var cScore float32 = 0
//		for j := 0; j < scores.Size()[2]; j++ {
//			score := scoresChannel.GetFloatAt(0, j)
//			if cScore < score {
//				c = j
//				cScore = score
//			}
//		}
//
//		if c != 0 {
//			text += string(alphabet[c-1])
//		} else {
//			text += "-"
//		}
//	}
//
//	// adjacent same letters as well as background text must be removed to get the final output
//	var charList strings.Builder
//	for i := 0; i < len(text); i++ {
//		if string(text[i]) != "-" && !(i > 0 && text[i] == text[i-1]) {
//			charList.WriteByte(text[i])
//		}
//	}
//
//	return charList.String()
//}

func decodeBoundingBoxes(scores gocv.Mat, geometry gocv.Mat, threshold float32) (detections []gocv.RotatedRect, confidences []float32) {
	scoresChannel := gocv.GetBlobChannel(scores, 0, 0)
	x0DataChannel := gocv.GetBlobChannel(geometry, 0, 0)
	x1DataChannel := gocv.GetBlobChannel(geometry, 0, 1)
	x2DataChannel := gocv.GetBlobChannel(geometry, 0, 2)
	x3DataChannel := gocv.GetBlobChannel(geometry, 0, 3)
	angleChannel := gocv.GetBlobChannel(geometry, 0, 4)

	for y := 0; y < scoresChannel.Rows(); y++ {
		for x := 0; x < scoresChannel.Cols(); x++ {

			// Extract data from scores
			score := scoresChannel.GetFloatAt(y, x)

			// If score is lower than threshold score, move to next x
			if score < threshold {
				continue
			}

			x0Data := x0DataChannel.GetFloatAt(y, x)
			x1Data := x1DataChannel.GetFloatAt(y, x)
			x2Data := x2DataChannel.GetFloatAt(y, x)
			x3Data := x3DataChannel.GetFloatAt(y, x)
			angle := angleChannel.GetFloatAt(y, x)

			// Calculate offset
			// Multiple by 4 because feature maps are 4 time less than input image.
			offsetX := x * 4.0
			offsetY := y * 4.0

			// Calculate cos and sin of angle
			cosA := math.Cos(float64(angle))
			sinA := math.Sin(float64(angle))

			h := x0Data + x2Data
			w := x1Data + x3Data

			// Calculate offset
			offset := []float64{
				(float64(offsetX) + cosA*float64(x1Data) + sinA*float64(x2Data)),
				(float64(offsetY) - sinA*float64(x1Data) + cosA*float64(x2Data)),
			}

			// Find points for rectangle
			p1 := []float64{
				(-sinA*float64(h) + offset[0]),
				(-cosA*float64(h) + offset[1]),
			}
			p3 := []float64{
				(-cosA*float64(w) + offset[0]),
				(sinA*float64(w) + offset[1]),
			}

			center := image.Pt(
				int(0.5*(p1[0]+p3[0])),
				int(0.5*(p1[1]+p3[1])),
			)

			detections = append(detections, gocv.RotatedRect{
				Points: []image.Point{
					{int(p1[0]), int(p1[1])},
					{int(p3[0]), int(p3[1])},
				},
				BoundingRect: image.Rect(
					int(p1[0]), int(p1[1]),
					int(p3[0]), int(p3[1]),
				),
				Center: center,
				Width:  int(w),
				Height: int(h),
				Angle:  float64(-1 * angle * 180 / math.Pi),
			})
			confidences = append(confidences, score)
		}
	}

	// Return detections and confidences
	return
}

func fourPointsTransform(frame gocv.Mat, vertices gocv.PointVector) gocv.Mat {
	outputSize := image.Pt(100, 32)
	targetVertices := gocv.NewPointVectorFromPoints([]image.Point{
		image.Pt(0, outputSize.Y-1),
		image.Pt(0, 0),
		image.Pt(outputSize.X-1, 0),
		image.Pt(outputSize.X-1, outputSize.Y-1),
	})

	result := gocv.NewMat()
	rotationMatrix := gocv.GetPerspectiveTransform(vertices, targetVertices)
	gocv.WarpPerspective(frame, &result, rotationMatrix, outputSize)

	return result
}
