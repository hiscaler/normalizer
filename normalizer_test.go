package normalizer

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"strings"
	"testing"
)

var normalizer *Normalizer
var configs map[string]Config
var samples []sample

type sample struct {
	UseName     string                 `json:"useName"`
	Description string                 `json:"description"`
	Text        string                 `json:"text"`
	Ok          bool                   `json:"ok"`
	Want        map[string]interface{} `json:"want"`
}

func TestMain(m *testing.M) {
	var b []byte
	var err error
	b, err = os.ReadFile("./testdata/configs.json")
	if err != nil {
		panic(fmt.Sprintf("Read configs.json file error: %s", err.Error()))
	}

	err = json.Unmarshal(b, &configs)
	if err != nil {
		panic(fmt.Sprintf("Parse config.json file error: %s", err.Error()))
	}

	b, err = os.ReadFile("./testdata/samples.json")
	if err != nil {
		panic(fmt.Sprintf("Read samples.json file error: %s", err.Error()))
	}

	err = json.Unmarshal(b, &samples)
	if err != nil {
		panic(fmt.Sprintf("Parse samples.json file error: %s", err.Error()))
	}

	normalizer = NewNormalizer()
	m.Run()
}

func TestNormalizer_Parse(t *testing.T) {
	for _, d := range samples {
		useName := d.UseName
		for name, c := range configs {
			if !strings.EqualFold(useName, name) {
				continue
			}
			normalizer.SetOriginalText(d.Text).
				SetSeparator(c.Separator).
				SetLabels(c.Labels).
				SetPatterns(c.Patterns).
				Parse()
			assert.Equal(t, nil, normalizer.Validate(), "%s - %s Validate() error: %#v", name, d.Description, normalizer.Errors)
			assert.Equal(t, d.Ok, normalizer.Ok(), "%s - %s Ok() error: %#v", name, d.Description, normalizer.Errors)
			items := normalizer.Items
			for k, v := range items {
				if vv, ok := v.([]string); ok {
					interfaceValues := make([]interface{}, len(vv))
					for i := range vv {
						interfaceValues[i] = vv[i]
					}
					items[k] = interfaceValues
				} else if vv, ok := v.(int64); ok {
					items[k] = float64(vv)
				}
			}
			assert.Equal(t, d.Want, items, "%s - %s 项目比对错误：%#v", name, d.Description, normalizer.Errors)
		}
	}
}

func Example() {
	normalizer = NewNormalizer()
	normalizer.SetOriginalText("name:John\\nage: 12 years\\nmy fun:Basketball,Football and Swimming").
		SetSeparator("\\n").
		SetLabels([]string{"name", "age", "my fun"}).
		SetPatterns([]NormalizePattern{
			{
				Labels:       []string{"name"},
				MatchMethod:  0,
				Separator:    ":",
				ValueKey:     "name",
				ValueType:    "string",
				DefaultValue: "",
			},
			{
				Labels:      []string{"age"},
				MatchMethod: 0,
				Separator:   ":",
				ValueKey:    "age",
				ValueType:   "int",
				ValueTransform: ValueTransform{
					MatchMethod: 0,
					Replaces: map[string]string{
						"years": "",
					},
					Separators: nil,
				},
				DefaultValue: 10,
			},
			{
				Labels:      []string{"my fun"},
				MatchMethod: 0,
				Separator:   ":",
				ValueKey:    "fun",
				ValueType:   "array",
				ValueTransform: ValueTransform{
					MatchMethod: 0,
					Separators:  []string{",", "and"},
				},
				DefaultValue: []interface{}{},
			},
		}).
		Parse()

	fmt.Printf("items = %#v", normalizer.Items)
}

func normalizerExecute() {
	normalizer = NewNormalizer()
	normalizer.SetOriginalText("name:John\\nage: 12 years\\nmy fun:Basketball,Football and Swimming").
		SetSeparator("\\n").
		SetLabels([]string{"name", "age", "my fun"}).
		SetPatterns([]NormalizePattern{
			{
				Labels:       []string{"name"},
				MatchMethod:  0,
				Separator:    ":",
				ValueKey:     "name",
				ValueType:    "string",
				DefaultValue: "",
			},
			{
				Labels:      []string{"age"},
				MatchMethod: 0,
				Separator:   ":",
				ValueKey:    "age",
				ValueType:   "int",
				ValueTransform: ValueTransform{
					MatchMethod: 0,
					Replaces: map[string]string{
						"years": "",
					},
					Separators: nil,
				},
				DefaultValue: 10,
			},
			{
				Labels:      []string{"my fun"},
				MatchMethod: 0,
				Separator:   ":",
				ValueKey:    "fun",
				ValueType:   "array",
				ValueTransform: ValueTransform{
					MatchMethod: 0,
					Separators:  []string{",", "and"},
				},
				DefaultValue: []interface{}{},
			},
		}).
		Parse()
}

func Benchmark(b *testing.B) {
	for i := 0; i < b.N; i++ {
		normalizerExecute()
	}
}
