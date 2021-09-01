package transform

import (
	"bytes"
	"github.com/acarl005/stripansi"
	"github.com/karlderkaefer/cdk-notifier/config"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"regexp"
	"strings"
	"text/template"
	"unicode/utf8"
)

type LogTransformer struct {
	LogContent string
	Logfile    string
	TagId      string
}

type GithubTemplate struct {
	TagId     string
	Content   string
	JobLink   string
	Backticks string
}

func NewLogTransformer(config *config.AppConfig) *LogTransformer {
	return &LogTransformer{
		LogContent: "",
		Logfile:    config.LogFile,
		TagId:      config.TagId,
	}
}

func (t *LogTransformer) ReadFile() error {
	content, err := ioutil.ReadFile(t.Logfile)
	if err != nil {
		return err
	}
	t.LogContent = string(content)
	return nil
}

func (t *LogTransformer) RemoveAnsiCode() {
	t.LogContent = stripansi.Strip(t.LogContent)
}

func trimFirstRune(s string) string {
	_, i := utf8.DecodeRuneInString(s)
	return s[i:]
}

func (t *LogTransformer) TransformDiff() {
	lines := strings.Split(t.LogContent, "\n")
	var output []string
	for _, line := range lines {
		// https://regex101.com/r/9ORjxP/1
		regex := regexp.MustCompile(`(?m)(?:(?:\[(?P<resourcesSymbol>[\+-]+)\])|(?:│\s{1}(?P<securitySymbol>[\+-]+)\s{1}│))`)
		matches := regex.FindStringSubmatch(line)
		var foundSymbol string
		for i, m := range matches {
			// we got two possible matches
			// 1. [+] or [-] (group resourceSymbol)
			// 2. | + | or | - | (group securitySymbol)
			// if we hit one of those conditions we capture symbol
			if i != 0 && m != "" {
				foundSymbol = m
				logrus.Tracef("Detected change for symbol %s for line %s", foundSymbol, line)
			}
		}
		// replace first character of line with the diff symbol
		modifiedLine := line
		if foundSymbol != "" {
			// keep first character for resource elements
			if !strings.HasPrefix(line, "[") {
				modifiedLine = trimFirstRune(line)
			}
			modifiedLine = foundSymbol + modifiedLine
		}
		output = append(output, modifiedLine)
	}
	t.LogContent = strings.Join(output, "\n")
}

func (t *LogTransformer) Truncate() {
	// Message:Body is too long (maximum is 65536 characters)
	runes := bytes.Runes([]byte(t.LogContent))
	if len(runes) > 65000 {
		truncated := string(runes[:65000])
		truncated += "\n...truncated"
		t.LogContent = truncated
	}
}

func (t *LogTransformer) AddHeader() {
	templateContent := `
## cdk diff for {{ .TagId }} {{ .JobLink }}
{{ .Backticks }}diff
{{ .Content }}
{{ .Backticks }}
`
	githubTemplate := &GithubTemplate{
		TagId:     t.TagId,
		Content:   t.LogContent,
		Backticks: "```",
		JobLink:   "",
	}
	tmpl, err := template.New("githubTemplate").Parse(templateContent)
	if err != nil {
		logrus.Fatal(err)
	}
	stringWriter := bytes.NewBufferString("")
	err = tmpl.Execute(stringWriter, githubTemplate)
	if err != nil {
		logrus.Fatal(err)
	}
	t.LogContent = stringWriter.String()
}

func (t *LogTransformer) PrintFile() {
	logrus.Infof("File contents: %s", t.LogContent)
}

func (t *LogTransformer) Process() {
	err := t.ReadFile()
	if err != nil {
		logrus.Fatal(err)
	}
	t.RemoveAnsiCode()
	t.TransformDiff()
	t.AddHeader()
}
