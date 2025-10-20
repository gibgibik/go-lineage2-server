package internal

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gibgibik/go-lineage2-server/internal/config"
	"github.com/gibgibik/go-lineage2-server/internal/core"
	"github.com/gibgibik/go-lineage2-server/pkg/entity"
	"image"
	"image/jpeg"
	"sync"
)

var (
	CurrentImg struct {
		sync.Mutex
		ImageJpeg []byte
	}
)

type ocrClient struct {
	excludeBounds []image.Rectangle
	cnf           config.Client
}

func newOcrClient(cnf config.Client) *ocrClient {
	var excludeBounds = make([]image.Rectangle, 0)
	for _, v := range cnf.ExcludeBounds {
		excludeBounds = append(excludeBounds, image.Rectangle{image.Point{v[0], v[1]}, image.Point{v[2], v[3]}})
	}
	return &ocrClient{excludeBounds: excludeBounds, cnf: cnf}
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
		ExcludeBounds: cl.excludeBounds,
		NpcThreshold:  cl.cnf.NpcThreshold,
		NpcNms:        cl.cnf.NpcNmc,
		Resolution:    cl.cnf.Resolution,
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
	imgJpeg, _ := jpeg.Decode(bytes.NewReader(cpImg))
	subImg := imgJpeg.(interface {
		SubImage(r image.Rectangle) image.Image
	}).SubImage(image.Rectangle{
		Min: image.Point{
			X: config.Cnf.ClientConfig.TargetNameRect[0],
			Y: config.Cnf.ClientConfig.TargetNameRect[1],
		},
		Max: image.Point{
			X: config.Cnf.ClientConfig.TargetNameRect[2],
			Y: config.Cnf.ClientConfig.TargetNameRect[3],
		},
	})
	var imgB bytes.Buffer
	jpeg.Encode(&imgB, subImg, &jpeg.Options{Quality: 100})
	//if err != nil {
	//	return nil, err
	//}

	return core.HttpCl.Post("findTargetName", imgB.Bytes())
}
