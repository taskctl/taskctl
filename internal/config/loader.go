package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/imdario/mergo"
	"github.com/mitchellh/mapstructure"
	"github.com/pelletier/go-toml"
	log "github.com/sirupsen/logrus"
	"github.com/trntv/wilson/pkg/util"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var ErrConfigNotFound = errors.New("config file not found")

type ConfigLoader struct {
	Values  map[string]string
	imports map[string]bool
	dir     string
	homeDir string
}

func NewConfigLoader() ConfigLoader {
	h, err := os.UserHomeDir()
	if err != nil {
		log.Warning(err)
	}

	dir, err := os.Getwd()
	if err != nil {
		log.Warning(err)
	}

	return ConfigLoader{
		Values:  make(map[string]string),
		imports: make(map[string]bool),
		homeDir: h,
		dir:     dir,
	}
}

func (cl *ConfigLoader) Set(key string, value string) {
	cl.Values[key] = value
}

func (cl *ConfigLoader) Load(file string) (*Config, error) {
	var err error
	cfg, err = cl.LoadGlobalConfig()
	if err != nil {
		return nil, err
	}

	if !util.FileExists(path.Join(cl.dir, file)) {
		return cfg, ErrConfigNotFound
	}

	file = path.Join(cl.dir, file)
	localCfg, err := cl.loadFile(file)
	if err != nil {
		return nil, err
	}

	lcfg, err := cl.decode(localCfg)
	if err != nil {
		return nil, err
	}

	err = cfg.merge(lcfg)
	if err != nil {
		return nil, err
	}
	cfg.init()

	log.Debugf("config %s loaded", file)
	return cfg, nil
}

func (cl *ConfigLoader) LoadGlobalConfig() (*Config, error) {
	if cl.homeDir == "" {
		return &Config{}, nil
	}

	file := path.Join(cl.homeDir, ".wilson", "config.yaml")
	if !util.FileExists(file) {
		return &Config{}, nil
	}

	cfg, err := cl.loadFile(file)
	if err != nil {
		return &Config{}, err
	}

	return cl.decode(cfg)
}

func (cl *ConfigLoader) loadFile(file string) (map[string]interface{}, error) {
	cl.imports[file] = true
	config, err := cl.readFile(file)
	if err != nil {
		return nil, err
	}

	var cm = make(map[string]interface{})
	importDir := path.Dir(file)
	if imports, ok := config["import"]; ok {
		for _, v := range imports.([]interface{}) {
			if util.IsUrl(v.(string)) {
				cm, err = cl.loadImportUrl(v.(string))
			} else {
				cm, err = cl.loadImportPath(v.(string), importDir)
			}
			if err != nil {
				return nil, fmt.Errorf("load import error: %v", err)
			}

			err = mergo.Merge(&config, cm)
			if err != nil {
				return nil, err
			}
		}
	}

	return config, nil
}

func (cl *ConfigLoader) loadImportPath(file string, dir string) (map[string]interface{}, error) {
	importFile := path.Join(dir, file)

	fi, err := os.Stat(importFile)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", importFile, err)
	}

	q := make([]string, 1)
	if !fi.IsDir() {
		q[0] = importFile
	} else {
		pattern := filepath.Join(importFile, "*.yaml")
		q, err = filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("%s: %v", importFile, err)
		}
	}

	cm := make(map[string]interface{})
	for _, importFile := range q {
		if cl.imports[importFile] == true {
			continue
		}

		cml, err := cl.loadFile(importFile)
		if err != nil {
			return nil, fmt.Errorf("%s: %v", importFile, err)
		}

		err = mergo.Merge(&cm, cml, mergo.WithOverride, mergo.WithAppendSlice, mergo.WithTypeCheck)
		if err != nil {
			return nil, fmt.Errorf("%s: %v", importFile, err)
		}
	}

	return cm, nil
}

func (cl *ConfigLoader) loadImportUrl(u string) (map[string]interface{}, error) {
	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", u, err)
	}

	ext := filepath.Ext(u)
	return cl.unmarshallData(data, ext)
}

func (cl *ConfigLoader) readFile(filename string) (map[string]interface{}, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", filename, err)
	}

	ext := filepath.Ext(filename)

	return cl.unmarshallData(data, ext)
}

func (cl *ConfigLoader) unmarshallData(data []byte, ext string) (map[string]interface{}, error) {
	var cm = make(map[string]interface{})

	switch strings.ToLower(ext) {
	case ".yaml", ".yml":
		err := yaml.NewDecoder(bytes.NewReader(data)).Decode(cm)
		if err != nil {
			return nil, err
		}
	case ".json":
		err := json.NewDecoder(bytes.NewReader(data)).Decode(cm)
		if err != nil {
			return nil, err
		}
	case ".toml":
		err := toml.NewDecoder(bytes.NewReader(data)).Decode(cm)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("unsupported config file type")
	}

	return cm, nil
}

func (cl *ConfigLoader) applyValues(cm map[string]interface{}) (err error) {
	for k, v := range cl.Values {
		err = util.SetByPath(cm, k, v)
	}

	return err
}

func (cl *ConfigLoader) decode(cm map[string]interface{}) (*Config, error) {
	err := cl.applyValues(cm)
	if err != nil {
		return nil, err
	}

	c := &Config{}
	md, _ := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
		),
		ErrorUnused:      true,
		WeaklyTypedInput: true,
		Result:           c,
		TagName:          "",
	})

	err = md.Decode(cm)
	if err != nil {
		return nil, err
	}

	return c, nil
}
