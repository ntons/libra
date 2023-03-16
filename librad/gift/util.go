package gift

import (
	"time"

	v1pb "github.com/ntons/libra-go/api/libra/v1"
	"github.com/ntons/libra/librad/db"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func giftFromData(in *v1pb.GiftData) (_ *db.Gift, err error) {
	out := &db.Gift{
		Id: in.Id,
	}
	if in.ExpireAt > 0 {
		out.ExpireAt = time.Unix(in.ExpireAt, 0)
	}
	if out.Payload, err = proto.Marshal(in.Payload); err != nil {
		return
	}
	return out, nil
}

func giftToData(in *db.Gift) (_ *v1pb.GiftData, err error) {
	out := &v1pb.GiftData{
		Id:      in.Id,
		Payload: &anypb.Any{},
	}
	if !in.ExpireAt.IsZero() {
		out.ExpireAt = in.ExpireAt.Unix()
	}
	if err = proto.Unmarshal(in.Payload, out.Payload); err != nil {
		return
	}
	return out, nil
}
