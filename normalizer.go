package normalizer

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hiscaler/gox/inx"
	"github.com/hiscaler/gox/jsonx"
	"github.com/hiscaler/gox/slicex"
	"github.com/hiscaler/gox/stringx"
	"github.com/spf13/cast"
	"go/types"
	"regexp"
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
	// ExactMatch 精准匹配
	//
	// 必须是完整的匹配
	// 比如 name 仅仅匹配 name 这个文本，不会匹配 baby name, your name?, please input you name!
	ExactMatch = iota
	// FuzzyMatch 模糊匹配
	//
	// 只要包含相应的文本单词即认为匹配成功，需要注意这里匹配的是单词，而不是文本。
	//
	// 比如 name 匹配 baby name, your name?, please input you name!
	//
	// 但是不会匹配 baby username, you username?, please input you username!
	//
	// 原因是虽然 username 单词中包含 name 字符，但是 name 和 username 不是同一个单词，所以会匹配失败。
	FuzzyMatch
)

var (
	rxSpaceless            = regexp.MustCompile("\\s{2,}")
	spaceCharacterReplacer = strings.NewReplacer("　", " ") // 全角空格替换
)

type ValueTransform struct {
	MatchMethod int               `json:"match_method"` // 匹配方式（0: 精准匹配、1: 模糊匹配）
	Replaces    map[string]string `json:"replaces"`     // 需要替换的字符串
	Separators  []string          `json:"separators"`   // 值分隔符（返回为数组的时候可用）
}

type NormalizePattern struct {
	used           bool           // 是否使用过（用于内部判断是否需要使用该规则）
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
	separator    string                 // 文本行分隔符
	strictMode   bool                   // 严格模式
	validate     bool                   // 设置是否有效
	Errors       []string               // 错误信息
	OriginalText string                 // 原始的文本
	Patterns     []NormalizePattern     // 解析规则
	Items        map[string]interface{} // 解析后返回的值
}

func NewNormalizer() *Normalizer {
	return &Normalizer{
		Errors:    []string{},
		separator: "\n",
		Items:     make(map[string]interface{}, 0),
	}
}

// SetSeparator 设置文本行分隔符
func (n *Normalizer) SetSeparator(sep string) *Normalizer {
	if sep == "" {
		sep = "\n"
	}
	n.separator = sep
	return n
}

// SetStrictMode 设置是否为严格模式，默认为宽松模式
//
// 与之相对应的是宽松模式，严格模式下，会区分标签字符的大小写，且不会去掉设置标签和定制内容标间单词之间多余的空格，
// 宽松模式下则会去掉，仅保留一个。
//
// 比如：
//
// "Please input you      name" 在宽松模式下则会变为 "please input you name" 进行比较
func (n *Normalizer) SetStrictMode(strictMode bool) *Normalizer {
	n.strictMode = strictMode
	return n
}

// SetOriginalText 设置要解析的文本内容
func (n *Normalizer) SetOriginalText(text string) *Normalizer {
	n.OriginalText = strings.TrimSpace(text)
	n.Items = map[string]interface{}{}
	n.Errors = []string{}
	return n
}

func clean(label string, strictMode bool) string {
	if label == "" {
		return ""
	}
	if strictMode {
		return strings.TrimSpace(label)
	}
	label = rxSpaceless.ReplaceAllLiteralString(spaceCharacterReplacer.Replace(label), " ")
	return strings.ToLower(strings.TrimSpace(label))
}

func (n *Normalizer) SetLabels(labels []string) *Normalizer {
	cleanedLabels := make(map[string]struct{}, len(labels))
	for _, label := range labels {
		label = clean(label, n.strictMode)
		if label != "" {
			cleanedLabels[label] = struct{}{}
		}
	}
	n.labels = cleanedLabels
	return n
}

// SetPatterns 设置匹配规则
func (n *Normalizer) SetPatterns(patterns []NormalizePattern) *Normalizer {
	n.validate = false
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
		for j, label := range pattern.Labels {
			label = clean(label, n.strictMode)
			n.Patterns[i].Labels[j] = label
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

	for i := range n.Patterns {
		// Reset
		n.Patterns[i].used = false
	}

	type labelValue struct {
		key            string
		label          string
		value          string
		valueType      string
		valueTransform ValueTransform
	}

	lines := make([]labelValue, 0)
	appendText := true
	for _, lineText := range strings.Split(n.OriginalText, n.separator) {
		lineText = strings.TrimSpace(lineText)
		if lineText == "" {
			continue
		}

		isPureText := true // 是否为纯文本（不包含标签）
		cleanedLineText := clean(lineText, n.strictMode)
		for label := range n.labels {
			if strings.HasPrefix(cleanedLineText, label) {
				isPureText = false
				break
			}
		}
		if isPureText && appendText {
			m := len(lines)
			if m == 0 {
				continue
			}
			m--
			lines = append(lines, labelValue{
				key:            lines[m].key,
				label:          lines[m].label,
				value:          lineText,
				valueType:      lines[m].valueType,
				valueTransform: lines[m].valueTransform,
			})
		}
		matched := false
		lv := labelValue{}
		for i, pattern := range n.Patterns {
			if pattern.used {
				continue
			}
			separatorIndex := strings.Index(lineText, pattern.Separator)
			if separatorIndex == -1 {
				continue
			}
			label := clean(lineText[0:separatorIndex], n.strictMode)
			for _, keyword := range pattern.Labels {
				if pattern.MatchMethod == FuzzyMatch {
					// 匹配单词（忽略大小写）
					reg := regexp.MustCompile(`(?i)(^|([\s\t\n]+))(` + keyword + `)($|([\s\t\n]+))`)
					matched = reg.MatchString(label)
				} else {
					matched = label == keyword
				}
				if matched {
					lv.key = pattern.ValueKey
					lv.label = label
					lv.value = strings.TrimSpace(lineText[separatorIndex+1:])
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
				n.Patterns[i].used = true
				break
			}
		}
		if matched {
			appendText = true
			lines = append(lines, lv)
		} else {
			if !isPureText {
				appendText = false
			}
		}
	}

	for _, line := range lines {
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
	if n.validate {
		return nil
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

	n.validate = true
	return nil
}

// Ok 验证是否处理成功
func (n *Normalizer) Ok() bool {
	return len(n.Errors) == 0
}

// ToJson 输出 JSON 字符
func (n *Normalizer) ToJson() string {
	return jsonx.ToPrettyJson(n.Items)
}

// ToJsonRawMessage 转换为 json.RawMessage
func (n *Normalizer) ToJsonRawMessage() (json.RawMessage, error) {
	return jsonx.ToRawMessage(n.Items, "{}")
}
