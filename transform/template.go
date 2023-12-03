package transform

import (
	"bytes"
	"os"
	"text/template"

	"github.com/sirupsen/logrus"
)

var defaultTemplate = `
{{ .HeaderPrefix }} {{ .TagID }} {{ .JobLink }}
{{- if .Collapsible }}
<details>
<summary>Click to expand</summary>
{{- end }}

{{ .Backticks }}diff
{{ .Content }}
{{ .Backticks }}
{{- if .Collapsible }}
</details>
{{- end }}
`

var extendedTemplate = `
{{ .HeaderPrefix }} {{ .TagID }} {{ .JobLink }}
{{ .NumberOfDifferencesString }}
{{- if .NumberReplaces }}
⚠️ Number of resources that require replacement: {{ .NumberReplaces }}
{{- end }}
{{- if .Collapsible }}
<details>
<summary>Click to expand</summary>
{{- end }}

{{ .Backticks }}diff
{{ .Content }}
{{ .Backticks }}
{{- if .Collapsible }}
</details>
{{- end }}
`

var extendedWithResourcesTemplate = `
{{ .HeaderPrefix }} {{ .TagID }} {{ .JobLink }}
{{ .NumberOfDifferencesString }}
{{- if .NumberReplaces }}
⚠️ Number of resources that require replacement: {{ .NumberReplaces }}
{{- end }}
{{- if .ChangedBaseResource }}
### Resources that are subject of change
{{- range $key, $value := .ChangedBaseResource }}
{{ $key }}: {{ $value.Count }}{{ if $value.Replaced }} (required replacement){{ end }}
{{- end }}
{{- end }}
{{- if .Collapsible }}
<details>
<summary>Click to expand</summary>
{{- end }}

{{ .Backticks }}diff
{{ .Content }}
{{ .Backticks }}
{{- if .Collapsible }}
</details>
{{- end }}
`

// commentTemplate wrapper object to use go templating
type commentTemplate struct {
	TagID                     string
	Content                   string
	JobLink                   string
	Backticks                 string
	HeaderPrefix              string
	Collapsible               bool
	ShowOverview              bool
	NumberOfDifferencesString string
	NumberReplaces            int
	ChangedBaseResource       map[string]ResourceMetric
	Template                  string // template type
	customTemplate            string // template file or string
}

type TemplateStrategy interface {
	getTemplateContent() string
}

type DefaultTemplate struct{}

func (d DefaultTemplate) getTemplateContent() string {
	return defaultTemplate
}

type ExtendedTemplate struct{}

func (e ExtendedTemplate) getTemplateContent() string {
	return extendedTemplate
}

type ExtendedWithResourcesTemplate struct{}

func (e ExtendedWithResourcesTemplate) getTemplateContent() string {
	return extendedWithResourcesTemplate
}

type CustomTemplate struct {
	TemplateContent string
}

func (ct CustomTemplate) getTemplateContent() string {
	return ct.TemplateContent
}

func (t *commentTemplate) ChooseTemplate() TemplateStrategy {
	// TODO deprecated
	if t.ShowOverview {
		t.Template = "extended"
	}
	// If customTemplate is set, use it as the template
	if t.customTemplate != "" {
		templateContent, err := t.getCustomTemplate()
		if err != nil {
			logrus.Fatal(err)
		}
		return CustomTemplate{TemplateContent: templateContent}
	}
	logrus.Debugf("Using template %s", t.Template)
	// If customTemplate is not set, use the template type
	switch t.Template {
	case "default":
		return DefaultTemplate{}
	case "extended":
		return ExtendedTemplate{}
	case "extendedWithResources":
		return ExtendedWithResourcesTemplate{}
	default:
		logrus.Warnf("Template %s not found, using default template", t.Template)
		return DefaultTemplate{}
	}
}

// getCustomTemplate reads the file from provided file path. If the file path is not valid, it will use the string as the template
func (ct *commentTemplate) getCustomTemplate() (string, error) {
	// Check if customTemplate is a valid file path
	_, err := os.Stat(ct.customTemplate)
	if err == nil {
		// If it's a valid file path, read the file
		b, err := os.ReadFile(ct.customTemplate)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	logrus.Warnf("Template file %s not found, using string as template", ct.customTemplate)
	// If it's not a valid file path, use customTemplate as the template string
	return ct.customTemplate, nil
}

func (t *commentTemplate) render() (string, error) {
	templateContent := t.ChooseTemplate().getTemplateContent()
	logrus.Debugf("Using template content %s", templateContent)
	tmpl, err := template.New("commentTemplate").Parse(templateContent)
	if err != nil {
		return "", err
	}
	stringWriter := bytes.NewBufferString("")
	err = tmpl.Execute(stringWriter, t)
	if err != nil {
		return "", err
	}
	return stringWriter.String(), nil
}
