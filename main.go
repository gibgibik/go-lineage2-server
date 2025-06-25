package main

/*
#cgo pkg-config: opencv4
#include <opencv2/core/version.hpp>
const char* cvVersion() {
    return CV_VERSION;
}
*/
import (
	"fmt"
	"gocv.io/x/gocv"
	"image"
	"image/color"
	"io"
	"log"
	"math"
	"os/exec"
	"syscall"
	"time"
	"unsafe"
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
		cmd := exec.Command("ffmpeg",
			"-f", "gdigrab", // або інше джерело
			"-i", "desktop",
			"-pix_fmt", "bgr24",
			"-vcodec", "rawvideo",
			"-an", // без аудіо
			"-sn", // без субтитрів
			"-r", "10",
			"-f", "rawvideo",
			"-")

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Fatal(err)
		}

		if err := cmd.Start(); err != nil {
			log.Fatal(err)
		}

		resized := gocv.NewMat()
		defer resized.Close()

		frame := gocv.NewMat()
		defer frame.Close()

		gray := gocv.NewMat()
		defer gray.Close()

		prevGray := gocv.NewMat()
		defer prevGray.Close()
		fullHD := image.Pt(1920, 1080)
		_ = image.Pt(1920, 1080)

		buf := make([]byte, frameSize)
		net := gocv.ReadNet("frozen_east_text_detection.pb", "")
		defer net.Close()
		outputs := []gocv.Mat{
			gocv.NewMat(),
			gocv.NewMat(),
		}
		defer func() {
			for _, m := range outputs {
				m.Close()
			}
		}()
		net.ForwardLayers([]string{"feature_fusion/Conv_7/Sigmoid", "feature_fusion/concat_3"})
		for {
			fmt.Println("tick")

			// читаємо один кадр
			_, err := io.ReadFull(stdout, buf)
			if err != nil {
				log.Println("End of stream or error:", err)
				break
			}
			//go clearOverlay(hwnd)
			fmt.Println("tick")
			frame, _ = gocv.NewMatFromBytes(height, width, gocv.MatTypeCV8UC3, buf)
			buf = make([]byte, frameSize)
			if frame.Empty() {
				log.Println("Empty frame or error")
				continue
			}
			boxes, _ := decodeEAST(outputs[0], outputs[1], 0.5)
			fmt.Println(boxes)
			gocv.Resize(frame, &resized, fullHD, 0, 0, gocv.InterpolationLinear)
			gocv.CvtColor(resized, &gray, gocv.ColorBGRToGray)
			gocv.Threshold(gray, &gray, 0, 255, gocv.ThresholdBinaryInv+gocv.ThresholdOtsu)
			//gocv.GaussianBlur(gray, &gray, image.Pt(21, 21), 0, 0, gocv.BorderDefault)
			if !prevGray.Empty() {
				diff := gocv.NewMat()
				gocv.AbsDiff(gray, prevGray, &diff)

				thresh := gocv.NewMat()
				gocv.Threshold(diff, &thresh, 25, 255, gocv.ThresholdBinary)
				diff.Close()

				gocv.Dilate(thresh, &thresh, gocv.NewMat())

				contours := gocv.FindContours(thresh, gocv.RetrievalExternal, gocv.ChainApproxSimple)
				thresh.Close()

				clearOverlay(hwnd)
				for i := 0; i < contours.Size(); i++ {
					c := contours.At(i)

					//area := gocv.ContourArea(c)
					//if area < 1000 {
					//	continue
					//}
					//if area > 1000 {
					//	rect := gocv.BoundingRect(c)
					//	gocv.Rectangle(&resized, rect, color.RGBA{0, 255, 0, 0}, 2)
					//}

					rect := gocv.BoundingRect(c)
					if rect.Dx() > 50 && rect.Dy() > 20 {
						go draw(hwnd, uintptr(rect.Min.X), uintptr(rect.Min.Y), uintptr(rect.Max.X), uintptr(rect.Max.Y))
					}
					fmt.Println(rect)
					//gocv.Rectangle(&resized, rect, color.RGBA{0, 255, 0, 0}, 2)
				}
				contours.Close()
			}
			gray.CopyTo(&prevGray)
			//if window.WaitKey(1) == 27 { // ESC
			//	break
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

func draw(hwnd uintptr, left uintptr, top uintptr, right uintptr, bottom uintptr) {
	hdc, _, _ := procGetDC.Call(hwnd)
	// Create red pen (BGR format: 0x00RRGGBB → 0x000000FF = red)
	pen, _, _ := procCreatePen.Call(PS_SOLID, 3, 0x000000FF)
	oldPen, _, _ := procSelectObject.Call(hdc, pen)

	// Select NULL_BRUSH to avoid filling the rectangle
	nullBrush, _, _ := procGetStockObject.Call(5)
	oldBrush, _, _ := procSelectObject.Call(hdc, nullBrush)

	// Draw transparent (non-filled) red rectangle
	procRectangle.Call(hdc, left, top, right, bottom)

	// Optionally draw text (commented out for clarity)
	// procSetTextColor.Call(hdc, 0x00FFFFFF) // white
	// procSetBkMode.Call(hdc, BKMODE_TRANSPARENT)
	// text := syscall.StringToUTF16Ptr("Hello Overlay")
	// procTextOutW.Call(hdc, 110, 130, uintptr(unsafe.Pointer(text)), uintptr(len("Hello Overlay")))

	// Cleanup
	procSelectObject.Call(hdc, oldPen)
	procSelectObject.Call(hdc, oldBrush)
	procDeleteObject.Call(pen)
	procReleaseDC.Call(hwnd, hdc)
	//hdc, _, _ := procGetDC.Call(hwnd)
	//
	//// Red pen
	//pen, _, _ := procCreatePen.Call(PS_SOLID, 3, 0x000000FF) // red: BGR
	//oldPen, _, _ := procSelectObject.Call(hdc, pen)
	//
	//// Rectangle
	//procRectangle.Call(hdc, left, top, right, bottom)
	//
	//// Text settings
	////procSetTextColor.Call(hdc, 0x00FFFFFF) // white
	////procSetBkMode.Call(hdc, BKMODE_TRANSPARENT)
	////text := syscall.StringToUTF16Ptr("Hello Overlay")
	////procTextOutW.Call(hdc, 110, 130, uintptr(unsafe.Pointer(text)), uintptr(len("Hello Overlay")))
	//
	//// Cleanup
	//procSelectObject.Call(hdc, oldPen)
	//procDeleteObject.Call(pen)
	//procReleaseDC.Call(hwnd, hdc)
}

func colorRGBA(r, g, b uint8) color.RGBA {
	return color.RGBA{R: r, G: g, B: b, A: 255}
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
func decodeEAST(scores, geometry gocv.Mat, confThreshold float32) ([]image.Rectangle, []float32) {
	sz := scores.Size()
	height := sz[2]
	width := sz[3]

	var boxes []image.Rectangle
	var confidences []float32

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			score := scores.GetFloatAt(0, 0, y, x)
			if score < confThreshold {
				continue
			}

			offsetX := float32(x) * 4.0
			offsetY := float32(y) * 4.0

			angle := geometry.GetFloatAt(0, 4, y, x)
			cosA := float32(math.Cos(float64(angle)))
			sinA := float32(math.Sin(float64(angle)))

			h := geometry.GetFloatAt(0, 0, y, x) + geometry.GetFloatAt(0, 2, y, x)
			w := geometry.GetFloatAt(0, 1, y, x) + geometry.GetFloatAt(0, 3, y, x)

			endX := offsetX + cosA*geometry.GetFloatAt(0, 1, y, x) + sinA*geometry.GetFloatAt(0, 2, y, x)
			endY := offsetY - sinA*geometry.GetFloatAt(0, 1, y, x) + cosA*geometry.GetFloatAt(0, 2, y, x)
			startX := endX - w
			startY := endY - h

			box := image.Rect(int(startX), int(startY), int(endX), int(endY))
			boxes = append(boxes, box)
			confidences = append(confidences, score)
		}
	}
	return boxes, confidences
}
