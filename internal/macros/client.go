package macros

import (
	"github.com/gibgibik/go-lineage2-server/internal/core"
	"net"
	"net/http"
	"sync"
	"time"
)

type DefaultStat struct {
	Percent    float64
	LastUpdate int64
}
type StatStr struct {
	CP     DefaultStat
	HP     DefaultStat
	MP     DefaultStat
	Target struct {
		HpPercent  float64
		LastUpdate int64
	}
	Party map[uint8]struct {
		HP DefaultStat
	}
}

var (
	HttpCl *core.HttpClient
	Stat   struct {
		sync.Mutex
		StatStr
	}
)

func IniHttpClient(baseUrl string) {
	HttpCl = &core.HttpClient{
		BaseUrl: baseUrl,
		Client: &http.Client{
			Timeout: time.Second,
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout:   time.Second,
					KeepAlive: 30 * time.Second, // Persistent connections
				}).DialContext,
				MaxIdleConns:        5,                // Total idle connections
				IdleConnTimeout:     90 * time.Second, // Keep idle connection alive
				TLSHandshakeTimeout: time.Second,
			},
		},
	}
}
