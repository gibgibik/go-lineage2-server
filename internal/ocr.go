package internal

import (
	"errors"
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

	boxes, err := core.HttpCl.Post("/findBounds", cpImg)
	if err != nil {
		return boxes, err
	}
	//ClearOverlay(Hwnd)
	//for _, v := range boxes.Boxes {
	//	Draw(Hwnd, uintptr(v[0]), uintptr(v[1]), uintptr(v[2]), uintptr(v[3]), "")
	//}
	return boxes, err
}
