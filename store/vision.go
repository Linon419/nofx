package store

// VisionImageMeta describes a rendered chart image used for vision-capable models.
type VisionImageMeta struct {
	Name      string `json:"name"`
	Symbol    string `json:"symbol"`
	Timeframe string `json:"timeframe"`
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
}
