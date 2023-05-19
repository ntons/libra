package internal

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"strings"
	"testing"
	"time"
)

func TestSuffixTree1(t *testing.T) {
	x := NewSuffixTree()
	x.Add("a")
	x.Add("ab")
	x.Add("ac")
	if b, err := json.MarshalIndent(x, "", "  "); err == nil {
		fmt.Printf("%s\n", b)
	}
	x.Del("ab")
	if b, err := json.MarshalIndent(x, "", "  "); err == nil {
		fmt.Printf("%s\n", b)
	}
	x.Add("bcd")
	fmt.Println(x.Search("c"))
}

func TestSuffixTree2(t *testing.T) {
	inSlice := func(s string, a []string) bool {
		for _, _s := range a {
			if _s == s {
				return true
			}
		}
		return false
	}

	check := func(a []string, x *SuffixTree) {
		n := 10000
		//m := time.Now()
		for _i := 0; _i < n; _i++ {
			r := []rune(a[rand.Intn(len(a))])
			if len(r) == 0 {
				continue
			}
			i, j := rand.Intn(len(r)), rand.Intn(len(r))
			if i == j {
				continue
			} else if i > j {
				i, j = j, i
			}
			k := string(r[i:j])
			v := x.Search(k)
			// 验证：
			// 所有字符串都包含键值
			// 其他字符串都不包含键值
			for _, s := range v {
				//fmt.Printf("checking: %v, %v\n", s, k)
				if !strings.Contains(s, k) {
					t.Fatalf("%s not contains %s", s, k)
				}
			}
			for _, s := range a {
				if inSlice(s, v) {
					continue
				}
				//fmt.Printf("checking: %v, %v\n", s, k)
				if strings.Contains(s, k) {
					t.Fatalf("%s contains %s", s, k)
				}
			}
		}
		//fmt.Printf("查询%v次耗时%v\n", n, time.Since(m))
	}

	b, err := ioutil.ReadFile("nickname.txt")
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	a := strings.Split(string(b), "\n")
	//for n, s := range a {
	//	fmt.Printf("%d %q\n", n, s)
	//}

	fmt.Printf("总共%v个字符串\n", len(a))

	rand.Shuffle(len(a), func(i, j int) { a[i], a[j] = a[j], a[i] })

	x := NewSuffixTree()

	m := time.Now()
	for _, s := range a {
		if len(s) == 0 {
			fmt.Printf("排除空串\n")
			continue
		}
		if !x.Add(s) {
			fmt.Printf("添加失败%v\n", s)
		}
	}
	fmt.Printf("构造字典树耗时%v\n", time.Since(m))
	//fmt.Printf("字典树中字符串数%v\n", len(x.HashToValue))
	//fmt.Printf("序列化后大小%v\n", proto.Size(x))
	check(a, x)

	aa := make([]string, 0)
	for _i := 0; _i < 1000; _i++ {
		i := rand.Intn(len(a))
		s := a[i]
		if len(s) == 0 {
			continue
		}
		x.Del(s)
		aa = append(aa, a[i])
		a = append(a[:i], a[i+1:]...)
	}
	check(a, x)

	for _i := 0; _i < 1000; _i++ {
		i := rand.Intn(len(aa))
		s := aa[i]
		if len(s) == 0 {
			continue
		}
		if x.Add(s) {
			a = append(a, s)
		}
	}
	check(a, x)
}
