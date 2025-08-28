package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gibgibik/go-lineage2-server/internal/core"
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

	res, err := core.HttpCl.Post("findBounds", cpImg)
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
	//ClearOverlay(Hwnd)
	//for _, v := range boxes.Boxes {
	//	Draw(Hwnd, uintptr(v[0]), uintptr(v[1]), uintptr(v[2]), uintptr(v[3]), "")
	//}
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
