package core

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"time"
)

var HttpCl *HttpClient

type HttpClient struct {
	Client  *http.Client
	BaseUrl string
}

type BoxesStruct struct {
	Boxes [][]int `json:"boxes"`
}

func (cl *HttpClient) Post(path string, body []byte) ([]byte, error) {
	resp, err := cl.Client.Post(cl.BaseUrl+path, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	res, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func IniHttpClient(baseUrl string) {
	HttpCl = &HttpClient{
		BaseUrl: baseUrl,
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
