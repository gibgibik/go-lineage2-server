package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

var HttpCl *HttpClient

type HttpClient struct {
	Client  *http.Client
	baseUrl string
}

type BoxesStruct struct {
	Boxes [][]int `json:"boxes"`
}

func (cl *HttpClient) Post(path string, body []byte) (*BoxesStruct, error) {
	const maxRetries = 10
	//var resp *http.Response
	for attempt := 1; attempt <= maxRetries; attempt++ {
		fmt.Println("start", time.Now().UTC())
		resp, err := cl.Client.Post(cl.baseUrl+path, "application/json", bytes.NewBuffer(body))
		fmt.Println("end", time.Now().UTC())
		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			res, err := io.ReadAll(resp.Body)
			if err != nil {
				fmt.Println("read error", err)
			} else {
				//fmt.Println(string(res))
			}
			result := &BoxesStruct{
				Boxes: make([][]int, 0),
			}
			if err := json.Unmarshal(res, &result); err != nil {
				fmt.Println("JSON decode error:", err)
				return nil, err
			}
			return result, nil
		}

		// Log error and retry
		if err != nil {
			fmt.Printf("Attempt %d: Request failed: %v\n", attempt, err)
		} else {
			fmt.Printf("Attempt %d: Unexpected status: %s\n", attempt, resp.Status)
			resp.Body.Close()
		}

		time.Sleep(time.Second / 2)
	}
	return nil, errors.New("no data")
}

func IniHttpClient(baseUrl string) {
	HttpCl = &HttpClient{
		baseUrl: baseUrl,
		Client: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout:   5 * time.Second,
					KeepAlive: 30 * time.Second, // Persistent connections
				}).DialContext,
				MaxIdleConns:        10,               // Total idle connections
				IdleConnTimeout:     90 * time.Second, // Keep idle connection alive
				TLSHandshakeTimeout: 5 * time.Second,
			},
		},
	}
}
