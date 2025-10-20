package main

import (
	"bufio"
	"bytes"
	json2 "encoding/json"
	"flag"
	"fmt"
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
	statsPointers []pointerStruct
	targetRect    image.Rectangle
	//statsPointers = []pointerStruct{
	//	{
	//		colorToCheck: yellowCheck,
	//		rect:         image.Rectangle{image.Point{33, 49}, image.Point{238, 49}},
	//	},
	//	{
	//		colorToCheck: redCheck,
	//		rect:         image.Rectangle{image.Point{33, 66}, image.Point{238, 66}},
	//	},
	//	{
	//		colorToCheck: blueCheck,
	//		rect:         image.Rectangle{image.Point{33, 84}, image.Point{238, 84}},
	//	},
	//}
	//partyStatsHpPointers = []pointerStruct{
	//	{colorToCheck: redCheck, rect: image.Rectangle{image.Point{27, 148}, image.Point{207, 148}}},
	//	{colorToCheck: redCheck, rect: image.Rectangle{image.Point{27, 202}, image.Point{207, 202}}},
	//	{colorToCheck: redCheck, rect: image.Rectangle{image.Point{27, 256}, image.Point{207, 256}}},
	//	{colorToCheck: redCheck, rect: image.Rectangle{image.Point{27, 310}, image.Point{207, 310}}},
	//	{colorToCheck: redCheck, rect: image.Rectangle{image.Point{27, 364}, image.Point{207, 364}}},
	//	{colorToCheck: redCheck, rect: image.Rectangle{image.Point{27, 418}, image.Point{207, 418}}},
	//	{colorToCheck: redCheck, rect: image.Rectangle{image.Point{27, 472}, image.Point{207, 472}}},
	//	{colorToCheck: redCheck, rect: image.Rectangle{image.Point{27, 526}, image.Point{207, 526}}},
	//}
	newTargetDelta             = uint8(30)
	fullTargetHpUnchangedSince time.Time
	configName                 string
)

func main() {
	cnfName := flag.String("config", "mw", "config name")
	flag.Parse()
	err := config.InitConfig(*cnfName)
	if err != nil {
		panic(err)
	}
	internal.InitWinApi(mainRun)
}

func mainRun(hwnd uintptr) {
	for idx, v := range config.Cnf.ClientConfig.PlayerRects {
		statsPointers = append(statsPointers, pointerStruct{
			colorToCheck: uint8(idx),
			rect:         image.Rectangle{image.Point{v[0], v[1]}, image.Point{v[2], v[3]}},
		})
	}
	targetRect = image.Rectangle{image.Point{config.Cnf.ClientConfig.TargetRect[0], config.Cnf.ClientConfig.TargetRect[1]}, image.Point{config.Cnf.ClientConfig.TargetRect[2], config.Cnf.ClientConfig.TargetRect[3]}}
	internal.StartHttpServer(config.Cnf)

	cmd := exec.Command("ffmpeg",
		"-f", "gdigrab",
		"-framerate", "10",
		"-i", "desktop",
		"-f", "image2pipe",
		"-vcodec", "mjpeg", // або "png"
		"-q:v", "1",
		"-s", fmt.Sprintf("%vx%v", config.Cnf.ClientConfig.Resolution[0], config.Cnf.ClientConfig.Resolution[1]),
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
	//mask, err := jpeg.Decode(maskF)
	//if err != nil {
	//	panic(err)
	//}
	//hash1, _ := goimagehash.AverageHash(mask)
	for i := 0; i < 3; i++ {
		internal.GetPids()
		if len(internal.PidsMap) > 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	//fmt.Println(internal.PidsMap)
	for {
		//start := time.Now()
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
		currentPid := internal.ResolveCurrentPid()
		lastUpdate := time.Now().UnixMilli()
		//fmt.Println(currentPid)
		//if currentPid > 0 {
		//pieces := strings.Split(text, "\n")
		percent, playerStat := handleTargetState(imgJpeg, currentPid, lastUpdate)
		handlePlayerState(percent, imgJpeg, playerStat, lastUpdate, currentPid)
		//}
		//handlePartyState(imgJpeg, hash1, lastUpdate) //@todo enable later?
		buf, err := json2.Marshal(macros.Stat)
		if err != nil {
			//fmt.Println(macros.Stat)
			panic(err)
		}
		//elapsed := time.Since(start)
		//fmt.Printf("Execution took %s\n", elapsed)
		_, err = macros.HttpCl.Client.Post(macros.HttpCl.BaseUrl+"api/stats", "application/json", bytes.NewBuffer(buf))
		if err != nil {
			fmt.Println("Write stats error:", err)
		}
		continue
	}
}

//func handlePartyState(imgJpeg image.Image, hash1 *goimagehash.ImageHash, lastUpdate int64) {
//	party := make(map[uint8]entity.PartyMember)
//	var percent float64
//	for idx, ss := range partyStatsHpPointers {
//		partyMemberOffset := idx * 54
//		compareMask := imgJpeg.(i	nterface {
//			SubImage(r image.Rectangle) image.Image
//		}).SubImage(image.Rect(6, 146+partyMemberOffset, 25, 151+partyMemberOffset))
//		hash2, _ := goimagehash.AverageHash(compareMask)
//		distance, _ := hash2.Distance(hash1)
//		if distance > 5 {
//			continue
//		} else {
//			percent = calculatePercent(imgJpeg, ss.rect, ss.colorToCheck)
//		}
//		party[uint8(idx)] = struct{ HP entity.DefaultStat }{HP: entity.DefaultStat{Percent: percent, LastUpdate: lastUpdate}}
//
//	}
//	macros.Stat.Party = party
//}

func handlePlayerState(percent float64, imgJpeg image.Image, playerStat entity.PlayerStat, lastUpdate int64, currentPid uint32) {
	for _, ss := range statsPointers {
		percent = calculatePercent(imgJpeg, ss.rect, ss.colorToCheck)
		if math.IsNaN(percent) {
			percent = -1
		}
		switch ss.colorToCheck {
		case yellowCheck:
			if percent > 0 {
				//fmt.Print("cp ", percent)
				playerStat.CP = entity.DefaultStat{Percent: percent, LastUpdate: lastUpdate}
			}
		case redCheck:
			if percent > 0 {
				//fmt.Print("hp ", percent)
				playerStat.HP = entity.DefaultStat{Percent: percent, LastUpdate: lastUpdate}
			}
		case blueCheck:
			if percent > 0 {
				//fmt.Print("mp ", percent)

				playerStat.MP = entity.DefaultStat{Percent: percent, LastUpdate: lastUpdate}
			}
		}
	}
	//fmt.Println("")
	macros.Stat.Player[currentPid] = playerStat
}

func handleTargetState(imgJpeg image.Image, currentPid uint32, lastUpdate int64) (float64, entity.PlayerStat) {
	targetDelta := uint8(20)
	targetR, targetG, targetB := uint8(108), uint8(23), uint8(13)
	//internal.ClearOverlay(internal.Hwnd)
	//internal.Draw(internal.Hwnd, uintptr(targetRect.Min.X), uintptr(targetRect.Min.Y), uintptr(targetRect.Max.X), uintptr(targetRect.Max.Y+2), "")
	var targetResultRes int
	var maxX = 0
	for x := targetRect.Max.X; x >= targetRect.Min.X; x-- {
		r, g, b, _ := imgJpeg.At(x, targetRect.Min.Y).RGBA()
		r8 := uint8(r >> 8)
		g8 := uint8(g >> 8)
		b8 := uint8(b >> 8)
		if withinDelta(r8, targetR, targetDelta) &&
			withinDelta(g8, targetG, targetDelta) &&
			withinDelta(b8, targetB, targetDelta) {
			targetResultRes = targetResultRes + 1
		}
		if targetResultRes >= 3 {
			maxX = x - targetRect.Min.X
			break
		}
		//gray := uint8((uint16(r8) + uint16(g8) + uint16(b8)) / 3)
	}
	percent := round(float64(maxX)/(float64(targetRect.Max.X-targetRect.Min.X)/float64(100)), 2)
	if math.IsNaN(percent) {
		percent = -1
	}
	playerStat := macros.Stat.Player[currentPid]
	playerStat.Target.HpPercent = percent
	playerStat.Target.LastUpdate = lastUpdate
	if percent > 0 {
		playerStat.Target.HpWasPresentAt = time.Now().UnixMilli()
	}
	if playerStat.Target.HpPercent >= 99 {
		if fullTargetHpUnchangedSince.IsZero() {
			fullTargetHpUnchangedSince = time.Now()
		}
	} else {
		fullTargetHpUnchangedSince = time.Now()
	}
	playerStat.Target.FullHpUnchangedSince = fullTargetHpUnchangedSince.UnixMilli()
	return percent, playerStat
}

func calculatePercent(imgJpeg image.Image, rect image.Rectangle, colorToCheck uint8) float64 {
	var matchCount float64
	targetDelta := uint8(20)
	var targetR, targetG, targetB uint8
	switch colorToCheck {
	case yellowCheck:
		targetR, targetG, targetB = uint8(125), uint8(90), uint8(19)
	case redCheck:
		targetR, targetG, targetB = uint8(135), uint8(30), uint8(20)
	case blueCheck:
		targetR, targetG, targetB = uint8(8), uint8(68), uint8(159)
	}
	var maxX = 0
	for x := rect.Max.X; x >= rect.Min.X; x-- {
		r, g, b, _ := imgJpeg.At(x, rect.Min.Y).RGBA()
		r8 := uint8(r >> 8)
		g8 := uint8(g >> 8)
		b8 := uint8(b >> 8)

		if withinDelta(r8, targetR, targetDelta) &&
			withinDelta(g8, targetG, targetDelta) &&
			withinDelta(b8, targetB, targetDelta) {
			matchCount++
		}
		if matchCount >= 3 {
			maxX = x - rect.Min.X
			break
		}
	}
	percent := round(float64(maxX)/(float64(rect.Max.X-rect.Min.X)/float64(100)), 2)
	if math.IsNaN(percent) {
		percent = -1
	}
	return percent
	//return round(matchCount/float64(rect.Max.X-rect.Min.X)*100, 2)
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
