package transform

import (
	"bytes"
	"fmt"
	"github.com/acarl005/stripansi"
	"github.com/karlderkaefer/cdk-notifier/config"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"regexp"
	"strings"
	"text/template"
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

func transformDiff(diffSymbol string, content string) string {
	lines := strings.Split(content, "\n")
	regexDiffSymbol := diffSymbol
	// plus needs to be escaped in regex
	if diffSymbol == "+" {
		regexDiffSymbol = "\\+"
	}
	regex := fmt.Sprintf("(.*\\[%s\\].*)", regexDiffSymbol)
	re := regexp.MustCompile(regex)
	var output []string
	for _, line := range lines {
		if re.MatchString(line) {
			// we gonna add diff symbol as first char of line
			newLine := re.ReplaceAllString(line, diffSymbol+"$1")
			// remove one trailing space to keep number of characters per line equal
			newLine = strings.Replace(newLine, diffSymbol+" ", diffSymbol, 1)
			output = append(output, newLine)
		} else {
			output = append(output, line)
		}
	}
	return strings.Join(output, "\n")
}

func (t *LogTransformer) TransformDiff() {
	t.LogContent = transformDiff("+", t.LogContent)
	t.LogContent = transformDiff("-", t.LogContent)
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
