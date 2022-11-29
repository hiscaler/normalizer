Normalizer
==========
文本规范器

用于根据指定的规则解析文本内容，并返回符合您标准的一个对象。比如

```text
name:John\nage: 12 years\nfuns:Basketball,Football and Swimming
```

通过解析后将转变为

```json
{
  "name": "John",
  "age": "12",
  "funs": [
    "Basketball",
    "Football",
    "Swimming"
  ]
}
```

## 安装

```go
go get github.com/hiscaler/normalizer
```

## 使用
```go
normalizer = NewNormalizer()
normalizer.SetOriginalText("name:John\\nage: 12 years\\nmy fun:Basketball,Football and Swimming").
    SetSeparator("\\n").
    SetLabels([]string{"name", "age", "my fun"}).
    SetPatterns([]NormalizePattern{
        {
            LabelKeywords: []string{"name"},
            MatchType:     0,
            Separator:     ":",
            ValueKey:      "name",
            ValueType:     "string",
            DefaultValue:  "",
        },
        {
            LabelKeywords: []string{"age"},
            MatchType:     0,
            Separator:     ":",
            ValueKey:      "age",
            ValueType:     "int",
            ValueTransform: ValueTransform{
                MatchType: 0,
                Replaces: map[string]string{
                    "years": "",
                },
                Separators: nil,
            },
            DefaultValue: 10,
        },
        {
            LabelKeywords: []string{"my fun"},
            MatchType:     0,
            Separator:     ":",
            ValueKey:      "fun",
            ValueType:     "array",
            ValueTransform: ValueTransform{
                MatchType:  0,
                Separators: []string{",", "and"},
            },
            DefaultValue: []interface{}{},
        },
    }).
    Parse()

fmt.Printf("items = %#v", normalizer.Items)
```

将会输出

```go
map[string]interface {}{"age":12, "fun":[]interface {}{"Basketball", "Football", "Swimming"}, "name":"John"}
```