package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/otiai10/gosseract/v2"
	"gocv.io/x/gocv"
	"image"
	"log"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
	"time"
	"unsafe"
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
)

const (
	width             = 1920 // заміни на своє
	height            = 1080 // заміни на своє
	channels          = 3    // для bgr24
	frameSize         = width * height * channels
	WS_EX_LAYERED     = 0x00080000
	WS_EX_TRANSPARENT = 0x00000020
	WS_EX_TOPMOST     = 0x00000008
	WS_POPUP          = 0x80000000
	SW_SHOWNOACTIVATE = 4

	PS_SOLID           = 0
	BKMODE_TRANSPARENT = 1
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

func main() {
	//fmt.Println("GoCV version:", C.GoString(C.cvVersion()))
	// Replace with your RTSP/RTP stream
	//streamURL := "rtsp://admin:rfhm2tpx47@192.168.1.123:554/Preview_01_main"

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

	//go draw(hwnd, 200, 200, 600, 600)

	go func() {
		//cmd := exec.Command("ffmpeg",
		//	"-f", "gdigrab", // або інше джерело
		//	"-i", "desktop",
		//	"-pix_fmt", "bgr24",
		//	"-vcodec", "rawvideo",
		//	"-an", // без аудіо
		//	"-sn", // без субтитрів
		//	"-r", "10",
		//	"-f", "rawvideo",
		//	"-")
		//mainImg := gocv.IMRead("Screenshot 2025-06-26 192855.png", gocv.IMReadColor)
		//defer mainImg.Close()
		tmplImg := gocv.IMRead("mask.png", gocv.IMReadColor)
		defer tmplImg.Close()

		cmd := exec.Command("ffmpeg",
			"-f", "gdigrab", // screen capture
			"-framerate", "1", // 1 кадр/сек (зменши для тесту)
			"-i", "desktop",
			//"-vframes", "1", // лише один кадр
			"-f", "image2pipe",
			"-vcodec", "mjpeg", // або "png"
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

		//buf := make([]byte, frameSize)
		net := gocv.ReadNet("frozen_east_text_detection1.pb", "")
		if net.Empty() {
			log.Fatal("❌ Failed to load EAST model")
		}

		//fmt.Println(net.GetLayerNames())

		defer net.Close()
		client := gosseract.NewClient()
		defer client.Close()
		//client.SetLanguage("eng")
		reader := bufio.NewReader(stdout)
		img := gocv.NewMat()
		defer img.Close()
		var (
			minX uintptr
			minY uintptr
			maxX uintptr
			maxY uintptr
		)
		reg := regexp.MustCompile("\\s+")
		for {
			frame, err := readNextJPEGFrame(reader)
			fmt.Println("tick")
			if err != nil {
				fmt.Println("Read frame error:", err)
				break
			}
			img, err = gocv.IMDecode(frame, gocv.IMReadColor)
			// Create матрицю результату
			resultCols := img.Cols() - tmplImg.Cols() + 1
			resultRows := img.Rows() - tmplImg.Rows() + 1
			result := gocv.NewMatWithSize(resultRows, resultCols, gocv.MatTypeCV32F)
			//defer result.Close()

			//gocv.CvtColor(img, &img, gocv.ColorBGRToHSV)
			//gocv.CvtColor(tmplImg, &tmplImg, gocv.ColorBGRToHSV) // Template Matching
			//gocv.MatchTemplate(img, tmplImg, &result, gocv.TmCcoeffNormed, gocv.NewMat())
			gocv.MatchTemplate(img, tmplImg, &result, gocv.TmSqdiff, gocv.NewMat())

			// Знаходимо точку з найкращим збігом
			//_, maxVal, _, maxLoc := gocv.MinMaxLoc(result)
			minVal, _, maxLoc, _ := gocv.MinMaxLoc(result)
			if minVal <= 100000 {
				fmt.Printf("🔍 Min diff: %.6f at X=%d Y=%d\n", minVal, maxLoc.X, maxLoc.Y)
				//fmt.Printf("🎯 Match at: %sx%s (score: %.3f)\n", maxLoc.X, maxLoc.Y, maxVal)
				//if maxVal >= 0.8 {
				clearOverlay(hwnd)
				minX = uintptr(maxLoc.X - 328)
				minY = uintptr(maxLoc.Y + 130)
				maxX = uintptr(maxLoc.X + tmplImg.Size()[1])
				maxY = uintptr(maxLoc.Y + tmplImg.Size()[0])
				go draw(hwnd, minX, minY, maxX, maxY, "")
			}
			if minX != 0 && minY != 0 && maxX != 0 && maxY != 0 {
				rect := image.Rect(int(minX), int(minY), int(maxX), int(maxY)) // x1, y1, x2, y2
				roi := img.Region(rect)
				//defer roi.Close()
				buf, err := gocv.IMEncode(".png", roi)
				if err != nil {
					log.Fatal("❌ Не вдалося закодувати зображення:", err)
				}

				client.SetImageFromBytes(buf.GetBytes())

				text, err := client.Text()
				if err != nil {
					log.Println("❌ Помилка розпізнавання:", err)
					continue
				}
				text = reg.ReplaceAllString(strings.ReplaceAll(text, "\n", " "), " ")
				fmt.Println("🧾 Текст у прямокутнику:", text)
				if strings.Contains(text, "has been affected by your Dreaming Spirit") {
					go func() {
						for i := 30; i > 0; i-- {
							clearOverlay(hwnd)
							draw(hwnd, 0, 0, 0, 0, fmt.Sprintf("Dreaming spirit: %d", i))
							time.Sleep(time.Second)
						}
					}()
				}

			}
			//}
		}
	}()
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
	fmt.Println("end")
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
