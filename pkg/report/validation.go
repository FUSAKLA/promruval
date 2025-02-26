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
  <br/>
  <h2><a href="#{{.Name}}">{{.Name}}</a></h2>
	  {{- if .OnlyIf }}
  	  <h4>Only if ALL the following conditions are met:</h4>
	  <ul>
	  {{- range .OnlyIf }}
		<li>{{. | backticksToCodeTag | indentedToNewLines | escape }}</li>
	  {{- end }}
	  </ul>
	  {{- end }}
	  <h4>Following conditions MUST be met:</h4>
	  <ul>
	  {{- range .Validations }}
		<li>{{. | backticksToCodeTag | indentedToNewLines | escape }}</li>
	  {{- end }}
	  </ul>
{{- end }}
`

var markdownTemplate = `
# Validation rules
{{- range .Rules }}

## {{.Name}}
{{- if .OnlyIf }}
#### Only if ALL the following conditions are met:
{{- range .OnlyIf }}
  - {{. | escape}}
{{- end }}
{{- end }}
#### Following conditions MUST be met:
{{- range .Validations }}
  - {{. | escape}}
{{- end }}
{{- end }}
`

var textTemplate = `
Validation rules:
{{- range .Rules }}

  {{.Name}} ({{.Scope}})
	{{- if .OnlyIf }}
    Only if ALL the following conditions are met:
	{{- range .OnlyIf }}
      - {{. | escape}}
	{{- end }}
	{{- end }}
    Following conditions MUST be met:
    {{- range .Validations }}
      - {{. | escape}}
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
	Validations []string
	OnlyIf      []string
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
			OnlyIf:      rule.OnlyIfValidationTexts(),
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
