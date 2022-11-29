package normalizer

type Config struct {
	Separator string             `json:"separator"`
	Labels    []string           `json:"labels"`
	Patterns  []NormalizePattern `json:"patterns"`
}
