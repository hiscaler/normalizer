package normalizer

import (
	"github.com/hiscaler/gox/jsonx"
	"github.com/hiscaler/gox/stringx"
	"strconv"
	"strings"
)

const (
	booleanValueType = "boolean" // 布尔值类型
	stringValueType  = "string"  // 文本类型
	arrayValueType   = "array"   // 数组类型
	intValueType     = "int"     // 整数类型
	floatValueType   = "float"   // 浮点数类型
)

const (
	AccurateMatch = iota // 精准匹配
	BlurryMatch          // 模糊匹配（只要包含相应的文本即认为匹配成功）
)

type NormalizePattern struct {
	LabelKeywords  []string `json:"label_keywords"` // 标签关键词（可以有多个）
	MatchType      int      `json:"match_type"`     // 匹配方式
	Separator      string   `json:"separator"`      // 文本分隔符
	ValueKey       string   `json:"value_key"`      // 解析后返回数据中值使用的 key
	ValueTransform struct {
		Replaces   map[string]string `json:"replaces"`   // 需要替换的字符串
		Separators []string          `json:"separators"` // 分隔字符（返回为数组的时候可用）
	} `json:"value_transform"` // 值转化方式
	ValueType    string      `json:"value_type"`    // 值类型
	DefaultValue interface{} `json:"default_value"` // 默认值
}

type Normalizer struct {
	ok           bool                   // 解析是否成功
	Errors       []string               // 错误信息
	OriginalText string                 // 原始的文本
	Separator    string                 // 文本行分隔符
	Patterns     []NormalizePattern     // 解析规则
	Items        map[string]interface{} // 解析后返回的值
}

var noOriginalErrorMessage = "not set original text"

func NewNormalizer() *Normalizer {
	return &Normalizer{
		ok:        false,
		Errors:    []string{noOriginalErrorMessage},
		Separator: "\n",
		Items:     make(map[string]interface{}, 0),
	}
}

// SetSeparator 设置文本分隔符
func (n *Normalizer) SetSeparator(sep string) *Normalizer {
	n.Separator = sep
	return n
}

// SetOriginalText 设置要解析的文本内容
func (n *Normalizer) SetOriginalText(text string) *Normalizer {
	n.OriginalText = text
	errors := n.Errors
	if len(errors) > 0 && errors[0] == noOriginalErrorMessage {
		if len(errors) == 1 {
			n.Errors = make([]string, 0)
		} else {
			n.Errors = errors[1:]
		}
	}
	return n
}

func (n *Normalizer) SetPatterns(patterns []NormalizePattern) *Normalizer {
	n.Patterns = patterns
	for _, pattern := range n.Patterns {
		n.Items[pattern.ValueKey] = pattern.DefaultValue
	}
	return n
}

// Parse 文本解析
func (n *Normalizer) Parse() *Normalizer {
	if len(n.Patterns) == 0 {
		return n
	}
	for _, text := range strings.Split(n.OriginalText, n.Separator) {
		for _, pattern := range n.Patterns {
			for _, keyword := range pattern.LabelKeywords {
				if pattern.MatchType == BlurryMatch {
					keyword = strings.ToLower(keyword)
				}
				rowSep := pattern.Separator
				if rowSep == "" {
					rowSep = ":"
				}
				if !strings.Contains(text, rowSep) {
					continue
				}
				labelValue := strings.Split(text, rowSep)
				matched := false
				label := strings.TrimSpace(labelValue[0])
				if pattern.MatchType == BlurryMatch {
					matched = strings.Contains(label, keyword)
				} else {
					matched = strings.EqualFold(label, keyword)
				}
				if !matched {
					continue
				}

				rawValue := labelValue[1]
				for oldValue, newValue := range pattern.ValueTransform.Replaces {
					rawValue = strings.ReplaceAll(rawValue, oldValue, newValue)
				}
				rawValue = strings.TrimSpace(rawValue)

				var value interface{}
				var err error
				switch pattern.ValueType {
				case booleanValueType:
					value, err = strconv.ParseBool(rawValue)
				case intValueType:
					value, err = strconv.ParseInt(rawValue, 10, 64)
				case floatValueType:
					value, err = strconv.ParseFloat(rawValue, 64)
				case arrayValueType:
					value = stringx.Split(rawValue, pattern.ValueTransform.Separators...)
				default:
					// Value is string type
					value = rawValue
				}
				if err != nil {
					n.Errors = append(n.Errors, err.Error())
				}
				n.Items[pattern.ValueKey] = value
				break
			}
		}
	}
	n.ok = len(n.Errors) == 0
	return n
}

func (n *Normalizer) Ok() bool {
	return n.ok
}

// Output 输出 JSON 字符串
func (n *Normalizer) Output() string {
	return jsonx.ToJson(n.Items, "{}")
}
