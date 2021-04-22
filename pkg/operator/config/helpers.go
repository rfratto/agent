package config

import (
	"reflect"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"gopkg.in/yaml.v2"
)

var (
	inlineYamlField    = `{{ .Name }}: {{ .Value | trim }}`
	multilineYamlField = `{{ .Name }}:
  {{- .Value | trim | nindent 2 }}`

	emptyFieldTemplate, inlineYamlFieldTemplate, multilineYamlFieldTemplate *template.Template
)

func init() {
	inlineYamlFieldTemplate = template.New("inline-field")
	inlineYamlFieldTemplate.Funcs(sprig.TxtFuncMap())
	_, err := inlineYamlFieldTemplate.Parse(inlineYamlField)
	if err != nil {
		panic(err)
	}

	multilineYamlFieldTemplate = template.New("multiline-field")
	multilineYamlFieldTemplate.Funcs(sprig.TxtFuncMap())
	_, err = multilineYamlFieldTemplate.Parse(multilineYamlField)
	if err != nil {
		panic(err)
	}
}

// yamlField is a helper function that makes removes boilerplate for optional
// YAML fields.
//
// Using
//
//   {{ yamlField "some_field" .FieldValue }}
//
// in a template is equivalent to:
//
//   {{ if .FieldValue }}some_field: {{ yaml .FieldValue }}{{ end }}
//
// v in yamlField will be marshaled as YAML. If v is an object or an array,
// it will be rendered as multiline, with subsequent lines indented by 2
// spaces. You may have to use sprig's nindent to indent further if the object
// is deeply nested.
func yamlField(name string, v interface{}) (string, error) {
	if reflect.ValueOf(v).IsZero() {
		return "", nil
	}

	bb, err := yaml.Marshal(v)
	if err != nil {
		return "", err
	}
	_ = bb

	// Gross hack: figure out what template we should use by unmarshaling the
	// value back to figure out if it's a map or array.
	var tmpl *template.Template
	var val interface{}
	if err := yaml.Unmarshal(bb, &val); err != nil {
		panic(err)
	}
	switch val.(type) {
	case map[interface{}]interface{}, []interface{}:
		tmpl = multilineYamlFieldTemplate
	default:
		tmpl = inlineYamlFieldTemplate
	}

	var sw strings.Builder
	err = tmpl.Execute(&sw, struct{ Name, Value string }{Name: name, Value: string(bb)})
	return sw.String(), err
}
