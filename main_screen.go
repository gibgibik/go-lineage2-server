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
	"time"
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

func createRequestError(w http.ResponseWriter, err string, code int) {
	w.WriteHeader(code)
	_, _ = w.Write([]byte(err))
}

func main() {
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
	client := gosseract.NewClient()
	client.SetVariable("tessedit_char_whitelist", "0123456789/% ")
	defer client.Close()
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
		client.SetImageFromBytes(buf.Bytes())
		text, err := client.Text()
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
		statLock.Unlock()
	}
	//}
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
