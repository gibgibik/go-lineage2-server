package internal

import (
	"bytes"
	"context"
	json2 "encoding/json"
	"errors"
	"fmt"
	"github.com/gibgibik/go-lineage2-server/internal/config"
	"github.com/gibgibik/go-lineage2-server/internal/core"
	"github.com/gibgibik/go-lineage2-server/internal/macros"
	"golang.org/x/sync/errgroup"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	ocrCl    *ocrClient
	PidsMap  map[uint32]uintptr
	writeMut sync.Mutex
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
	http.HandleFunc("/getCurrentTarget", getCurrentTarget)
	http.HandleFunc("/findBoundsTest", findBoundsHandlerTest)
	http.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		CurrentImg.Lock()
		defer CurrentImg.Unlock()
		if len(CurrentImg.ImageJpeg) == 0 {
			fmt.Println("image not found")
			return
		}
		cpImg := make([]byte, len(CurrentImg.ImageJpeg))
		copy(cpImg, CurrentImg.ImageJpeg)
		resp, _ := core.HttpCl.Client.Post("http://127.0.0.1:2224/test", "application/json", bytes.NewBuffer(cpImg))
		defer resp.Body.Close()
		res, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("read error", err)
			return
		}
		fmt.Println(len(res))
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write(res)
	})
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

func getCurrentTarget(writer http.ResponseWriter, request *http.Request) {
	var parsed struct {
		Name string
	}
	name, err := ocrCl.findTargetName()
	if err != nil {
		return
	}
	err = json2.Unmarshal(name, &parsed)
	if err != nil {
		return
	}
	parsed.Name = strings.Trim(parsed.Name, "\n")
	buf, err := json2.Marshal(parsed)
	if err != nil {
		fmt.Println(err)
		return
	}
	writer.Write(buf)
}

func findBoundsHandler(writer http.ResponseWriter, request *http.Request) {
	ctx := context.Background()
	g, ctx := errgroup.WithContext(ctx)
	var result struct {
		Boxes [][]int `:"boxes"`
	}
	//g.Go(func() error {
	//	start := time.Now()
	//
	//	var parsed struct {
	//		Name string
	//	}
	//	name, err := ocrCl.findTargetName()
	//	if err != nil {
	//		return err
	//	}
	//	err = json2.Unmarshal(name, &parsed)
	//	if err != nil {
	//		return err
	//	}
	//	writeMut.Lock()
	//	defer writeMut.Unlock()
	//	result.TargetName = strings.Trim(parsed.Name, "\n")
	//	elapsed := time.Since(start)
	//	fmt.Printf("Execution name took %s\n", elapsed)
	//
	//	return nil
	//})
	g.Go(func() error {
		start := time.Now()
		bounds, err := ocrCl.findBounds()
		elapsed := time.Since(start)

		fmt.Printf("Execution bounds took %s\n", elapsed)

		if err != nil {
			return err
		}
		writeMut.Lock()
		defer writeMut.Unlock()
		result.Boxes = bounds.Boxes
		return nil
	})
	if err := g.Wait(); err != nil {
		createRequestError(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	res, err := json2.Marshal(result)
	if err != nil {
		createRequestError(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	writer.Write(res)
	return
}

func findBoundsHandlerTest(writer http.ResponseWriter, request *http.Request) {
	ctx := context.Background()
	g, ctx := errgroup.WithContext(ctx)
	var writeMut sync.Mutex
	var result struct {
		TargetName string  `json:"target_name" :"target_name"`
		Boxes      [][]int `:"boxes"`
	}
	g.Go(func() error {
		start := time.Now()

		var parsed struct {
			Name string
		}
		name, err := ocrCl.findTargetName()
		if err != nil {
			return err
		}
		err = json2.Unmarshal(name, &parsed)
		if err != nil {
			return err
		}
		writeMut.Lock()
		defer writeMut.Unlock()
		result.TargetName = strings.Trim(parsed.Name, "\n")
		elapsed := time.Since(start)
		fmt.Printf("Execution name took %s\n", elapsed)

		return nil
	})
	g.Go(func() error {
		start := time.Now()
		bounds, err := ocrCl.findBounds()
		elapsed := time.Since(start)

		fmt.Printf("Execution bounds took %s\n", elapsed)

		if err != nil {
			return err
		}
		writeMut.Lock()
		defer writeMut.Unlock()
		result.Boxes = bounds.Boxes
		return nil
	})
	if err := g.Wait(); err != nil {
		createRequestError(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	res, err := json2.Marshal(result)
	if err != nil {
		createRequestError(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	writer.Write(res)
	return
}

func createRequestError(w http.ResponseWriter, err string, code int) {
	w.WriteHeader(code)
	_, _ = w.Write([]byte(err))
}
