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
type Config struct {
	Web           Web    `mapstructure:"web"`
	MacrosBaseUrl string `mapstructure:"macros_base_url"`
	CudaBaseUrl   string `mapstructure:"cuda_base_url"`
}

func InitConfig() (*Config, error) {
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

	return config, nil
}
