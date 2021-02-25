package syncer

type ReMonConfig struct {
	Redis []string `json:"redis"`
	Mongo string   `json:"mongo"`
}

func (cfg *ReMonConfig) parse() (err error) {
	return
}

type config struct {
	Database ReMonConfig `json:"database"`
	MailBox  ReMonConfig `json:"mailbox"`
}

var cfg = &config{}
