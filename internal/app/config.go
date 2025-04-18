package app

import (
	"flag"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"tgstreamer/config"
	"tgstreamer/lib/pg"
	"time"

	"github.com/mitchellh/mapstructure"
	_ "github.com/mitchellh/mapstructure"
	"gopkg.in/yaml.v3"
)

type Config struct {
	VideosDir     string    `yaml:"videos_dir"`
	YoutubeApiKey string    `yaml:"youtube_api_key"`
	Pg            pg.Config `yaml:"pg"`
	TgBotToken    string    `yaml:"tg_bot_token"`
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
	m := map[string]any{}
	err = mapstructure.Decode(conf, &m)
	if err != nil {
		panic(err)
	}
	env(nil, m, &conf)
	return conf
}

func env(prefix []string, x any, conf any) {
	fmt.Println(prefix, reflect.TypeOf(x))
	switch m := x.(type) {
	case map[string]interface{}:
		for key, val := range m {
			env(append(prefix, key), val, conf)
		}
	case bool:
		storeBool(prefix, m, conf)
	case string:
		storeString(prefix, m, conf)
	case float64:
		storeNumber(prefix, m, conf)
	case int:
		storeNumber(prefix, float64(m), conf)
	case []any:
		storeSlice(prefix, m, conf)
	}
}

func findValue(prefix []string, config any) reflect.Value {
	v := reflect.ValueOf(config).Elem()
	for i := 0; i < len(prefix); i++ {
		if v.Kind() == reflect.Map {
			if v.IsNil() {
				v.Set(reflect.MakeMap(v.Type()))
			}
			found := false
			for _, mk := range v.MapKeys() {
				if mk.String() == prefix[i] {
					v = v.MapIndex(mk).Elem()
					found = true
					break
				}
			}
			if !found {
				var typ reflect.Type
				if v.Type().Elem().Kind() == reflect.Pointer {
					typ = v.Type().Elem().Elem()
				} else {
					typ = v.Type().Elem()
				}
				v.SetMapIndex(reflect.ValueOf(prefix[i]), reflect.New(typ))
				for _, mk := range v.MapKeys() {
					if mk.String() == prefix[i] {
						v = v.MapIndex(mk).Elem()
						break
					}
				}
			}
		} else {
			found := false
			for j := 0; j < v.NumField(); j++ {
				if strings.EqualFold(v.Type().Field(j).Name, strings.ReplaceAll(prefix[i], "_", "")) {
					v = v.Field(j)
					found = true
					break
				}
			}
			if !found {
				panic(strings.Join(prefix, ".") + " not found in config struct")
			}
		}
	}
	return v
}

func storeSlice(prefix []string, val []any, config any) {
	v := findValue(prefix, config)
	if !v.IsValid() {
		return
	}
	if len(val) == 0 {
		return
	}
	var confVal reflect.Value
	switch val[0].(type) {
	case int:
		if v.Type().String() == "[]int64" {
			s := make([]int64, len(val))
			for i := range val {
				s[i] = int64(val[i].(int))
			}
			confVal = reflect.ValueOf(s)
		}
	case string:
		s := make([]string, len(val))
		for i := range val {
			s[i] = val[i].(string)
		}
		confVal = reflect.ValueOf(s)
	}
	v.Set(confVal)
}

func storeBool(prefix []string, val bool, config any) {
	v := findValue(prefix, config)
	if !v.IsValid() {
		return
	}
	v.SetBool(val)
	env, ok := os.LookupEnv(strings.Join(prefix, "_"))
	if !ok {
		return
	}
	val, err := strconv.ParseBool(env)
	if err != nil {
		panic(err)
	}
	v.SetBool(val)
}

func storeString(prefix []string, val string, config any) {
	v := findValue(prefix, config)
	if !v.IsValid() {
		return
	}
	envName := strings.Join(prefix, "_")
	fmt.Println("lookup env:", strings.ToUpper(envName))
	env, ok := os.LookupEnv(strings.ToUpper(envName))
	if ok {
		val = env
	}
	if v.Type().String() == "time.Duration" {
		d, err := time.ParseDuration(val)
		if err != nil {
			panic(err)
		}
		v.SetInt(int64(d))
	} else {
		v.SetString(val)
	}
}

func storeNumber(prefix []string, val float64, config any) {
	v := findValue(prefix, config)
	if !v.IsValid() {
		return
	}
	env, ok := os.LookupEnv(strings.Join(prefix, "_"))
	if ok {
		f, err := strconv.ParseFloat(env, 64)
		if err != nil {
			panic(err)
		}
		val = f
	}
	switch v.Kind() {
	case reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8:
		v.SetInt(int64(val))
	case reflect.Uint, reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.Uint8:
		v.SetUint(uint64(val))
	case reflect.Float32, reflect.Float64:
		v.SetFloat(val)
	}
}
