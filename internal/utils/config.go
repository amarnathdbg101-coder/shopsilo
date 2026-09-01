package utils

import (
	"fmt"
	"log"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

var k = koanf.New(".")

type Config struct {
	Port        int    `koanf:"port"`
	Env         string `koanf:"env"`
	DbUrl string `koanf:"dburl"`
	JwtSecret   string `koanf:"jwt_secret"`
}

func MustLoad() Config {
	if err := k.Load(file.Provider("config.yaml"), yaml.Parser()); err != nil {
		log.Fatalf("error loading  config :%v", err)
	}

	   // Load from environment variables (prefix APP_)
    if err := k.Load(env.Provider("", ".", func(s string) string {
        return strings.Replace(strings.ToLower(s), "_", ".", -1)
    }), nil); err != nil {
        log.Fatalf("error loading env vars: %v", err)
    }

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		log.Fatalf("error unmarshalling config: %v", err)
	}

	fmt.Printf("loaded database url:%v",cfg.DbUrl)
	return cfg
}
