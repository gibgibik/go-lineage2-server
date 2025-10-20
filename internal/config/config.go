package config

import (
	"github.com/spf13/viper"
)

type Web struct {
	Port string
}
type Client struct {
	CudaBaseUrl    string  `mapstructure:"cuda_base_url"`
	Resolution     []int   `mapstructure:"resolution"`
	ExcludeBounds  [][]int `mapstructure:"exclude_bounds"`
	NpcThreshold   float32 `mapstructure:"npc_threshold"`
	NpcNmc         float32 `mapstructure:"npc_nmc"`
	TargetRect     []int   `mapstructure:"target_rect"`
	TargetNameRect []int   `mapstructure:"target_name_rect"`
	PlayerRects    [][]int `mapstructure:"player_rects"`
	Web            Web
	MacrosBaseUrl  string `mapstructure:"macros_base_url"`
}

type Config struct {
	CudaBaseUrl  string `mapstructure:"cuda_base_url"`
	ClientConfig Client
}

var (
	Cnf *Config
)

func InitConfig(cnfName string) error {
	viper.SetConfigFile("configs/main.yaml")
	if err := viper.ReadInConfig(); err != nil {
		return err
	}
	v := viper.New()
	v.SetConfigFile("configs/main.env.yaml")
	if v.ReadInConfig() == nil {
		err := viper.MergeConfigMap(v.AllSettings())
		if err != nil {
			return err
		}
	}
	Cnf = &Config{}
	if err := viper.Unmarshal(Cnf); err != nil {
		return err
	}

	var cnfS Client
	_ = viper.Sub("client_config." + cnfName).Unmarshal(&cnfS)
	Cnf.ClientConfig = cnfS

	return nil
}
