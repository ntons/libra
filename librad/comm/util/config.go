package util

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/flosch/pongo2"
	"github.com/ghodss/yaml"
)

func LoadFromFile(filePath string, target interface{}) (err error) {
	tpl, err := pongo2.FromFile(filePath)
	if err != nil {
		return
	}
	b, err := tpl.ExecuteBytes(nil)
	if err != nil {
		return
	}
	switch ext := filepath.Ext(filePath); ext {
	case ".json":
	case ".yml", ".yaml":
		if b, err = yaml.YAMLToJSON(b); err != nil {
			return
		}
	default:
		return fmt.Errorf("unknown config file extension: %v", ext)
	}
	if err = json.Unmarshal(b, target); err != nil {
		return
	}
	return
}

func init() {
	for _, e := range []struct {
		names  []string
		filter pongo2.FilterFunction
	}{
		{
			names: []string{
				"env",
			},
			filter: filterEnv,
		},
	} {
		for _, name := range e.names {
			pongo2.RegisterFilter(name, e.filter)
		}
	}
}

// get value from environ
// eg: {{ default|env:name }}
func filterEnv(
	in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	key, value, found := param.String(), "", false
	for _, s := range os.Environ() {
		pair := strings.SplitN(s, "=", 2)
		if key == pair[0] {
			value, found = pair[1], true
			break
		}
	}
	if found {
		return pongo2.AsSafeValue(value), nil
	} else {
		return in, nil
	}
}
