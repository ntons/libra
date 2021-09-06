package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/ntons/libra/librad/database"
)

func main() {
	b, err := ioutil.ReadFile("archive.txt")
	if err != nil {
		fmt.Println(err)
		return
	}
	b, err = base64.StdEncoding.DecodeString(string(b))
	if err != nil {
		fmt.Println(err)
		return
	}
	b, err = database.Decode(b)
	if err != nil {
		fmt.Println(err)
		return
	}
	var a anypb.Any
	if err = proto.Unmarshal(b, &a); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(base64.StdEncoding.EncodeToString(a.Value))
}
