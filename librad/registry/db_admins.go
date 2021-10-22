package registry

type xAdmin struct {
	Id          string         `bson:"_id"`
	Name        string         `bson:"name"`
	Description string         `bson:"description"`
	Secret      string         `bson:"secret,omitempty"`
	Permissions []*xPermission `bson:"permissions,omitempty"`
}

func (x *xAdmin) parse() (err error) {
	for _, p := range x.Permissions {
		if err = p.parse(); err != nil {
			return
		}
	}
	return
}
func (x *xAdmin) isPermitted(path string) bool {
	for _, p := range x.Permissions {
		if p.isPermitted(path) {
			return true
		}
	}
	return false
}

type xAdminIndex struct {
	a       []*xAdmin
	idIndex map[string]*xAdmin
}

func newAdminIndex(as []*xAdmin) *xAdminIndex {
	var idIndex = make(map[string]*xAdmin)
	for _, a := range as {
		idIndex[a.Id] = a
	}
	return &xAdminIndex{
		a:       as,
		idIndex: idIndex,
	}
}

func findAdminById(id string) *xAdmin {
	a, _ := xAdmins.idIndex[id]
	return a
}
