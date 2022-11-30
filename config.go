package normalizer

type Config struct {
	Separator string             `json:"separator"` // 行分隔符
	Labels    []string           `json:"labels"`    // 文本中涉及到的所有标签
	Patterns  []NormalizePattern `json:"patterns"`  // 匹配规则
}
