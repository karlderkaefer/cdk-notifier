package transform

import (
	"bytes"
	"github.com/acarl005/stripansi"
	"github.com/karlderkaefer/cdk-notifier/ci"
	"github.com/karlderkaefer/cdk-notifier/config"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"regexp"
	"strings"
	"text/template"
	"unicode/utf8"
)

// LogTransformer is responsible to process the log file and do following transformation steps
// 1. Clean any ANSI chars and XTERM color created from cdk diff command
// 2. Transform additions and removals to markdown diff syntax
// 3. Create unique message header
// 4. truncate content if message is longer than GitHub API can handle
type LogTransformer struct {
	LogContent string
	Logfile    string
	TagID      string
}

// githubTemplate wrapper object to use go templating
type githubTemplate struct {
	TagID        string
	Content      string
	JobLink      string
	Backticks    string
	HeaderPrefix string
}

// NewLogTransformer create new log transfer based on config.AppConfig
func NewLogTransformer(config *config.NotifierConfig) *LogTransformer {
	return &LogTransformer{
		LogContent: "",
		Logfile:    config.LogFile,
		TagID:      config.TagID,
	}
}

func (t *LogTransformer) readFile() error {
	content, err := ioutil.ReadFile(t.Logfile)
	if err != nil {
		return err
	}
	t.LogContent = string(content)
	return nil
}

func (t *LogTransformer) removeAnsiCode() {
	t.LogContent = stripansi.Strip(t.LogContent)
}

func trimFirstRune(s string) string {
	_, i := utf8.DecodeRuneInString(s)
	return s[i:]
}

func (t *LogTransformer) transformDiff() {
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

// truncate to avoid Message:Body is too long (maximum is 65536 characters)
func (t *LogTransformer) truncate() {
	runes := bytes.Runes([]byte(t.LogContent))
	if len(runes) > 65000 {
		truncated := string(runes[:65000])
		truncated += "\n...truncated"
		t.LogContent = truncated
	}
}

func (t *LogTransformer) addHeader() {
	templateContent := `
{{ .HeaderPrefix }} {{ .TagID }} {{ .JobLink }}
{{ .Backticks }}diff
{{ .Content }}
{{ .Backticks }}
`
	githubTemplate := &githubTemplate{
		TagID:        t.TagID,
		Content:      t.LogContent,
		Backticks:    "```",
		JobLink:      "",
		HeaderPrefix: ci.HeaderPrefix,
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

func (t *LogTransformer) printFile() {
	logrus.Infof("File contents: %s", t.LogContent)
}

// Process log file
// 1. Clean any ANSI chars and XTERM color created from cdk diff command
// 2. Transform additions and removals to markdown diff syntax
// 3. Create unique message header
// 4. truncate content if message is longer than GitHub API can handle
func (t *LogTransformer) Process() {
	err := t.readFile()
	if err != nil {
		logrus.Fatal(err)
	}
	t.removeAnsiCode()
	t.transformDiff()
	t.addHeader()
	t.truncate()
}
