package gateway

type config struct {
	Broadcast struct {
		Redis string `json:"redis"`
	} `json:"broadcast"`
}
