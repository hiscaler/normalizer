package normalizer

import (
	"encoding/json"
	"fmt"
	"github.com/hiscaler/gox/inx"
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
	Output    string
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
			if normalizer.Ok() != d.Ok {
				t.Errorf("%s normalizer.Ok() 期望 %v, 实际 %v", tag, d.Ok, normalizer.ok)
			}
			if normalizer.Output() != d.Output {
				t.Errorf("%s normalizer.Output() 期望 %s, 实际 %s", tag, d.Output, normalizer.Output())
			}
			t.Logf("normalizer.Errors = %#v", normalizer.Errors)
		}
	}
}
