package normalizer

import (
	"errors"
	"fmt"
	"github.com/hiscaler/gox/inx"
	"github.com/hiscaler/gox/jsonx"
	"github.com/hiscaler/gox/slicex"
	"github.com/hiscaler/gox/stringx"
	"github.com/spf13/cast"
	"go/types"
	"sort"
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
	ExactMatch = iota // 精准匹配
	FuzzyMatch        // 模糊匹配（只要包含相应的文本即认为匹配成功）
)

type ValueTransform struct {
	MatchMethod int               `json:"match_method"` // 匹配方式（0: 精准匹配、1: 模糊匹配）
	Replaces    map[string]string `json:"replaces"`     // 需要替换的字符串
	Separators  []string          `json:"separators"`   // 值分隔符（返回为数组的时候可用）
}

type NormalizePattern struct {
	Labels         []string       `json:"labels"`          // 标签关键词（可以有多个）
	MatchMethod    int            `json:"match_method"`    // 匹配方式（0: 精准匹配、1: 模糊匹配）
	Separator      string         `json:"separator"`       // 文本段分隔符
	ValueKey       string         `json:"value_key"`       // 解析后返回数据中值使用的 key
	ValueTransform ValueTransform `json:"value_transform"` // 值转化设置
	ValueType      string         `json:"value_type"`      // 值类型
	DefaultValue   interface{}    `json:"default_value"`   // 默认值
}

type Normalizer struct {
	labels       map[string]struct{}    // 文本中所有的标签
	Errors       []string               // 错误信息
	OriginalText string                 // 原始的文本
	Separator    string                 // 文本行分隔符
	Patterns     []NormalizePattern     // 解析规则
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

func cleanLabel(label string) string {
	if label == "" {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(label))
}

func (n *Normalizer) SetLabels(labels []string) *Normalizer {
	cleanedLabels := make(map[string]struct{}, len(labels))
	for _, label := range labels {
		label = cleanLabel(label)
		if label == "" {
			continue
		}
		cleanedLabels[label] = struct{}{}
	}
	n.labels = cleanedLabels
	return n
}

// SetPatterns 设置匹配规则
func (n *Normalizer) SetPatterns(patterns []NormalizePattern) *Normalizer {
	n.Patterns = patterns
	items := make(map[string]interface{}, len(patterns))
	for i, pattern := range n.Patterns {
		// 规则设置规则
		if pattern.Separator == "" {
			n.Patterns[i].Separator = ":" // Default separator value
		}
		for k := range pattern.ValueTransform.Replaces {
			if k == "" {
				delete(n.Patterns[i].ValueTransform.Replaces, k)
			}
		}
		for _, label := range pattern.Labels {
			label = cleanLabel(label)
			if label == "" {
				continue
			}
			if _, ok := n.labels[label]; !ok {
				n.labels[label] = struct{}{}
			}
		}
		// 防止默认值设置错误
		defaultValue := pattern.DefaultValue
		switch pattern.ValueType {
		case booleanValueType:
			defaultValue = cast.ToBool(defaultValue)
		case arrayValueType:
			switch defaultValue.(type) {
			case types.Slice:
				defaultValue = cast.ToSlice(defaultValue)
			default:
				defaultValue = []interface{}{}
			}
		case intValueType:
			defaultValue = cast.ToInt(defaultValue)
		case floatValueType:
			defaultValue = cast.ToFloat64(defaultValue)
		default:
			defaultValue = cast.ToString(defaultValue)
		}
		items[pattern.ValueKey] = defaultValue
	}
	n.Items = items
	n.Errors = []string{}
	return n
}

// Parse 文本解析
func (n *Normalizer) Parse() *Normalizer {
	n.Errors = []string{}
	if len(n.Patterns) == 0 || n.OriginalText == "" {
		return n
	}
	err := n.Validate()
	if err != nil {
		n.Errors = append(n.Errors, err.Error())
		return n
	}

	type labelValue struct {
		key            string
		label          string
		value          string
		valueType      string
		valueTransform ValueTransform
	}

	kvLines := make([]labelValue, 0)
	appendText := true
	for _, lineText := range strings.Split(n.OriginalText, n.Separator) {
		lineText = strings.TrimSpace(lineText)
		if lineText == "" {
			continue
		}

		isPureText := true // 是否为纯文本（不包含标签）
		lowerLineText := strings.ToLower(lineText)
		for label := range n.labels {
			if strings.HasPrefix(lowerLineText, label) {
				isPureText = false
				break
			}
		}
		if isPureText && appendText {
			m := len(kvLines)
			if m > 0 {
				m--
				kvLines = append(kvLines, labelValue{
					key:            kvLines[m].key,
					label:          kvLines[m].label,
					value:          lineText,
					valueType:      kvLines[m].valueType,
					valueTransform: kvLines[m].valueTransform,
				})
			} else {
				continue
			}
		}
		matched := false
		lv := labelValue{}
		for _, pattern := range n.Patterns {
			for _, keyword := range pattern.Labels {
				segmentSep := pattern.Separator
				if !strings.Contains(lineText, segmentSep) {
					continue
				}
				segments := strings.Split(lineText, segmentSep)
				label := strings.TrimSpace(segments[0])
				if pattern.MatchMethod == FuzzyMatch {
					matched = strings.Contains(label, strings.ToLower(keyword))
				} else {
					matched = strings.EqualFold(label, keyword)
				}
				if matched {
					lv.key = pattern.ValueKey
					lv.label = label
					lv.value = strings.TrimSpace(strings.Join(segments[1:], segmentSep))
					lv.valueType = pattern.ValueType
					lv.valueTransform = ValueTransform{
						MatchMethod: pattern.ValueTransform.MatchMethod,
						Replaces:    pattern.ValueTransform.Replaces,
						Separators:  pattern.ValueTransform.Separators,
					}
					break
				}
			}
			if matched {
				break
			}
		}
		if matched {
			appendText = true
			kvLines = append(kvLines, lv)
		} else {
			if !isPureText {
				appendText = false
			}
		}
	}

	for _, line := range kvLines {
		rawValue := line.value
		if len(line.valueTransform.Replaces) > 0 {
			if line.valueTransform.MatchMethod == FuzzyMatch {
				rawValue = strings.ToLower(rawValue)
				replaces := make(map[string]string, len(line.valueTransform.Replaces))
				for k, v := range line.valueTransform.Replaces {
					replaces[strings.ToLower(k)] = v
				}
				line.valueTransform.Replaces = replaces
			}

			// 根据字符的长度执行替换的顺序，比如替换 {"fourteen": 14, "four": 4} 的规则应用于
			// `fourteen,four` 替换后的值为 `14,4`
			keys := make([]string, 0)
			for k := range line.valueTransform.Replaces {
				keys = append(keys, k)
			}
			if len(keys) > 0 {
				sort.Slice(keys, func(i, j int) bool {
					return len(keys[i]) > len(keys[j])
				})
				oldNews := make([]string, len(keys)*2)
				for i, key := range keys {
					oldNews[i*2] = key
					oldNews[i*2+1] = line.valueTransform.Replaces[key]
				}
				rawValue = strings.NewReplacer(oldNews...).Replace(rawValue)
			}

			rawValue = strings.TrimSpace(rawValue)
		}
		var value interface{}
		switch line.valueType {
		case booleanValueType:
			if rawValue == "" {
				value = false
			} else {
				rawValue = strings.ToLower(rawValue)
				switch rawValue {
				case "y", "yes":
					value = true
				case "n", "no":
					value = false
				default:
					value, err = strconv.ParseBool(rawValue)
				}
			}
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

	return n
}

// Validate 验证设置是否有效
func (n *Normalizer) Validate() error {
	if n.Separator == "" {
		return errors.New("文本行分隔符不能为空")
	}
	m := len(n.Patterns)
	if m == 0 {
		return errors.New("未设置解析规则")
	}

	valueTypes := []string{stringValueType, booleanValueType, floatValueType, intValueType, arrayValueType}

	for i := range n.Patterns {
		p1 := n.Patterns[i]
		if strings.TrimSpace(p1.ValueKey) == "" {
			return fmt.Errorf("解析规则第 %d 项未设置键名或者为空", i+1)
		}
		if !inx.StringIn(p1.ValueType, valueTypes...) {
			return fmt.Errorf("解析规则第 %d 项返回值类型 %s 设置有误，有效的类型为：%s", i+1, p1.ValueType, strings.Join(valueTypes, ", "))
		}
		if len(p1.Labels) == 0 {
			return fmt.Errorf("解析规则第 %d 项未设置标签关键词", i+1)
		}
		for j := i + 1; j < m; j++ {
			p2 := n.Patterns[j]
			if strings.EqualFold(p1.ValueKey, p2.ValueKey) {
				return fmt.Errorf("解析规则第 %d 项与第 %d 项 %s 键名重复", i+1, j+1, p1.ValueKey)
			}
			for _, k1 := range p1.Labels {
				for _, k2 := range p2.Labels {
					if strings.EqualFold(k1, k2) {
						return fmt.Errorf("解析规则第 %d 项与第 %d 项 %s 标签关键词重复", i+1, j+1, k1)
					}
				}
			}
		}
	}

	return nil
}

// Ok 验证是否处理成功
func (n *Normalizer) Ok() bool {
	return len(n.Errors) == 0
}

// Output 输出 JSON 字符串
func (n *Normalizer) Output() string {
	return jsonx.ToPrettyJson(n.Items)
}
