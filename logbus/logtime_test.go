package main

import (
	"fmt"
	"testing"
)

func TestLogTime(t *testing.T) {
	ts := `{"@timestamp":"2021-03-16T09:22:10.032Z"}`
	fmt.Println(ts)
	fmt.Println("local: ", getLogTime([]byte(ts), true))
	fmt.Println("utc:   ", getLogTime([]byte(ts), false))
}
