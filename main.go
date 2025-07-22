package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/LA/internal"
	"image"
	"image/jpeg"
	"log"
	"math"
	"os/exec"
	"strings"
	"time"
)

func main() {
	internal.InitWinApi(mainRun)
}

func mainRun(hwnd uintptr) {
	internal.StartHttpServer()
	//file, err := os.Open("Untitled.png")
	//if err != nil {
	//	panic(err)
	//}
	//defer file.Close()
	//
	//// Декодуємо PNG
	//img, err := png.Decode(file)
	//if err != nil {
	//	panic(err)
	//}
	//
	//bounds := img.Bounds()
	//newImg := image.NewRGBA(bounds)
	//
	//// Порогове значення для "майже білого"
	//threshold := uint32(46000) // 65535 - це 100% білий (16-біт)
	//
	//// Проходимо по всіх пікселях
	//for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
	//	for x := bounds.Min.X; x < bounds.Max.X; x++ {
	//		c := img.At(x, y)
	//		r, g, b, _ := c.RGBA() // 16-бітні значення (0-65535)
	//		if r > threshold && g > threshold && b > threshold {
	//			newImg.Set(x, y, color.White)
	//		} else {
	//			newImg.Set(x, y, color.Black)
	//		}
	//	}
	//}
	//
	//// Зберігаємо PNG
	//outFile, err := os.Create("output1.png")
	//if err != nil {
	//	panic(err)
	//}
	//defer outFile.Close()
	//
	//err = png.Encode(outFile, newImg)
	//if err != nil {
	//	panic(err)
	//}
	//
	//println("Готово!")
	//return

	cmd := exec.Command("ffmpeg",
		"-f", "gdigrab", // screen capture
		"-framerate", "4", // 1 кадр/сек (зменши для тесту)
		//"-vframes", "1", // лише один кадр
		//"-video_size", "250x105",
		//"-video_size", "1920x1080",
		//"-offset_x", "1",
		//"-offset_y", "30",
		//"-show_region", "1",
		"-i", "desktop",
		"-f", "image2pipe",
		"-vcodec", "mjpeg", // або "png"
		"-q:v", "1",
		"-s", "1920x1080",
		"pipe:1",
	)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	reader := bufio.NewReader(stdout)
	//img := gocv.NewMat()
	//defer img.Close()

	//plStatClient := gosseract.NewClient()
	//plStatClient.SetVariable("tessedit_char_whitelist", "0123456789/% ")
	//defer plStatClient.Close()

	//npcClient := gosseract.NewClient()
	//npcClient.SetVariable("tessedit_char_whitelist", "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789 ")
	//defer npcClient.Close()

	//npcThreshold := 0.9
	//npcNms := 0.4
	//resizeWidth := 1920
	//resizeHeight := 1088
	//rW := float64(1920) / float64(resizeWidth)
	//rH := float64(1080) / float64(resizeHeight)
	//net := gocv.ReadNet("frozen_east_text_detection1.pb", "")
	//defer net.Close()
	for {
		frame, err := readNextJPEGFrame(reader)
		internal.CurrentImg.Lock()
		internal.CurrentImg.ImageJpeg = frame
		internal.CurrentImg.Unlock()
		//err = os.WriteFile("frame.jpg", frame, 0644)
		//if err != nil {
		//	panic(err)
		//}
		//return
		//fmt.Println(time.Now().Unix(), "tick")
		if err != nil {
			fmt.Println("Read frame error:", err)
			break
		}
		imgJpeg, err := jpeg.Decode(bytes.NewReader(frame))
		if err != nil {
			panic(err)
		}

		// Threshold value (0-255)
		//const threshold = 185
		//statRect := image.Rect(1, 30, 251, 135)
		//statImg := imgJpeg.(interface {
		//	SubImage(r image.Rectangle) image.Image
		//}).SubImage(statRect)
		//statBounds := statImg.Bounds()

		//newImg := image.NewGray(statBounds)
		//// Convert each pixel to grayscale + threshold
		//for y := statBounds.Min.Y; y < statBounds.Max.Y; y++ {
		//	for x := statBounds.Min.X; x < statBounds.Max.X; x++ {
		//		r, g, b, _ := statImg.At(x, y).RGBA()
		//		// Convert to 8-bit (0-255)
		//		r8 := uint8(r >> 8)
		//		g8 := uint8(g >> 8)
		//		b8 := uint8(b >> 8)
		//		// Grayscale average
		//		gray := uint8((uint16(r8) + uint16(g8) + uint16(b8)) / 3)
		//
		//		// Apply threshold
		//		if gray > threshold {
		//			newImg.SetGray(x, y, color.Gray{Y: 255}) // White
		//		} else {
		//			newImg.SetGray(x, y, color.Gray{Y: 0}) // Black
		//		}
		//	}
		//}
		//var buf bytes.Buffer
		//err = jpeg.Encode(&buf, newImg, nil)
		//if err != nil {
		//	log.Fatal(err)
		//}

		//var buf1 bytes.Buffer
		//err = jpeg.Encode(&buf1, targetImg, nil)
		//_ = os.WriteFile("frame2.jpg", buf1.Bytes(), 0644)
		//
		//if err != nil {
		//	log.Fatal(err)
		//}
		//
		//_ = os.WriteFile("frame1.jpg", buf.Bytes(), 0644)
		//return
		//if err != nil {
		//	panic(err)
		//}
		//return
		//img, err = gocv.IMDecode(buf.Bytes(), gocv.IMReadColor)
		//if err != nil {
		//	log.Fatal(err)
		//}
		//plStatClient.SetImageFromBytes(buf.Bytes())
		//text, err := plStatClient.Text()
		if err != nil {
			log.Fatal(err)
		}
		//pieces := strings.Split(text, "\n")
		targetRect := image.Rect(787, 0, 1133, 28)
		targetImg := imgJpeg.(interface {
			SubImage(r image.Rectangle) image.Image
		}).SubImage(targetRect)
		targetBounds := targetImg.Bounds()
		targetDelta := uint8(5)
		targetR, targetG, targetB := uint8(254), uint8(0), uint8(0)
		var targetResultRes int
		for x := targetBounds.Min.X; x < targetBounds.Max.X; x++ {
			r, g, b, _ := targetImg.At(x, 1).RGBA()
			r8 := uint8(r >> 8)
			g8 := uint8(g >> 8)
			b8 := uint8(b >> 8)

			if withinDelta(r8, targetR, targetDelta) &&
				withinDelta(g8, targetG, targetDelta) &&
				withinDelta(b8, targetB, targetDelta) {
				targetResultRes = targetResultRes + 1

				// Detected: mark with bright green
			}
			//gray := uint8((uint16(r8) + uint16(g8) + uint16(b8)) / 3)
		}
		percent := float64(targetResultRes) / (float64(targetBounds.Max.X-targetBounds.Min.X) / float64(100))
		internal.StatLock.Lock()
		lastUpdate := time.Now().Unix()
		internal.Stat.Target = struct {
			HpPercent  float64
			LastUpdate int64
		}{HpPercent: round(percent, 2), LastUpdate: lastUpdate}
		statsPointers := []image.Rectangle{
			{image.Point{33, 49}, image.Point{238, 49}},
			{image.Point{33, 66}, image.Point{238, 66}},
			{image.Point{33, 84}, image.Point{238, 84}},
		}
		colors := map[int]struct {
			match     int
			not_match int
		}{
			0: {match: 0, not_match: 0},
			1: {match: 0, not_match: 0},
			2: {match: 0, not_match: 0},
		}
		newTargetDelta := uint8(20)
		for idx, point := range statsPointers {
			for x := point.Min.X; x < point.Max.X; x++ {
				r, g, b, _ := imgJpeg.At(x, point.Min.Y).RGBA()

				r8 := uint8(r >> 8)
				g8 := uint8(g >> 8)
				b8 := uint8(b >> 8)
				match := false
				switch idx {
				case 0:
					if isYellow(r8, g8, b8, newTargetDelta) {
						match = true
					}
				case 1:
					if isRed(r8, g8, b8, newTargetDelta) {
						match = true
					}
				case 2:
					if isBlue(r8, g8, b8, newTargetDelta) {
						match = true
					}
				}
				if match {
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
			}
			percent = round(float64(colors[idx].match)/205*100, 2)
			switch idx {
			case 0:
				if percent > 0 {
					internal.Stat.CP = struct {
						Percent    float64
						LastUpdate int64
					}{Percent: percent, LastUpdate: lastUpdate}
				}
			case 1:
				if percent > 0 {
					internal.Stat.HP = struct {
						Percent    float64
						LastUpdate int64
					}{Percent: percent, LastUpdate: lastUpdate}
				}
			case 2:
				if percent > 0 {
					internal.Stat.MP = struct {
						Percent    float64
						LastUpdate int64
					}{Percent: percent, LastUpdate: lastUpdate}
				}
				//fmt.Println(float32(colors[idx].match) / 205 * 100)
				//return
			}
		}
		//for idx, piece := range pieces {
		//	piece = strings.TrimSpace(piece)
		//	switch idx {
		//	case 0:
		//		piece = replaceMidSlash(piece)
		//		if len(piece) > 3 {
		//			stat.CP = struct {
		//				Value      string
		//				LastUpdate int64
		//			}{Value: piece, LastUpdate: lastUpdate}
		//		}
		//	case 1:
		//		piece = replaceMidSlash(piece)
		//		if len(piece) > 3 {
		//			stat.HP = struct {
		//				Value      string
		//				LastUpdate int64
		//			}{Value: piece, LastUpdate: lastUpdate}
		//		}
		//	case 2:
		//		piece = replaceMidSlash(piece)
		//		if len(piece) > 3 {
		//			stat.MP = struct {
		//				Value      string
		//				LastUpdate int64
		//			}{Value: piece, LastUpdate: lastUpdate}
		//		}
		//	case 3:
		//		//stat.EXP = piece
		//	}
		//}
		internal.StatLock.Unlock()
		continue
		//mat, _ := gocv.IMDecode(frame, gocv.IMReadColor)
		//blob := gocv.BlobFromImage(mat, 1.0, image.Pt(int(resizeWidth), int(resizeHeight)), gocv.NewScalar(123.68, 116.78, 103.94, 0), true, false)
		//net.SetInput(blob, "")
		// Define output layers
		//outputNames := []string{"feature_fusion/Conv_7/Sigmoid", "feature_fusion/concat_3"}
		//outputBlobs := net.ForwardLayers(outputNames)
		//
		//Decode results (this part can be tricky, depends on your task)
		//scores := outputBlobs[0]
		//geometry := outputBlobs[1]

		//rotatedBoxes, confidences := decodeBoundingBoxes(scores, geometry, float32(npcThreshold))
		//boxes := []image.Rectangle{}
		//for _, rotatedBox := range rotatedBoxes {
		//	//if !checkExcludeBox(rotatedBox.BoundingRect) {
		//	//	continue
		//	//}
		//	boxes = append(boxes, rotatedBox.BoundingRect)
		//}
		//// Only Apply NMS when there are at least one box
		//indices := make([]int, len(boxes))
		//if len(boxes) > 0 {
		//	indices = gocv.NMSBoxes(boxes, confidences, float32(npcThreshold), float32(npcNms))
		//}
		//// Resize indices to only include those that have values other than zero
		//var numIndices int = 0
		//for _, value := range indices {
		//	if value != 0 {
		//		numIndices++
		//	}
		//}
		//indices = indices[0:numIndices]
		//internal.ClearOverlay(hwnd) //@todo uncomment
		//for i := 0; i < len(indices); i++ {
		//	// get 4 corners of the rotated rect
		//	verticesMat := gocv.NewMat()
		//	if err := gocv.BoxPoints(rotatedBoxes[indices[i]], &verticesMat); err != nil {
		//		log.Fatal(err)
		//	}
		//
		//	//
		//	//	// scale the bounding box coordinates based on the respective ratios
		//	vertices := []image.Point{}
		//	var minX, minY, maxX, maxY int
		//	for j := 0; j < 4; j++ {
		//		p1 := image.Pt(
		//			int(verticesMat.GetFloatAt(j, 0)*float32(rW)),
		//			int(verticesMat.GetFloatAt(j, 1)*float32(rH)),
		//		)
		//
		//		//p2 := image.Pt(
		//		//	int(verticesMat.GetFloatAt((j+1)%4, 0)*float32(rW)),
		//		//	int(verticesMat.GetFloatAt((j+1)%4, 1)*float32(rH)),
		//		//)
		//		if minX == 0 || minX > p1.X {
		//			minX = p1.X
		//		}
		//		if minY == 0 || minY > p1.Y {
		//			minY = p1.Y
		//		}
		//		if maxX == 0 || maxX < p1.X {
		//			maxX = p1.X
		//		}
		//		if maxY == 0 || maxY < p1.Y {
		//			maxY = p1.Y
		//		}
		//		vertices = append(vertices, p1)
		//		//gocv.Line(&mat, p1, p2, color.RGBA{0, 255, 0, 0}, 1)
		//	}
		//	rect := image.Rect(minX, minY, maxX, maxY)
		//	if !checkExcludeBox(rect) {
		//		continue
		//	}
		//
		//	//continue //@todo remove
		//	//cropped := fourPointsTransform(mat, gocv.NewPointVectorFromPoints(vertices))
		//	//// Create a 4D blob from cropped image
		//	////blob = gocv.BlobFromImage(cropped, 1/127.5, image.Pt(128, 32), gocv.NewScalar(127.5, 0, 0, 0), false, false) //120?
		//	//buf, _ := gocv.IMEncode(gocv.PNGFileExt, cropped)
		//	//npcClient.SetImageFromBytes(buf.GetBytes())
		//	//foundText, _ := npcClient.Text()
		//	//
		//	//if _, ok := internal.NpcList[foundText]; ok {
		//	//gocv.Rectangle(&mat, rect, color.RGBA{0, 255, 0, 0}, 1)
		//
		//	go internal.Draw(hwnd, uintptr(rect.Min.X), uintptr(rect.Min.Y), uintptr(rect.Max.X), uintptr(rect.Max.Y), "")
		//	//_ = gocv.IMWrite("output.png", mat)
		//
		//	//fmt.Println(1)
		//	//}
		//
		//}
		//_ = blob.Close()
	}
} // Function to check if the pixel is blue based on the threshold
func isBlue(r, g, b, threshold uint8) bool {
	return b > r+threshold && b > g+threshold
}
func isRed(r, g, b, threshold uint8) bool {
	return r > g+threshold && r > b+threshold
}
func isYellow(r, g, b, threshold uint8) bool {
	// Yellow is when both red and green are higher than blue with the threshold
	return r > b+threshold && g > b+threshold
}

func round(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}
func withinDelta(val, target, delta uint8) bool {
	if val >= target {
		return val-target <= delta
	}
	return target-val <= delta
}

func replaceMidSlash(s string) string {
	var offset int
	s = strings.ReplaceAll(s, " ", "")
	strl := len(s)
	r := []rune(s)
	if strl%2 == 0 {
		offset = int(math.Ceil(float64(strl) / 2))
		return string(append(r[:offset], append([]rune("/"), r[offset:]...)...))
	} else {
		offset = int(math.Floor(float64(strl) / 2))
		r[offset] = []rune("/")[0]
		return string(r)
	}
}

//net := gocv.ReadNet("frozen_east_text_detection1.pb", "")
//if net.Empty() {
//	log.Fatal("❌ Failed to load EAST model")
//}
//defer net.Close()
//img1 := gocv.IMRead("output1.png", gocv.IMReadAnyColor)
//if img1.Empty() {
//	fmt.Println("Error reading image")
//	return
//}
//client := gosseract.NewClient()
//client.SetVariable("tessedit_char_whitelist", "0123456789/%")
//defer client.Close()
//buf, err := gocv.IMEncode(".png", img1)
//if err != nil {
//	log.Fatal("err:", err)
//}
//client.SetImageFromBytes(buf.GetBytes())
//
//text, err := client.Text()
//if err != nil {
//	log.Println("❌ Помилка розпізнавання:", err)
//	return
//}
////text = reg.ReplaceAllString(strings.ReplaceAll(text, "\n", " "), " ")
//fmt.Println("🧾 Текст у прямокутнику:", text)

func readNextJPEGFrame(r *bufio.Reader) ([]byte, error) {
	var buf bytes.Buffer
	started := false
	var last byte

	// Start reading the stream byte by byte
	for {
		b, err := r.ReadByte()
		if err != nil {
			// Return any errors while reading the byte (e.g., end of stream)
			if err.Error() == "EOF" {
				log.Println("End of stream reached")
			} else {
				log.Printf("Error reading byte: %v", err)
			}
			return nil, fmt.Errorf("error reading byte from buffer: %w", err)
		}

		// Check if we're starting the JPEG frame
		if !started {
			if last == 0xFF && b == 0xD8 {
				// Begin frame when we find the start marker 0xFF 0xD8
				buf.WriteByte(0xFF)
				buf.WriteByte(0xD8)
				started = true
			}
			last = b
			continue
		}

		// Write byte to buffer
		buf.WriteByte(b)

		// Check for the end of the JPEG frame
		if last == 0xFF && b == 0xD9 {
			return buf.Bytes(), nil
		}

		last = b
	}
}
