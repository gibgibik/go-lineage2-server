package internal

import (
	json2 "encoding/json"
	"errors"
	"github.com/gibgibik/go-lineage2-server/internal/config"
	"github.com/gibgibik/go-lineage2-server/internal/core"
	"github.com/gibgibik/go-lineage2-server/internal/macros"
	"log"
	"net/http"
	"time"
)

var (
	ocrCl   *ocrClient
	PidsMap map[uint32]uintptr
)

func StartHttpServer(cnf *config.Config) {
	ocrCl = newOcrClient()
	handle := &http.Server{
		Addr:         cnf.Web.Port,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	core.IniHttpClient(cnf.CudaBaseUrl)
	macros.IniHttpClient(cnf.MacrosBaseUrl)
	//http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
	//	StatLock.RLock()
	//	json, err := json2.Marshal(Stat)
	//	if err != nil {
	//		createRequestError(writer, err.Error(), http.StatusInternalServerError)
	//		return
	//	}
	//	writer.Write(json)
	//	defer StatLock.RUnlock()
	//})
	http.HandleFunc("/findBounds", findBoundsHandler)
	http.HandleFunc("/init", func(writer http.ResponseWriter, request *http.Request) {
		result := struct {
			PidsData map[uint32]string
		}{
			PidsData: GetLu4Pids(),
		}
		js, _ := json2.Marshal(result)
		_, _ = writer.Write(js)
	})
	http.HandleFunc("/getForegroundWindowPid", func(writer http.ResponseWriter, request *http.Request) {
		var body struct {
			Pid uint32 `json:"pid"`
		}
		for i := 0; i < 3; i++ {
			pid := ResolveCurrentPid()
			if pid > 0 {
				body.Pid = pid
				break
			}
			time.Sleep(time.Millisecond * 50)
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

func ResolveCurrentPid() uint32 {
	currentWindowPid := getForegroundWindowHwnd()
	for pid, hwnd := range PidsMap {
		if hwnd == currentWindowPid {
			return pid
		}
	}
	return 0
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
