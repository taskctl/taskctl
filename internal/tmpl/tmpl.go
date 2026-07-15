// Package tmpl renders Go text/template strings with taskctl's helper funcs.
package tmpl

import (
	"bytes"
	"reflect"
	"text/template"
)

// RenderString parses given string as a template and executes it with provided params
func RenderString(tmpl string, variables map[string]any) (string, error) {
	funcMap := template.FuncMap{
		"default": func(arg any, value any) any {
			v := reflect.ValueOf(value)
			switch v.Kind() {
			case reflect.String, reflect.Slice, reflect.Array, reflect.Map:
				if v.Len() == 0 {
					return arg
				}
			case reflect.Bool:
				if !v.Bool() {
					return arg
				}
			default:
				return value
			}

			return value
		},
	}

	var buf bytes.Buffer
	t, err := template.New("interpolate").Funcs(funcMap).Option("missingkey=error").Parse(tmpl)
	if err != nil {
		return "", err
	}

	err = t.Execute(&buf, variables)

	return buf.String(), err
}
