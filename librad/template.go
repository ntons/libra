package main

import (
	"os"
	"strings"

	"github.com/flosch/pongo2"
)

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
