package internal

import (
	json2 "encoding/json"
	"errors"
	"fmt"
	"github.com/LA/internal/core"
	"log"
	"net/http"
	"sync"
	"time"
)

type StatStr struct {
	CP struct {
		Percent    float64
		LastUpdate int64
	}
	HP struct {
		Percent    float64
		LastUpdate int64
	}
	MP struct {
		Percent    float64
		LastUpdate int64
	}
	Target struct {
		HpPercent  float64
		LastUpdate int64
	}
}

type BoxesStruct struct {
	Boxes [][]int `json:"boxes"`
}

var (
	Stat     StatStr
	StatLock sync.RWMutex
	ocrCl    *ocrClient
	pidsMap  map[uint32]uintptr
)

func StartHttpServer() {
	ocrCl = newOcrClient()
	handle := &http.Server{
		Addr:         ":2223",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	core.IniHttpClient("http://127.0.0.1:/2224") //@todo config
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		StatLock.RLock()
		json, err := json2.Marshal(Stat)
		if err != nil {
			createRequestError(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		writer.Write(json)
		defer StatLock.RUnlock()
	})
	http.HandleFunc("/findBounds", findBoundsHandler)
	http.HandleFunc("/init", func(writer http.ResponseWriter, request *http.Request) {
		fmt.Println(GetLu4Pids())
		result := struct {
			PidsData map[uint32]string
		}{
			PidsData: GetLu4Pids(),
		}
		js, _ := json2.Marshal(result)
		_, _ = writer.Write(js)
	})
	http.HandleFunc("/getForegroundWindowPid", func(writer http.ResponseWriter, request *http.Request) {
		currentWindowPid := getForegroundWindowHwnd()
		var body struct {
			Pid uint32 `json:"pid"`
		}
		for pid, hwnd := range pidsMap {
			if hwnd == currentWindowPid {
				body.Pid = pid
			}
		}
		buf, _ := json2.Marshal(body)
		writer.Write(buf)
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
}

func findBoundsHandler(writer http.ResponseWriter, request *http.Request) {
	boxes, err := ocrCl.findBounds()
	if err != nil {
		createRequestError(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	b, _ := json2.Marshal(boxes)
	writer.Write(b)
}

func createRequestError(w http.ResponseWriter, err string, code int) {
	w.WriteHeader(code)
	_, _ = w.Write([]byte(err))
}
