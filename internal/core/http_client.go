package core

import (
	"bytes"
	"encoding/json"
	"github.com/gibgibik/go-lineage2-server/pkg/entity"
	"io"
	"mime/multipart"
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

func (cl *HttpClient) FindBounds(config entity.GetBoundsConfig, body []byte) ([]byte, error) {
	buf := &bytes.Buffer{}
	writer := multipart.NewWriter(buf)
	part, _ := writer.CreateFormFile("file", "img.png")
	_, _ = io.Copy(part, bytes.NewBuffer(body))
	meta, _ := json.Marshal(config)
	_ = writer.WriteField("meta", string(meta))
	_ = writer.Close()
	req, err := http.NewRequest("POST", cl.BaseUrl+"findBounds", buf)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	client := &http.Client{}
	resp, err := client.Do(req)
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
