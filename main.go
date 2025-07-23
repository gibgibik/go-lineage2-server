package main

import (
	"bufio"
	"bytes"
	json2 "encoding/json"
	"fmt"
	"github.com/corona10/goimagehash"
	"github.com/gibgibik/go-lineage2-server/internal"
	"github.com/gibgibik/go-lineage2-server/internal/config"
	"github.com/gibgibik/go-lineage2-server/internal/macros"
	"github.com/gibgibik/go-lineage2-server/pkg/entity"
	"image"
	"image/jpeg"
	"log"
	"math"
	"os"
	"os/exec"
	"time"
)

const (
	yellowCheck = iota
	redCheck
	blueCheck
)

type pointerStruct struct {
	colorToCheck uint8
	rect         image.Rectangle
}

var (
	statsPointers = []pointerStruct{
		{
			colorToCheck: yellowCheck,
			rect:         image.Rectangle{image.Point{33, 49}, image.Point{238, 49}},
		},
		{
			colorToCheck: redCheck,
			rect:         image.Rectangle{image.Point{33, 66}, image.Point{238, 66}},
		},
		{
			colorToCheck: blueCheck,
			rect:         image.Rectangle{image.Point{33, 84}, image.Point{238, 84}},
		},
	}
	partyStatsHpPointers = []pointerStruct{
		{colorToCheck: redCheck, rect: image.Rectangle{image.Point{27, 148}, image.Point{207, 148}}},
		{colorToCheck: redCheck, rect: image.Rectangle{image.Point{27, 202}, image.Point{207, 202}}},
		{colorToCheck: redCheck, rect: image.Rectangle{image.Point{27, 256}, image.Point{207, 256}}},
		{colorToCheck: redCheck, rect: image.Rectangle{image.Point{27, 310}, image.Point{207, 310}}},
		{colorToCheck: redCheck, rect: image.Rectangle{image.Point{27, 364}, image.Point{207, 364}}},
		{colorToCheck: redCheck, rect: image.Rectangle{image.Point{27, 418}, image.Point{207, 418}}},
		{colorToCheck: redCheck, rect: image.Rectangle{image.Point{27, 472}, image.Point{207, 472}}},
		{colorToCheck: redCheck, rect: image.Rectangle{image.Point{27, 526}, image.Point{207, 526}}},
	}
	newTargetDelta = uint8(20)
	partySize      uint8
)

// 6, 146, 25, 151
func main() {
	internal.InitWinApi(mainRun)
}

func mainRun(hwnd uintptr) {
	cnf, err := config.InitConfig()
	if err != nil {
		panic(err)
	}
	internal.StartHttpServer(cnf)

	cmd := exec.Command("ffmpeg",
		"-f", "gdigrab", // screen capture
		"-framerate", "10", // 1 кадр/сек (зменши для тесту)
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
	maskF, _ := os.Open("party_pointer_mask.jpeg")
	defer maskF.Close()
	mask, err := jpeg.Decode(maskF)
	if err != nil {
		panic(err)
	}
	hash1, _ := goimagehash.AverageHash(mask)
	for i := 0; i < 3; i++ {
		internal.GetLu4Pids()
		if len(internal.PidsMap) > 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	fmt.Println(internal.PidsMap)
	for {
		start := time.Now()
		frame, err := readNextJPEGFrame(reader)
		internal.CurrentImg.Lock()
		internal.CurrentImg.ImageJpeg = frame
		internal.CurrentImg.Unlock()
		if err != nil {
			fmt.Println("Read frame error:", err)
			break
		}
		imgJpeg, err := jpeg.Decode(bytes.NewReader(frame))
		if err != nil {
			panic(err)
		}

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
		if math.IsNaN(percent) {
			percent = -1
		}
		currentPid := internal.ResolveCurrentPid()
		lastUpdate := time.Now().UnixMilli()
		if currentPid > 0 {
			var playerStat entity.PlayerStat
			playerStat.Target = struct {
				HpPercent  float64
				LastUpdate int64
			}{HpPercent: round(percent, 2), LastUpdate: lastUpdate}

			for _, ss := range statsPointers {
				percent = calculatePercent(imgJpeg, ss.rect, ss.colorToCheck)
				if math.IsNaN(percent) {
					percent = -1
				}
				switch ss.colorToCheck {
				case yellowCheck:
					if percent > 0 {
						playerStat.CP = entity.DefaultStat{Percent: percent, LastUpdate: lastUpdate}
					}
				case redCheck:
					if percent > 0 {
						playerStat.HP = entity.DefaultStat{Percent: percent, LastUpdate: lastUpdate}
					}
				case blueCheck:
					if percent > 0 {
						playerStat.MP = entity.DefaultStat{Percent: percent, LastUpdate: lastUpdate}
					}
				}
			}
			macros.Stat.Player[currentPid] = playerStat
		}
		party := make(map[uint8]struct {
			HP entity.DefaultStat
		})
		for idx, ss := range partyStatsHpPointers {
			partyMemberOffset := idx * 54
			compareMask := imgJpeg.(interface {
				SubImage(r image.Rectangle) image.Image
			}).SubImage(image.Rect(6, 146+partyMemberOffset, 25, 151+partyMemberOffset))
			hash2, _ := goimagehash.AverageHash(compareMask)
			distance, _ := hash2.Distance(hash1)
			if distance > 5 {
				continue
			} else {
				percent = calculatePercent(imgJpeg, ss.rect, ss.colorToCheck)
			}
			party[uint8(idx)] = struct{ HP entity.DefaultStat }{HP: entity.DefaultStat{Percent: percent, LastUpdate: lastUpdate}}

		}
		macros.Stat.Party = party
		buf, err := json2.Marshal(macros.Stat)
		if err != nil {
			fmt.Println(macros.Stat)
			panic(err)
		}
		elapsed := time.Since(start)
		fmt.Printf("Execution took %s\n", elapsed)
		_, err = macros.HttpCl.Client.Post(macros.HttpCl.BaseUrl+"api/stats", "application/json", bytes.NewBuffer(buf))
		if err != nil {
			fmt.Println("Write stats error:", err)
		}
		continue
	}
}

func calculatePercent(imgJpeg image.Image, rect image.Rectangle, colorToCheck uint8) float64 {
	var matchCount float64
	for x := rect.Min.X; x < rect.Max.X; x++ {
		r, g, b, _ := imgJpeg.At(x, rect.Min.Y).RGBA()

		r8 := uint8(r >> 8)
		g8 := uint8(g >> 8)
		b8 := uint8(b >> 8)
		match := false
		switch colorToCheck {
		case yellowCheck:
			if isYellow(r8, g8, b8, newTargetDelta) {
				match = true
			}
		case redCheck:
			if isRed(r8, g8, b8, newTargetDelta) {
				match = true
			}
		case blueCheck:
			if isBlue(r8, g8, b8, newTargetDelta) {
				match = true
			}
		}
		if match {
			matchCount++
		}
	}
	return round(matchCount/float64(rect.Max.X-rect.Min.X)*100, 2)
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
