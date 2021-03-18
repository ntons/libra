package main

import (
	"testing"
)

func TestParseSize(t *testing.T) {
	Expect := func(s string, v int) {
		if r, err := ParseSize(s); err != nil {
			t.Errorf("failed to parse: %s", s)
		} else if r != v {
			t.Errorf("unexpected: %s,%d,%d", s, v, r)
		}
	}
	ExpectFail := func(s string) {
		if _, err := ParseSize(s); err == nil {
			t.Errorf("expect fail but not: %s", s)
		}
	}
	Expect(" 7", 7)
	Expect("7g ", 7*1000*1000*1000)
	Expect("7gib", 7*1024*1024*1024)
	Expect("7Mb", 7*1000*1000)
	Expect("7MiB", 7*1024*1024)
	ExpectFail("7Mbx")
	ExpectFail("7M b")
	ExpectFail("x7M")
	ExpectFail("x7M suf")
}
