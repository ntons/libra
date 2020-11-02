package ranking

type bubbleConfig struct {
	Redis string
}
type leaderboardConfig struct {
	Redis string
}
type config struct {
	Bubble      bubbleConfig      `json:"bubble"`
	Leaderboard leaderboardConfig `json:"leaderboard"`
}
