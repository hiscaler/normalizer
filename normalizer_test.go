package normalizer

import (
	"encoding/json"
	"fmt"
	"github.com/hiscaler/gox/inx"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

var normalizer *Normalizer
var patterns []pattern
var texts []text

type pattern struct {
	Tags     []string
	Patterns []NormalizePattern
}

type text struct {
	Tag       string
	Text      string
	Separator string
	Ok        bool
	Want      map[string]interface{}
}

func TestMain(m *testing.M) {
	var b []byte
	var err error
	b, err = os.ReadFile("./testdata/patterns.json")
	if err != nil {
		panic(fmt.Sprintf("Read patterns.json file error: %s", err.Error()))
	}

	err = json.Unmarshal(b, &patterns)
	if err != nil {
		panic(fmt.Sprintf("Parse patterns.json file error: %s", err.Error()))
	}

	b, err = os.ReadFile("./testdata/texts.json")
	if err != nil {
		panic(fmt.Sprintf("Read texts.json file error: %s", err.Error()))
	}

	err = json.Unmarshal(b, &texts)
	if err != nil {
		panic(fmt.Sprintf("Parse texts.json file error: %s", err.Error()))
	}

	normalizer = NewNormalizer()
	m.Run()
}

func TestNormalizer_Parse(t *testing.T) {
	for _, d := range texts {
		tag := d.Tag
		for _, p := range patterns {
			if !inx.StringIn(tag, p.Tags...) {
				continue
			}
			normalizer.SetOriginalText(d.Text).
				SetSeparator(d.Separator).
				SetPatterns(p.Patterns).
				Parse()
			assert.Equal(t, d.Ok, normalizer.Ok(), "ok", normalizer.Errors)
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
			assert.Equal(t, d.Want, items, "items", normalizer.Errors)
		}
	}
}
