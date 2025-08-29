package config

import (
	"github.com/spf13/viper"
)

type Web struct {
	Port string
}
type Control struct {
	Port       string
	BaudRate   int   `mapstructure:"baud_rate"`
	Resolution []int `mapstructure:"resolution"`
}
type Client struct {
	ExcludeBounds [][]int `mapstructure:"exclude_bounds"`
	NpcThreshold  float32 `mapstructure:"npc_threshold"`
	NpcNmc        float32 `mapstructure:"npc_nmc"`
	TargetRect    []int   `mapstructure:"target_rect"`
	PlayerRects   [][]int `mapstructure:"player_rects"`
}

type Config struct {
	Web           Web    `mapstructure:"web"`
	MacrosBaseUrl string `mapstructure:"macros_base_url"`
	CudaBaseUrl   string `mapstructure:"cuda_base_url"`
	ClientConfig  Client
}

func InitConfig(cnfName string) (*Config, error) {
	viper.SetConfigFile("configs/main.yaml")
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}
	v := viper.New()
	v.SetConfigFile("configs/main.env.yaml")
	if v.ReadInConfig() == nil {
		err := viper.MergeConfigMap(v.AllSettings())
		if err != nil {
			return nil, err
		}
	}
	config := &Config{}
	if err := viper.Unmarshal(config); err != nil {
		return nil, err
	}

	var cnfS Client
	_ = viper.Sub("client_config." + cnfName).Unmarshal(&cnfS)
	config.ClientConfig = cnfS

	return config, nil
}
