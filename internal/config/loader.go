package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"dario.cat/mergo"
	"github.com/Ensono/taskctl/pkg/utils"
	"github.com/mitchellh/mapstructure"
	"github.com/pelletier/go-toml"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// ErrConfigNotFound occurs when requested config file does not exists
var ErrConfigNotFound = errors.New("config file not found")

// Loader reads and parses config files
type Loader struct {
	dst           *Config
	imports       map[string]bool
	dir           string
	homeDir       string
	strictDecoder bool
}

// NewConfigLoader is Loader constructor
func NewConfigLoader(dst *Config) Loader {
	return Loader{
		dst:           dst,
		imports:       make(map[string]bool),
		homeDir:       utils.MustGetUserHomeDir(),
		dir:           utils.MustGetwd(),
		strictDecoder: false,
	}
}

func (c *Loader) WithDir(dir string) *Loader {
	c.dir = dir
	return c
}

// Dir
func (c *Loader) Dir() string {
	return c.dir
}

func (c *Loader) WithStrictDecoder() *Loader {
	c.strictDecoder = true
	return c
}

type loaderContext struct {
	Dir string
}

// Load loads and parses requested config file
func (cl *Loader) Load(file string) (*Config, error) {
	cl.reset()
	lc := &loaderContext{
		Dir: cl.dir,
	}

	_, err := cl.LoadGlobalConfig()
	if err != nil {
		return nil, err
	}

	if file == "" {
		file, err = cl.ResolveDefaultConfigFile()
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

	localCfg, err := buildFromDefinition(def, lc)
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

	cfg, err := buildFromDefinition(def, &loaderContext{})
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

	importDir := filepath.Dir(file)
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
	// this is only going to work on yaml files
	// this program seems to want to accept json/toml and yaml
	//
	// TODO: remove json/toml support - unnecessary
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

func (cl *Loader) readURL(urlStr string) (map[string]interface{}, error) {
	resp, err := http.Get(urlStr)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("%d: config request failed - %s", resp.StatusCode, urlStr)
	}

	// data, err := io.ReadAll(resp.Body)
	// if err != nil {
	// 	return nil, fmt.Errorf("%s: %v", urlStr, err)
	// }

	ext := ""
	ct := resp.Header.Get("Content-Type")

	mediaType, _, _ := mime.ParseMediaType(ct)

	switch mediaType {
	case "application/json":
		ext = ".json"
	case "application/x-yaml", "application/yaml", "text/yaml":
		ext = ".yaml"
	case "application/x-toml", "application/toml", "text/toml":
		ext = ".toml"
	default:
		up, err := url.Parse(urlStr)
		if err != nil {
			return cl.unmarshalDataStream(resp.Body, "")
		}
		ext = filepath.Ext(up.Path)
	}

	return cl.unmarshalDataStream(resp.Body, ext)
}

func (cl *Loader) readFile(filename string) (map[string]interface{}, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", filename, err)
	}

	return cl.unmarshalDataByte(data, filepath.Ext(filename))
}

func (cl *Loader) unmarshalDataByte(data []byte, ext string) (map[string]interface{}, error) {
	cm := make(map[string]any)

	switch strings.ToLower(ext) {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &cm); err != nil {
			return nil, err
		}
	case ".json":
		if err := json.Unmarshal(data, &cm); err != nil {
			return nil, err
		}
	case ".toml":
		if err := toml.Unmarshal(data, &cm); err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("unsupported config file type")
	}

	return cm, nil
}

func (cl *Loader) unmarshalDataStream(data io.Reader, ext string) (map[string]interface{}, error) {
	cm := make(map[string]any)

	switch strings.ToLower(ext) {
	case ".yaml", ".yml":
		if err := yaml.NewDecoder(data).Decode(&cm); err != nil {
			return nil, err
		}
	case ".json":
		if err := json.NewDecoder(data).Decode(&cm); err != nil {
			return nil, err
		}
	case ".toml":
		if err := toml.NewDecoder(data).Decode(&cm); err != nil {
			return nil, err
		}
	default:
		// speed up GC cycle if data is not read
		_, _ = io.Copy(io.Discard, data)
		return nil, errors.New("unsupported config file type")
	}

	return cm, nil
}

func (cl *Loader) decode(cm map[string]interface{}) (*ConfigDefinition, error) {
	c := &ConfigDefinition{}
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

func (cl *Loader) ResolveDefaultConfigFile() (file string, err error) {
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
