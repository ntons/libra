package db

import (
	"encoding/json"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func msgToDoc(m proto.Message) (_ map[string]interface{}, err error) {
	b, err := (protojson.MarshalOptions{
		UseProtoNames:  true,
		UseEnumNumbers: true,
	}).Marshal(m)
	if err != nil {
		return
	}
	var d map[string]interface{}
	if err = json.Unmarshal(b, &d); err != nil {
		return
	}
	return d, nil
}

func docToMsg(d map[string]interface{}, m proto.Message) (err error) {
	b, err := json.Marshal(d)
	if err != nil {
		return
	}
	if err = (protojson.UnmarshalOptions{
		DiscardUnknown: true,
	}).Unmarshal(b, m); err != nil {
		return
	}
	return
}
