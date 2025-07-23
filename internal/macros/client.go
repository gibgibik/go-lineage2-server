package macros

import (
	"github.com/gibgibik/go-lineage2-server/internal/core"
	"github.com/gibgibik/go-lineage2-server/pkg/entity"
	"net"
	"net/http"
	"time"
)

var (
	HttpCl *core.HttpClient
	Stat   entity.StatStr
)

func IniHttpClient(baseUrl string) {
	Stat.Player = make(map[uint32]entity.PlayerStat)
	Stat.Party = make(map[uint8]entity.PartyMember)
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
