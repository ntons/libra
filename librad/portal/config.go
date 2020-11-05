package portal

type config struct {
	Redis string
	Mongo string
	// for token/ticket crypto
	Key string // 16/24/32 bytes
	IV  string // 16 bytes
}
