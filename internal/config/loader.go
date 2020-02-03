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
	"github.com/taskctl/taskctl/pkg/util"
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

	if file == "" {
		file, err = cl.resolveDefaultConfigFile()
		if err != nil {
			return cfg, err
		}
	}

	if !util.IsUrl(file) && !filepath.IsAbs(file) {
		file = path.Join(cl.dir, file)
	}

	localCfg, err := cl.load(file)
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

	file := path.Join(cl.homeDir, ".taskctl", "config.yaml")
	if !util.FileExists(file) {
		return &Config{}, nil
	}

	cfg, err := cl.load(file)
	if err != nil {
		return &Config{}, err
	}

	return cl.decode(cfg)
}

func (cl *ConfigLoader) load(file string) (config map[string]interface{}, err error) {
	cl.imports[file] = true

	if util.IsUrl(file) {
		config, err = cl.readUrl(file)
	} else {
		if !util.FileExists(file) {
			return config, fmt.Errorf("%s: %w", file, ErrConfigNotFound)
		}
		config, err = cl.readFile(file)
	}
	if err != nil {
		return nil, err
	}

	var cm = make(map[string]interface{})
	importDir := path.Dir(file)
	if imports, ok := config["import"]; ok {
		for _, v := range imports.([]interface{}) {
			if util.IsUrl(v.(string)) {
				cm, err = cl.load(v.(string))
			} else {
				importFile := path.Join(importDir, v.(string))
				fi, err := os.Stat(importFile)
				if err != nil {
					return nil, fmt.Errorf("%s: %v", importFile, err)
				}
				if !fi.IsDir() {
					cm, err = cl.load(importFile)
				} else {
					cm, err = cl.loadDir(importFile)
				}
			}
			if err != nil {
				return nil, fmt.Errorf("load import error: %v", err)
			}

			err = mergo.Merge(&config, cm, mergo.WithOverride, mergo.WithAppendSlice, mergo.WithTypeCheck)
			if err != nil {
				return nil, err
			}
		}
	}

	return config, nil
}

func (cl *ConfigLoader) loadDir(dir string) (map[string]interface{}, error) {
	pattern := filepath.Join(dir, "*.yaml")
	q, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", dir, err)
	}

	cm := make(map[string]interface{})
	for _, importFile := range q {
		if cl.imports[importFile] == true {
			continue
		}

		cml, err := cl.load(importFile)
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

func (cl *ConfigLoader) readUrl(u string) (map[string]interface{}, error) {
	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("%d: config request failed - %s", resp.StatusCode, u)
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

func (cl *ConfigLoader) resolveDefaultConfigFile() (file string, err error) {
	files := make([]string, 0)

	dir := cl.dir
	for {
		if dir == "/" {
			break
		}

		files = append(files, filepath.Join(dir, "taskctl.yaml"), filepath.Join(dir, "tasks.yaml"))
		dir = filepath.Dir(dir)
	}

	for _, file = range files {
		if util.FileExists(file) {
			return file, nil
		}
	}

	return file, fmt.Errorf("default config resolution failed: %w", ErrConfigNotFound)
}
