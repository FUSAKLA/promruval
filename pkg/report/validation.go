package report

import (
	"bytes"
	"fmt"
	"html/template"
	"regexp"
)

var htmlTemplate = `
<h1>Validation rules</h1>
{{- range .Rules }}
{{ $currentRule := . }}
  <h2><a href="#{{.Name}}">{{.Name}}</a></h2>
	  <ul>
	  {{- range .Validations }}
		<li>{{$currentRule.Scope}} {{. | backticksToCodeTag | indentedToNewLines | escape }}</li>
	  {{- end }}
	  </ul>
{{- end }}
`

var markdownTemplate = `
# Validation rules
{{- range .Rules }}
{{ $currentRule := . }}
## {{.Name}}
  {{- range .Validations }}
  - {{$currentRule.Scope}} {{.}}
  {{- end }}
{{- end }}
`

var textTemplate = `
Validation rules:
{{- range .Rules }}
{{ $currentRule := . }}
  {{.Name}}
	{{- range .Validations }}
    - {{$currentRule.Scope}} {{.}}
	{{- end }}
{{- end }}
`

var customFuncs = template.FuncMap{
	"backticksToCodeTag": func(s string) string {
		return regexp.MustCompile("`([^`]+)`").ReplaceAllString(s, "<code>$1</code>")
	},
	"indentedToNewLines": func(s string) string {
		return regexp.MustCompile(`( {4,})`).ReplaceAllString(s, "<br/>$1")
	},
	"escape": func(s string) template.HTML {
		return template.HTML(s)
	},
}

type templateRule struct {
	Name        string
	Scope       string
	Validations map[string][]string
}

type templateData struct {
	Rules []templateRule
}

func ValidationDocs(validationRules []ValidationRule, format string) (string, error) {
	data := templateData{}
	for _, rule := range validationRules {
		data.Rules = append(data.Rules, templateRule{
			Name:        rule.Name(),
			Scope:       string(rule.Scope()),
			Validations: rule.ValidationTexts(),
		})
	}

	var templateToUse string
	switch format {
	case "text":
		templateToUse = textTemplate
	case "html":
		templateToUse = htmlTemplate
	case "markdown":
		templateToUse = markdownTemplate
	default:
		return "", fmt.Errorf("unsupported format type %s", format)
	}

	tmpl, err := template.New("docs").Funcs(customFuncs).Parse(templateToUse)
	if err != nil {
		return "", err
	}
	buf := new(bytes.Buffer)
	if err := tmpl.Execute(buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
