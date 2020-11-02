package main

import (
	"fmt"

	"github.com/ghodss/yaml"

	"github.com/ntons/tongo/template"
)

func init() {
	cmds["get profile"] = getProfile
	cmds["set profile"] = setProfile
	cmds["list profile"] = listProfile
	cmds["flush profile"] = flushProfile
}

func getProfile(args ...string) (err error) {
	return get("profiles", args...)
}

func setProfile(args ...string) (err error) {
	var target string
	if len(args) > 0 {
		target = args[0]
	}
	for name, prof := range cfg.Profiles {
		if target != "" && target != name {
			continue
		}
		var b []byte
		if b, err = template.RenderFile(
			prof.Template, prof.Args); err != nil {
			return fmt.Errorf("failed to render config: %v", err)
		}
		if kapi == nil {
			fmt.Printf("Profile of %s:\n%s\n", name, b)
			return
		}
		if b, err = yaml.YAMLToJSON(b); err != nil {
			return fmt.Errorf("failed to convert YAML to JSON: %v", err)
		}
		if err = set("profiles", name, string(b)); err != nil {
			return
		}
	}
	return
}

func listProfile(args ...string) (err error) {
	return list("profiles", args...)
}

func flushProfile(args ...string) (err error) {
	return flush("profiles", args...)
}
