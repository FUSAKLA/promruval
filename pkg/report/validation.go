package report

import (
	"bytes"
	"fmt"
	"html/template"
)

var htmlTemplate = `
<h1>Validation rules</h1>
{{- range .Rules }}
{{ $currentRule := . }}
  <h2><a href="#{{.Name}}">{{.Name}}</a></h2>
	  <ul>
	  {{- range .Validations }}
		<li>{{$currentRule.Scope}} {{.}}</li>
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

type templateRule struct {
	Name        string
	Scope       string
	Validations []string
}

type templateData struct {
	Rules []templateRule
}

func ValidationDocs(validationRules []ValidationRule, format string) (string, error) {
	data := templateData{}
	for _, rule := range validationRules {
		data.Rules = append(data.Rules, templateRule{
			Name:        rule.Name(),
			Scope:       rule.Scope(),
			Validations: rule.ValidationTexts(),
		})
	}

	templateToUse := textTemplate
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

	tmpl, err := template.New("docs").Parse(templateToUse)
	if err != nil {
		return "", err
	}
	buf := new(bytes.Buffer)
	if err := tmpl.Execute(buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
