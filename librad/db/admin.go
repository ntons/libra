package db

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

type Admin struct {
	Id          string        `bson:"_id"`
	Name        string        `bson:"name"`
	Description string        `bson:"description"`
	Secret      string        `bson:"secret,omitempty"`
	Permissions []*Permission `bson:"permissions,omitempty"`
}

func (x *Admin) parse() (err error) {
	for _, p := range x.Permissions {
		if err = p.parse(); err != nil {
			return
		}
	}
	return
}
func (x *Admin) IsPermitted(path string) bool {
	for _, p := range x.Permissions {
		if p.isPermitted(path) {
			return true
		}
	}
	return false
}

type xAdminIndex struct {
	a       []*Admin
	idIndex map[string]*Admin
}

func newAdminIndex(as []*Admin) *xAdminIndex {
	var idIndex = make(map[string]*Admin)
	for _, a := range as {
		idIndex[a.Id] = a
	}
	return &xAdminIndex{
		a:       as,
		idIndex: idIndex,
	}
}

func FindAdminById(id string) *Admin {
	a, _ := xAdmins.idIndex[id]
	return a
}
func loadAdmins(ctx context.Context) (err error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	collection, err := getAdminCollection(ctx)
	if err != nil {
		return fmt.Errorf("failed to get admin collection: %w", err)
	}
	cursor, err := collection.Find(ctx, bson.D{})
	if err != nil {
		return fmt.Errorf("failed to query admins: %w", err)
	}
	var res []*Admin
	if err = cursor.All(ctx, &res); err != nil {
		return
	}
	for _, a := range res {
		if err = a.parse(); err != nil {
			return
		}
	}
	xAdmins = newAdminIndex(res)
	return
}
