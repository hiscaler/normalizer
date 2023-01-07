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

## 数据规格说明

每一个合规的数据分为标签和值两部分，比如：`number:123`
`number` 为标签
`123` 为值

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
            ValueKey:      "funs",
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
map[string]interface {}{"age":12, "funs":[]interface {}{"Basketball", "Football", "Swimming"}, "name":"John"}
```

针对标签的匹配，分为严格、宽松两种模式，无论是哪种模式，处理引擎都将去掉标签前后的空格。宽松模式则更进一步，将标签中多余的空格（包括全角空格）压缩为一个，比如标签为：`you     name`，使用松散模式下将会转换为 `you name` 进行匹配。

针对每一个匹配规则，单独设置了 `match_method` 属性，0 为完整匹配，1 为模糊匹配，默认情况为 0。模糊匹配下，将会根据标签中是否包含对应的单词来确定匹配结果。

## 注意

在使用 Parse() 方法对文本进行解析之后，您应该使用 normalizer.Ok() 来判断处理结果是否存在错误，为 false 的情况下，您可以通过 normalizer.Errors() 来获取所有的错误信息，以决定后续的业务流程。

在使用 Parse() 方法之前，您也可以使用 Validate() 方法来判断您的设置是否正确。比如未设置键名或者为空、重复的 key 等错误存在的情况下，Validate() 方法将返回对应的错误。