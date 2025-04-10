package app

import (
	"flag"
	"fmt"
	"os"
	"tgstreamer/config"
	"tgstreamer/lib/pg"

	"gopkg.in/yaml.v3"
)

type Config struct {
	VideosDir     string    `yaml:"videos_dir"`
	YoutubeApiKey string    `yaml:"youtube_api_key"`
	Pg            pg.Config `yaml:"pg"`
}

func ReadConfig() (conf Config) {
	err := yaml.Unmarshal(config.Reference, &conf)
	if err != nil {
		panic(fmt.Errorf("failed to unmarshal reference.yaml: %w", err))
	}
	configFile := flag.String("config", "", "configuration file")
	flag.Parse()
	if *configFile != "" {
		config, err := os.ReadFile(*configFile)
		if err != nil {
			panic(fmt.Errorf("read config file %q: %w", *configFile, err))
		}
		err = yaml.Unmarshal(config, &conf)
		if err != nil {
			panic(fmt.Errorf("failed to unmarshal config file %q: %w", *configFile, err))
		}
	}
	return conf
}
