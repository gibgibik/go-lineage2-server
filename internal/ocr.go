package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gibgibik/go-lineage2-server/internal/core"
	"github.com/gibgibik/go-lineage2-server/pkg/entity"
	"image"
	"sync"
)

var (
	CurrentImg struct {
		sync.Mutex
		ImageJpeg []byte
	}
)

type ocrClient struct {
}

func newOcrClient() *ocrClient {
	return &ocrClient{}
}

func (cl *ocrClient) findBounds() (*core.BoxesStruct, error) {
	CurrentImg.Lock()
	if len(CurrentImg.ImageJpeg) == 0 {
		return nil, errors.New("image not found")
	}
	cpImg := make([]byte, len(CurrentImg.ImageJpeg))
	copy(cpImg, CurrentImg.ImageJpeg)
	CurrentImg.Unlock()

	res, err := core.HttpCl.FindBounds(entity.GetBoundsConfig{
		ExcludeBounds: []image.Rectangle{
			image.Rect(0, 0, 247, 110),         // ex player stat
			image.Rect(0, 590, 370, 1074),      // chat
			image.Rect(697, 915, 1273, 1074),   // panel with skills
			image.Rect(1710, -50, 1920, 233),   // map
			image.Rect(1644, 0, 1748, 35),      // money
			image.Rect(775, 390, 1235, 811),    // me
			image.Rect(273, 6, 561, 52),        // buffs
			image.Rect(1849, 1061, 1888, 1076), // time
			image.Rect(787, 2, 1135, 29),       // target name
		},
		NpcThreshold: 0.9995,
		NpcNms:       0.4,
	}, cpImg)
	boxes := &core.BoxesStruct{
		Boxes: make([][]int, 0),
	}
	if err := json.Unmarshal(res, &boxes); err != nil {
		fmt.Println("JSON decode error:", err)
		return nil, err
	}
	if err != nil {
		return boxes, err
	}
	ClearOverlay(Hwnd)
	fmt.Println(boxes)
	for _, v := range boxes.Boxes {
		Draw(Hwnd, uintptr(v[0]), uintptr(v[1]), uintptr(v[2]), uintptr(v[3]), "")
	}
	return boxes, err
}

func (cl *ocrClient) findTargetName() ([]byte, error) {
	//start := time.Now()
	//img, err := screenshot.CaptureDisplay(0)
	//elapsed := time.Since(start)
	//fmt.Printf("Execution name took %s\n", elapsed)

	CurrentImg.Lock()
	if len(CurrentImg.ImageJpeg) == 0 {
		return nil, errors.New("image not found")
	}
	cpImg := make([]byte, len(CurrentImg.ImageJpeg))
	copy(cpImg, CurrentImg.ImageJpeg)
	CurrentImg.Unlock()
	//var imgB bytes.Buffer
	//jpeg.Encode(&imgB, cpImg, &jpeg.Options{Quality: 100})
	//if err != nil {
	//	return nil, err
	//}

	return core.HttpCl.Post("findTargetName", cpImg)
}
