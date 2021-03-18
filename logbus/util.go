package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var sizeregexp = regexp.MustCompile(`^([0-9]+)([TtGgMmKk]?i?)[Bb]?$`)

func ParseSize(s string) (int, error) {
	a := sizeregexp.FindStringSubmatch(strings.TrimSpace(s))
	if len(a) != 3 {
		return 0, fmt.Errorf("bad size: %s", s)
	}
	v, err := strconv.Atoi(a[1])
	if err != nil {
		panic(fmt.Errorf("bad parsed value"))
	}
	switch strings.ToLower(a[2]) {
	case "t":
		v *= 1000 * 1000 * 1000 * 1000
	case "ti":
		v *= 1024 * 1024 * 1024 * 1024
	case "g":
		v *= 1000 * 1000 * 1000
	case "gi":
		v *= 1024 * 1024 * 1024
	case "m":
		v *= 1000 * 1000
	case "mi":
		v *= 1024 * 1024
	case "k":
		v *= 1000
	case "ki":
		v *= 1024
	}
	return v, nil
}
