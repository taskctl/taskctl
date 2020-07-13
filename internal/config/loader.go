package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/imdario/mergo"
	"github.com/mitchellh/mapstructure"
	"github.com/pelletier/go-toml"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"

	"github.com/taskctl/taskctl/pkg/utils"
)

// ErrConfigNotFound occurs when requested config file does not exists
var ErrConfigNotFound = errors.New("config file not found")

// Loader reads and parses config files
type Loader struct {
	dst     *Config
	imports map[string]bool
	dir     string
	homeDir string
}

// NewConfigLoader is Loader constructor
func NewConfigLoader(dst *Config) Loader {
	return Loader{
		dst:     dst,
		imports: make(map[string]bool),
		homeDir: utils.MustGetUserHomeDir(),
		dir:     utils.MustGetwd(),
	}
}

// Load loads and parses requested config file
func (cl *Loader) Load(file string) (*Config, error) {
	cl.reset()
	_, err := cl.LoadGlobalConfig()
	if err != nil {
		return nil, err
	}

	if file == "" {
		file, err = cl.resolveDefaultConfigFile()
		if err != nil {
			return cl.dst, err
		}
	}

	if !utils.IsURL(file) && !filepath.IsAbs(file) {
		file = path.Join(cl.dir, file)
	}

	raw, err := cl.load(file)
	if err != nil {
		return nil, err
	}

	def, err := cl.decode(raw)
	if err != nil {
		return nil, err
	}

	localCfg, err := buildFromDefinition(def)
	if err != nil {
		return nil, err
	}

	err = cl.dst.merge(localCfg)
	if err != nil {
		return nil, err
	}
	cl.dst.Variables.Set("Root", cl.dir)

	logrus.Debugf("config %s loaded", file)
	return cl.dst, nil
}

// LoadGlobalConfig load global config file  - ~/.taskctl/config.yaml
func (cl *Loader) LoadGlobalConfig() (*Config, error) {
	if cl.homeDir == "" {
		return nil, nil
	}

	file := path.Join(cl.homeDir, ".taskctl", "config.yaml")
	if !utils.FileExists(file) {
		return cl.dst, nil
	}

	raw, err := cl.load(file)
	if err != nil {
		return nil, err
	}

	def, err := cl.decode(raw)
	if err != nil {
		return nil, err
	}

	cfg, err := buildFromDefinition(def)
	if err != nil {
		return nil, err
	}

	err = cl.dst.merge(cfg)
	if err != nil {
		return nil, err
	}

	return cl.dst, err
}

func (cl *Loader) reset() {
	cl.imports = make(map[string]bool)
}

func (cl *Loader) load(file string) (config map[string]interface{}, err error) {
	cl.imports[file] = true

	if utils.IsURL(file) {
		config, err = cl.readURL(file)
	} else {
		if !utils.FileExists(file) {
			return config, fmt.Errorf("%s: %w", file, ErrConfigNotFound)
		}
		config, err = cl.readFile(file)
	}
	if err != nil {
		return nil, err
	}

	var raw map[string]interface{}
	importDir := path.Dir(file)
	if imports, ok := config["import"]; ok {
		for _, v := range imports.([]interface{}) {
			if utils.IsURL(v.(string)) {
				if cl.imports[v.(string)] {
					continue
				}
				raw, err = cl.load(v.(string))
			} else {
				importFile := path.Join(importDir, v.(string))
				if cl.imports[importFile] {
					continue
				}
				fi, err := os.Stat(importFile)
				if err != nil {
					return nil, fmt.Errorf("%s: %v", importFile, err)
				}
				if !fi.IsDir() {
					raw, err = cl.load(importFile)
				} else {
					raw, err = cl.loadDir(importFile)
				}
				if err != nil {
					logrus.Error(err)
				}
			}
			if err != nil {
				return nil, fmt.Errorf("load import error: %v", err)
			}

			err = mergo.Merge(&config, raw, mergo.WithOverride, mergo.WithAppendSlice, mergo.WithTypeCheck)
			if err != nil {
				return nil, err
			}
		}
	}

	return config, nil
}

func (cl *Loader) loadDir(dir string) (map[string]interface{}, error) {
	pattern := filepath.Join(dir, "*.yaml")
	q, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", dir, err)
	}

	cm := make(map[string]interface{})
	for _, importFile := range q {
		if cl.imports[importFile] {
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

func (cl *Loader) readURL(u string) (map[string]interface{}, error) {
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

	var ext string
	ct := resp.Header.Get("Content-Type")
	if ct != "" {
		mediaType, _, _ := mime.ParseMediaType(ct)
		if mediaType == "application/json" {
			ext = ".json"
		}
	}

	if ext == "" {
		up, err := url.Parse(u)
		if err == nil {
			ext = filepath.Ext(up.Path)
		}
	}

	if ext == "" {
		ext = ".yaml"
	}

	return cl.unmarshalData(data, ext)
}

func (cl *Loader) readFile(filename string) (map[string]interface{}, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", filename, err)
	}

	ext := filepath.Ext(filename)

	return cl.unmarshalData(data, ext)
}

func (cl *Loader) unmarshalData(data []byte, ext string) (map[string]interface{}, error) {
	var cm map[string]interface{}

	switch strings.ToLower(ext) {
	case ".yaml", ".yml":
		err := yaml.NewDecoder(bytes.NewReader(data)).Decode(&cm)
		if err != nil {
			return nil, err
		}
	case ".json":
		err := json.NewDecoder(bytes.NewReader(data)).Decode(&cm)
		if err != nil {
			return nil, err
		}
	case ".toml":
		err := toml.NewDecoder(bytes.NewReader(data)).Decode(&cm)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("unsupported config file type")
	}

	return cm, nil
}

func (cl *Loader) decode(cm map[string]interface{}) (*configDefinition, error) {
	c := &configDefinition{}
	md, _ := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
		),
		ErrorUnused:      true,
		WeaklyTypedInput: true,
		Result:           c,
		TagName:          "",
	})

	err := md.Decode(cm)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (cl *Loader) resolveDefaultConfigFile() (file string, err error) {
	dir := cl.dir
	for {
		if dir == filepath.Dir(dir) {
			break
		}

		for _, v := range DefaultFileNames {
			file := filepath.Join(dir, v)
			if utils.FileExists(file) {
				cl.dir = dir
				return file, nil
			}
		}

		dir = filepath.Dir(dir)
	}

	return file, fmt.Errorf("default config resolution failed: %w", ErrConfigNotFound)
}
