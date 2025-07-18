package core

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"time"
)

var HttpCl *HttpClient

type HttpClient struct {
	Client  *http.Client
	baseUrl string
}

func (cl *HttpClient) Post(path string, body []byte) {
	const maxRetries = 10
	//var resp *http.Response
	for attempt := 1; attempt <= maxRetries; attempt++ {
		fmt.Println("start")
		resp, err := cl.Client.Post(cl.baseUrl+path, "application/json", bytes.NewBuffer(body))
		fmt.Println("end")
		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			//res, err := io.ReadAll(resp.Body)
			//if err != nil {
			//	return nil, err
			//}
			//var playerStat *entity.PlayerStat
			//if err := json.Unmarshal(res, &playerStat); err != nil {
			//	fmt.Println("JSON decode error:", err)
			//	return nil, err
			//}
			//return playerStat, nil
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
