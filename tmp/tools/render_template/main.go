package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/ntons/tongo/template"
)

func main() {
	var out string
	flag.StringVar(&out, "o", "", "output file path")
	if flag.Parse(); len(flag.Args()) < 1 {
		flag.Usage()
		os.Exit(1)
	}
	b, err := template.RenderFile(flag.Args()[0], nil)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if out != "" {
		ioutil.WriteFile(out, b, 0644)
	} else {
		fmt.Println(string(b))
	}
}
