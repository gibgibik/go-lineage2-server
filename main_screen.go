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
	"bufio"
	"bytes"
	json2 "encoding/json"
	"errors"
	"fmt"
	"github.com/otiai10/gosseract/v2"
	"gocv.io/x/gocv"
	"image"
	"image/color"
	"image/jpeg"
	"log"
	"math"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

type Stat struct {
	CP struct {
		Value      string
		LastUpdate int64
	}
	HP struct {
		Value      string
		LastUpdate int64
	}
	MP struct {
		Value      string
		LastUpdate int64
	}
	EXP struct {
		Value      string
		LastUpdate int64
	}
	Target struct {
		HpPercent  float64
		LastUpdate int64
	}
}

const (
	WS_EX_LAYERED     = 0x00080000
	WS_EX_TRANSPARENT = 0x00000020
	WS_EX_TOPMOST     = 0x00000008
	WS_POPUP          = 0x80000000
	SW_SHOWNOACTIVATE = 4

	PS_SOLID           = 0
	BKMODE_TRANSPARENT = 1
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procCreateWindowExW  = user32.NewProc("CreateWindowExW")
	procDefWindowProcW   = user32.NewProc("DefWindowProcW")
	procDispatchMessageW = user32.NewProc("DispatchMessageW")
	procGetMessageW      = user32.NewProc("GetMessageW")
	procRegisterClassExW = user32.NewProc("RegisterClassExW")
	procSetLayeredAttrs  = user32.NewProc("SetLayeredWindowAttributes")
	procShowWindow       = user32.NewProc("ShowWindow")
	procUpdateWindow     = user32.NewProc("UpdateWindow")
	procTranslateMessage = user32.NewProc("TranslateMessage")
	procGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
	procGetClientRect    = user32.NewProc("GetClientRect")
	procCreateSolidBrush = gdi32.NewProc("CreateSolidBrush")
	procFillRect         = user32.NewProc("FillRect")

	procGetDC          = user32.NewProc("GetDC")
	procReleaseDC      = user32.NewProc("ReleaseDC")
	procRectangle      = gdi32.NewProc("Rectangle")
	procGetStockObject = gdi32.NewProc("GetStockObject")
	procCreatePen      = gdi32.NewProc("CreatePen")
	procSelectObject   = gdi32.NewProc("SelectObject")
	procDeleteObject   = gdi32.NewProc("DeleteObject")
	procSetTextColor   = gdi32.NewProc("SetTextColor")
	procSetBkMode      = gdi32.NewProc("SetBkMode")
	procCreateFont     = gdi32.NewProc("CreateFontW")
	procTextOutW       = gdi32.NewProc("TextOutW")
	excludeBoundsArea  = []image.Rectangle{
		image.Rect(0, 0, 247, 104),
		image.Rect(0, 590, 370, 1074),
		image.Rect(697, 915, 1273, 1074),
	}
)

type POINT struct{ X, Y int32 }
type MSG struct {
	Hwnd    syscall.Handle
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}
type WNDCLASSEX struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     syscall.Handle
	HIcon         syscall.Handle
	HCursor       syscall.Handle
	HbrBackground syscall.Handle
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       syscall.Handle
}

func createRequestError(w http.ResponseWriter, err string, code int) {
	w.WriteHeader(code)
	_, _ = w.Write([]byte(err))
}

func main() {
	className := syscall.StringToUTF16Ptr("TransparentOverlay")
	hInstance, _, _ := procGetModuleHandleW.Call(0)
	// Minimal WndProc
	wndProc := syscall.NewCallback(func(hwnd syscall.Handle, msg uint32, wParam, lParam uintptr) uintptr {
		return callDefWindowProc(hwnd, msg, wParam, lParam)
	})
	// Register class
	wc := WNDCLASSEX{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEX{})),
		LpfnWndProc:   wndProc,
		HInstance:     syscall.Handle(hInstance),
		LpszClassName: className,
	}
	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))

	// Create overlay window
	hwnd, _, _ := procCreateWindowExW.Call(
		WS_EX_LAYERED|WS_EX_TRANSPARENT|WS_EX_TOPMOST,
		uintptr(unsafe.Pointer(className)),
		0,
		WS_POPUP,
		0, 0, 1920, 1080,
		0, 0, hInstance, 0,
	)

	// Make transparent
	procSetLayeredAttrs.Call(hwnd, 0, 0, 0x00000001)

	// Show
	procShowWindow.Call(hwnd, SW_SHOWNOACTIVATE)
	procUpdateWindow.Call(hwnd)
	var msg MSG
	go mainRun(hwnd)
	fmt.Println("end")
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if r == 0 {
			fmt.Println("win window exit")
			return
		}
		if int32(r) != 0 {
			procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
			procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
		}
		time.Sleep(10 * time.Millisecond)
		fmt.Println("tick main")
	}
}

func mainRun(hwnd uintptr) {
	var stat Stat
	var statLock sync.RWMutex
	handle := &http.Server{
		Addr:         ":2223",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		statLock.RLock()
		json, err := json2.Marshal(stat)
		if err != nil {
			createRequestError(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		writer.Write(json)
		defer statLock.RUnlock()
	})
	go func() {
		log.Println("starting server")
		if err := handle.ListenAndServe(); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				log.Println("http server fatal error: " + err.Error())
			}
			return
		}
	}()

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
		"-framerate", "2", // 1 кадр/сек (зменши для тесту)
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
		//"-s", "1920x1080",
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
	img := gocv.NewMat()
	defer img.Close()
	plStatClient := gosseract.NewClient()
	plStatClient.SetVariable("tessedit_char_whitelist", "0123456789/% ")
	defer plStatClient.Close()

	npcClient := gosseract.NewClient()
	npcClient.SetVariable("tessedit_char_whitelist", "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789 ")
	defer npcClient.Close()
	npcThreshold := 0.9
	npcNms := 0.4
	resizeWidth := 1920
	resizeHeight := 1088
	rW := float64(1920) / float64(resizeWidth)
	rH := float64(1080) / float64(resizeHeight)
	net := gocv.ReadNet("frozen_east_text_detection1.pb", "")
	defer net.Close()
	for {
		frame, err := readNextJPEGFrame(reader)
		//err = os.WriteFile("frame.jpg", frame, 0644)
		//if err != nil {
		//	panic(err)
		//}
		//return
		fmt.Println(time.Now().Unix(), "tick")
		if err != nil {
			fmt.Println("Read frame error:", err)
			break
		}
		imgJpeg, err := jpeg.Decode(bytes.NewReader(frame))
		if err != nil {
			panic(err)
		}

		// Threshold value (0-255)
		const threshold = 185
		statRect := image.Rect(1, 30, 251, 135)
		statImg := imgJpeg.(interface {
			SubImage(r image.Rectangle) image.Image
		}).SubImage(statRect)
		statBounds := statImg.Bounds()

		newImg := image.NewGray(statBounds)
		// Convert each pixel to grayscale + threshold
		for y := statBounds.Min.Y; y < statBounds.Max.Y; y++ {
			for x := statBounds.Min.X; x < statBounds.Max.X; x++ {
				r, g, b, _ := statImg.At(x, y).RGBA()
				// Convert to 8-bit (0-255)
				r8 := uint8(r >> 8)
				g8 := uint8(g >> 8)
				b8 := uint8(b >> 8)
				// Grayscale average
				gray := uint8((uint16(r8) + uint16(g8) + uint16(b8)) / 3)

				// Apply threshold
				if gray > threshold {
					newImg.SetGray(x, y, color.Gray{Y: 255}) // White
				} else {
					newImg.SetGray(x, y, color.Gray{Y: 0}) // Black
				}
			}
		}
		var buf bytes.Buffer
		err = jpeg.Encode(&buf, newImg, nil)
		if err != nil {
			log.Fatal(err)
		}

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
		plStatClient.SetImageFromBytes(buf.Bytes())
		text, err := plStatClient.Text()
		if err != nil {
			log.Fatal(err)
		}
		pieces := strings.Split(text, "\n")
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
		statLock.Lock()
		lastUpdate := time.Now().Unix()
		stat.Target = struct {
			HpPercent  float64
			LastUpdate int64
		}{HpPercent: round(percent, 2), LastUpdate: lastUpdate}
		for idx, piece := range pieces {
			piece = strings.TrimSpace(piece)
			switch idx {
			case 0:
				piece = replaceMidSlash(piece)
				if len(piece) > 3 {
					stat.CP = struct {
						Value      string
						LastUpdate int64
					}{Value: piece, LastUpdate: lastUpdate}
				}
			case 1:
				piece = replaceMidSlash(piece)
				if len(piece) > 3 {
					stat.HP = struct {
						Value      string
						LastUpdate int64
					}{Value: piece, LastUpdate: lastUpdate}
				}
			case 2:
				piece = replaceMidSlash(piece)
				if len(piece) > 3 {
					stat.MP = struct {
						Value      string
						LastUpdate int64
					}{Value: piece, LastUpdate: lastUpdate}
				}
			case 3:
				//stat.EXP = piece
			}
		}
		statsPointers := []struct {
			delta []uint8
			rest  image.Rectangle
		}{
			{delta: []uint8{71, 60, 22}, rest: image.Rectangle{image.Point{33, 49}, image.Point{238, 49}}},
			{delta: []uint8{104, 34, 22}, rest: image.Rectangle{image.Point{33, 66}, image.Point{238, 66}}},
			{delta: []uint8{24, 67, 107}, rest: image.Rectangle{image.Point{33, 84}, image.Point{238, 84}}},
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
			for x := point.rest.Min.X; x < point.rest.Max.X; x++ {
				r, g, b, _ := imgJpeg.At(x, point.rest.Min.Y).RGBA()

				r8 := uint8(r >> 8)
				g8 := uint8(g >> 8)
				b8 := uint8(b >> 8)
				//if x == 40 && point.rest.Max.Y == 84 {
				//	fmt.Printf("Pixel at (%d, %d): R=%d, G=%d, B=%d\n", x, point.rest.Min.Y, r8, g8, b8)
				//	fmt.Println(r8, g8, b8)
				//	return
				//}
				//if withinDelta(r8, point.delta[0], newTargetDelta) &&
				//	withinDelta(g8, point.delta[1], newTargetDelta) &&
				//	withinDelta(b8, point.delta[2], newTargetDelta) {
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
			//return
		}
		fmt.Println(colors)
		statLock.Unlock()
		continue
		mat, _ := gocv.IMDecode(frame, gocv.IMReadColor)
		blob := gocv.BlobFromImage(mat, 1.0, image.Pt(int(resizeWidth), int(resizeHeight)), gocv.NewScalar(123.68, 116.78, 103.94, 0), true, false)
		net.SetInput(blob, "")
		// Define output layers
		outputNames := []string{"feature_fusion/Conv_7/Sigmoid", "feature_fusion/concat_3"}
		outputBlobs := net.ForwardLayers(outputNames)
		//
		//Decode results (this part can be tricky, depends on your task)
		scores := outputBlobs[0]
		geometry := outputBlobs[1]

		rotatedBoxes, confidences := decodeBoundingBoxes(scores, geometry, float32(npcThreshold))
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
			indices = gocv.NMSBoxes(boxes, confidences, float32(npcThreshold), float32(npcNms))
		}
		// Resize indices to only include those that have values other than zero
		var numIndices int = 0
		for _, value := range indices {
			if value != 0 {
				numIndices++
			}
		}
		indices = indices[0:numIndices]
		clearOverlay(hwnd) //@todo uncomment
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

			//continue //@todo remove
			//cropped := fourPointsTransform(mat, gocv.NewPointVectorFromPoints(vertices))
			//// Create a 4D blob from cropped image
			////blob = gocv.BlobFromImage(cropped, 1/127.5, image.Pt(128, 32), gocv.NewScalar(127.5, 0, 0, 0), false, false) //120?
			//buf, _ := gocv.IMEncode(gocv.PNGFileExt, cropped)
			//npcClient.SetImageFromBytes(buf.GetBytes())
			//foundText, _ := npcClient.Text()
			//
			//if _, ok := internal.NpcList[foundText]; ok {
			//gocv.Rectangle(&mat, rect, color.RGBA{0, 255, 0, 0}, 1)

			go draw(hwnd, uintptr(rect.Min.X), uintptr(rect.Min.Y), uintptr(rect.Max.X), uintptr(rect.Max.Y), "")
			//_ = gocv.IMWrite("output.png", mat)

			//fmt.Println(1)
			//}

		}
		_ = blob.Close()
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

func createFont(height int32) uintptr {
	hFont, _, _ := procCreateFont.Call(
		uintptr(height), 0, 0, 0, // height, width, escapement, orientation
		400, 0, 0, 0, // weight, italic, underline, strikeout
		1, 0, 0, 0, // charset, outPrecision, clipPrecision, quality
		uintptr(1), // pitchAndFamily
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("Arial"))),
	)
	return hFont
}
func draw(hwnd uintptr, left uintptr, top uintptr, right uintptr, bottom uintptr, text string) {
	hdc, _, _ := procGetDC.Call(hwnd)
	if left != 0 && top != 0 && right != 0 && bottom != 0 {

		// Create red pen (BGR format: 0x00RRGGBB → 0x000000FF = red)
		pen, _, _ := procCreatePen.Call(PS_SOLID, 3, 0x008000)
		oldPen, _, _ := procSelectObject.Call(hdc, pen)

		// Select NULL_BRUSH to avoid filling the rectangle
		nullBrush, _, _ := procGetStockObject.Call(5)
		oldBrush, _, _ := procSelectObject.Call(hdc, nullBrush)

		// Draw transparent (non-filled) red rectangle
		procRectangle.Call(hdc, left, top, right, bottom)
		procSelectObject.Call(hdc, oldPen)
		procSelectObject.Call(hdc, oldBrush)
		procDeleteObject.Call(pen)
	}

	if text != "" {
		font := createFont(48)
		hdc, _, _ := procGetDC.Call(hwnd)
		procSelectObject.Call(hdc, font)
		defer procDeleteObject.Call(font) //
		procSetTextColor.Call(hdc, 0x00FF00)
		procSetBkMode.Call(hdc, BKMODE_TRANSPARENT)
		tx := syscall.StringToUTF16Ptr(text)
		procTextOutW.Call(hdc, 770, 100, uintptr(unsafe.Pointer(tx)), uintptr(len(text)))
	}
	// Optionally draw text (commented out for clarity)
	//
	//

	// Cleanup

	procReleaseDC.Call(hwnd, hdc)
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

	for {
		b, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		if !started {
			if last == 0xFF && b == 0xD8 {
				buf.WriteByte(0xFF)
				buf.WriteByte(0xD8)
				started = true
			}
			last = b
			continue
		}
		buf.WriteByte(b)
		if last == 0xFF && b == 0xD9 {
			return buf.Bytes(), nil
		}
		last = b
	}
}

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

func checkExcludeBox(box image.Rectangle) bool {
	for _, excludeBox := range excludeBoundsArea {
		if excludeBox.Min.X <= box.Min.X && excludeBox.Min.Y <= box.Min.Y && excludeBox.Max.X >= box.Max.X && excludeBox.Max.Y >= box.Max.Y {
			return false
		}
	}
	return true
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

func callDefWindowProc(hwnd syscall.Handle, msg uint32, wParam, lParam uintptr) uintptr {
	ret, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return ret
}

func clearOverlay(hwnd uintptr) {
	var rect struct {
		Left, Top, Right, Bottom int32
	}
	procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rect)))

	hdc, _, _ := procGetDC.Call(hwnd)

	// Brush: match SetLayeredWindowAttributes color key (e.g., black)
	brush, _, _ := procCreateSolidBrush.Call(0x000000)

	procFillRect.Call(hdc, uintptr(unsafe.Pointer(&rect)), brush)

	procDeleteObject.Call(brush)
	procReleaseDC.Call(hwnd, hdc)
}
