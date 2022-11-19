package normalizer

import (
	"fmt"
	"github.com/hiscaler/gox/jsonx"
	"github.com/hiscaler/gox/slicex"
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

type valueTransform struct {
	MatchType  int               `json:"match_type"` // 匹配方式
	Replaces   map[string]string `json:"replaces"`   // 需要替换的字符串
	Separators []string          `json:"separators"` // 值分隔符（返回为数组的时候可用）
}

type NormalizePattern struct {
	LabelKeywords  []string       `json:"label_keywords"`  // 标签关键词（可以有多个）
	MatchType      int            `json:"match_type"`      // 匹配方式
	Separator      string         `json:"separator"`       // 文本段分隔符
	ValueKey       string         `json:"value_key"`       // 解析后返回数据中值使用的 key
	ValueTransform valueTransform `json:"value_transform"` // 值转化设置
	ValueType      string         `json:"value_type"`      // 值类型
	DefaultValue   interface{}    `json:"default_value"`   // 默认值
}

type Normalizer struct {
	Errors       []string               // 错误信息
	OriginalText string                 // 原始的文本
	Separator    string                 // 文本行分隔符
	Patterns     []NormalizePattern     // 解析规则
	IgnoreLabels []string               // 忽略的标签
	Items        map[string]interface{} // 解析后返回的值
}

func NewNormalizer() *Normalizer {
	return &Normalizer{
		Errors:    []string{},
		Separator: "\n",
		Items:     make(map[string]interface{}, 0),
	}
}

// SetSeparator 设置文本行分隔符
func (n *Normalizer) SetSeparator(sep string) *Normalizer {
	n.Separator = sep
	return n
}

// SetOriginalText 设置要解析的文本内容
func (n *Normalizer) SetOriginalText(text string) *Normalizer {
	n.OriginalText = strings.TrimSpace(text)
	n.Items = map[string]interface{}{}
	n.Errors = []string{}
	return n
}

func (n *Normalizer) SetIgnoreLabels(labels []string) *Normalizer {
	n.IgnoreLabels = labels
	return n
}

// SetPatterns 设置匹配规则
func (n *Normalizer) SetPatterns(patterns []NormalizePattern) *Normalizer {
	n.Patterns = patterns
	items := make(map[string]interface{}, len(patterns))
	for _, pattern := range n.Patterns {
		items[pattern.ValueKey] = pattern.DefaultValue
	}
	n.Items = items
	n.Errors = []string{}
	return n
}

func (n *Normalizer) notInLabels(label string) bool {
	for _, pattern := range n.Patterns {
		for _, s := range pattern.LabelKeywords {
			if strings.EqualFold(s, label) {
				return true
			}
		}
	}
	return false
}

// Parse 文本解析
func (n *Normalizer) Parse() *Normalizer {
	n.Errors = []string{}
	if len(n.Patterns) == 0 {
		return n
	}

	type labelValue struct {
		key            string
		label          string
		value          string
		valueType      string
		valueTransform valueTransform
	}
	validLines := make([]labelValue, 0)
	for _, lineText := range strings.Split(n.OriginalText, n.Separator) {
		lineText = strings.TrimSpace(lineText)
		if lineText == "" {
			continue
		}
		ignore := false
		for _, label := range n.IgnoreLabels {
			if strings.HasPrefix(strings.ToLower(lineText), strings.ToLower(label)) {
				ignore = true
				break
			}
		}
		if ignore {
			continue
		}
		matched := false
		lv := labelValue{}
		for _, pattern := range n.Patterns {
			for _, keyword := range pattern.LabelKeywords {
				if pattern.MatchType == BlurryMatch {
					keyword = strings.ToLower(keyword)
				}
				segmentSep := pattern.Separator
				if segmentSep == "" {
					segmentSep = ":"
				}
				if !strings.Contains(lineText, segmentSep) {
					continue
				}
				segments := strings.Split(lineText, segmentSep)
				label := strings.TrimSpace(segments[0])
				if pattern.MatchType == BlurryMatch {
					matched = strings.Contains(label, keyword)
				} else {
					matched = strings.EqualFold(label, keyword)
				}
				if matched {
					lv.key = pattern.ValueKey
					lv.label = label
					rawValue := segments[1]
					if pattern.ValueTransform.MatchType == BlurryMatch && len(pattern.ValueTransform.Replaces) > 0 {
						rawValue = strings.ToLower(rawValue)
					}
					for oldValue, newValue := range pattern.ValueTransform.Replaces {
						rawValue = strings.ReplaceAll(rawValue, oldValue, newValue)
					}
					lv.value = strings.TrimSpace(rawValue)
					lv.valueType = pattern.ValueType
					lv.valueTransform = valueTransform{
						MatchType:  pattern.ValueTransform.MatchType,
						Replaces:   pattern.ValueTransform.Replaces,
						Separators: pattern.ValueTransform.Separators,
					}
					break
				}
			}
			if matched {
				break
			}
		}
		if matched {
			validLines = append(validLines, lv)
		} else {
			m := len(validLines)
			if m > 1 {
				m--
				validLines = append(validLines, labelValue{
					key:            validLines[m].key,
					label:          validLines[m].label,
					value:          lineText,
					valueType:      validLines[m].valueType,
					valueTransform: validLines[m].valueTransform,
				})
			}
		}
	}

	for _, line := range validLines {
		rawValue := line.value
		var value interface{}
		var err error
		switch line.valueType {
		case booleanValueType:
			value, err = strconv.ParseBool(rawValue)
		case intValueType:
			value, err = strconv.ParseInt(rawValue, 10, 64)
		case floatValueType:
			value, err = strconv.ParseFloat(rawValue, 64)
		case arrayValueType:
			value = slicex.StringToInterface(stringx.Split(rawValue, line.valueTransform.Separators...))
		default:
			// Value is string type
			value = rawValue
		}
		if err != nil {
			n.Errors = append(n.Errors, err.Error())
		}
		if v, ok := n.Items[line.key]; ok {
			switch line.valueType {
			case stringValueType:
				if v != "" {
					v = fmt.Sprintf("%s\n%s", v, value)
				} else {
					v = value
				}
				n.Items[line.key] = v
			case arrayValueType:
				n.Items[line.key] = append(v.([]interface{}), value.([]interface{})...)
			default:
				n.Items[line.key] = value
			}
		} else {
			n.Items[line.key] = value
		}
	}

	// for _, lineText := range strings.Split(n.OriginalText, n.Separator) {
	// 	matched := false
	// 	for _, pattern := range n.Patterns {
	// 		for _, keyword := range pattern.LabelKeywords {
	// 			if pattern.MatchType == BlurryMatch {
	// 				keyword = strings.ToLower(keyword)
	// 			}
	// 			segmentSep := pattern.Separator
	// 			if segmentSep == "" {
	// 				segmentSep = ":"
	// 			}
	// 			if !strings.Contains(lineText, segmentSep) {
	// 				continue
	// 			}
	// 			segments := strings.Split(lineText, segmentSep)
	// 			label := strings.TrimSpace(segments[0])
	// 			if pattern.MatchType == BlurryMatch {
	// 				matched = strings.Contains(label, keyword)
	// 			} else {
	// 				matched = strings.EqualFold(label, keyword)
	// 			}
	// 			if !matched {
	// 				continue
	// 			}
	// 			currentValueKey = pattern.ValueKey
	//
	// 			rawValue := segments[1]
	// 			if pattern.ValueTransform.MatchType == BlurryMatch && len(pattern.ValueTransform.Replaces) > 0 {
	// 				rawValue = strings.ToLower(rawValue)
	// 			}
	// 			for oldValue, newValue := range pattern.ValueTransform.Replaces {
	// 				rawValue = strings.ReplaceAll(rawValue, oldValue, newValue)
	// 			}
	// 			rawValue = strings.TrimSpace(rawValue)
	//
	// 			var value interface{}
	// 			var err error
	// 			switch pattern.ValueType {
	// 			case booleanValueType:
	// 				value, err = strconv.ParseBool(rawValue)
	// 			case intValueType:
	// 				value, err = strconv.ParseInt(rawValue, 10, 64)
	// 			case floatValueType:
	// 				value, err = strconv.ParseFloat(rawValue, 64)
	// 			case arrayValueType:
	// 				value = stringx.Split(rawValue, pattern.ValueTransform.Separators...)
	// 			default:
	// 				// Value is string type
	// 				value = rawValue
	// 			}
	// 			if err != nil {
	// 				n.Errors = append(n.Errors, err.Error())
	// 			}
	// 			n.Items[pattern.ValueKey] = value
	// 			break
	// 		}
	// 	}
	// 	if !matched && currentValueKey != "" {
	// 		valueType := ""
	//
	// 		if v, ok := n.Items[currentValueKey]; ok {
	// 			switch pattern.ValueType {
	// 			case stringValueType:
	// 				n.Items[currentValueKey] = fmt.Sprintf("%s\n%s", v, lineText)
	// 			case arrayValueType:
	// 				n.Items[currentValueKey] = append(v.([]string), lineText.([]string)...)
	// 			default:
	// 				n.Items[currentValueKey] = lineText
	// 			}
	// 		}
	// 	}
	// }

	return n
}

// Ok 验证是否处理成功
func (n *Normalizer) Ok() bool {
	return len(n.Errors) == 0
}

// Output 输出 JSON 字符串
func (n *Normalizer) Output() string {
	return jsonx.ToPrettyJson(n.Items)
}
