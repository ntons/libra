package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/ntons/libra/librad/database"
)

func LoadFromBase64File(fp string) {
	b, err := ioutil.ReadFile("archive.txt")
	if err != nil {
		log.Fatal(err)
		return
	}
	b, err = base64.StdEncoding.DecodeString(string(b))
	if err != nil {
		log.Fatal(err)
		return
	}
	b, err = database.Decode(b)
	if err != nil {
		log.Fatal(err)
		return
	}
	var a anypb.Any
	if err = proto.Unmarshal(b, &a); err != nil {
		log.Fatal(err)
		return
	}
	fmt.Println(base64.StdEncoding.EncodeToString(a.Value))
}

func ParseData(b64 string) {
	b, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		log.Fatal(err)
		return
	}
	b, err = database.Decode(b)
	if err != nil {
		log.Fatal(err)
		return
	}
	var a anypb.Any
	if err = proto.Unmarshal(b, &a); err != nil {
		log.Fatal(err)
		return
	}
	fmt.Println(base64.StdEncoding.EncodeToString(a.Value))
}

type JsonData struct {
	Id  string `json:"_id"`
	Val *struct {
		Binary *struct {
			Base64 string `json:"base64"`
		} `json:"$binary"`
	} `json:"val"`
}

func LoadFromExportedFile(fp string) {
	b, err := ioutil.ReadFile(fp)
	if err != nil {
		log.Fatal(err)
		return
	}
	for _, e := range bytes.Split(b, []byte{'\n'}) {
		var v JsonData
		if err := json.Unmarshal(e, &v); err != nil {
			continue
		}
		if v.Val == nil || v.Val.Binary == nil {
			log.Fatal(string(e))
		}
		ParseData(v.Val.Binary.Base64)
	}
}
