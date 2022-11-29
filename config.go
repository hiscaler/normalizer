package normalizer

type Config struct {
	Name      string             `json:"name"`
	Separator string             `json:"separator"`
	Labels    []string           `json:"labels"`
	Patterns  []NormalizePattern `json:"patterns"`
}
