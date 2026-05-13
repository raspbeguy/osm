package cmd

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
)

// tmplFuncs is the shared template.FuncMap for all --format flags.
var tmplFuncs = template.FuncMap{
	"date": func(s string) string {
		if len(s) >= 10 {
			return s[:10]
		}
		return s
	},
	"json": func(v any) (string, error) {
		b, err := json.Marshal(v)
		if err != nil {
			return "", err
		}
		return string(b), nil
	},
	"csv": func(args ...any) (string, error) {
		parts := make([]string, len(args))
		for i, a := range args {
			parts[i] = fmt.Sprint(a)
		}
		var buf bytes.Buffer
		w := csv.NewWriter(&buf)
		if err := w.Write(parts); err != nil {
			return "", err
		}
		w.Flush()
		return strings.TrimRight(buf.String(), "\r\n"), nil
	},
}
