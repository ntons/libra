package ranking

type chartConfig struct {
	Redis []string
}

type config struct {
	Bubblechart chartConfig `json:"bubblechart"`
	Leaderboard chartConfig `json:"leaderboard"`
}
