package transform

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/acarl005/stripansi"
	"github.com/karlderkaefer/cdk-notifier/config"
	"github.com/karlderkaefer/cdk-notifier/provider"
	"github.com/sirupsen/logrus"
)

// LogTransformer is responsible to process the log file and do following transformation steps
// 1. Clean any ANSI chars and XTERM color created from cdk diff command
// 2. Transform additions and removals to markdown diff syntax
// 3. Create unique message header
// 4. truncate content if message is longer than GitHub API can handle
type LogTransformer struct {
	LogContent                string
	Logfile                   string
	TagID                     string
	NoPostMode                bool
	Vcs                       string
	DisableCollapse           bool
	ShowOverview              bool
	NumberOfDifferencesString string
	NumberReplaces            int
	Template                  string
	CustomTemplate            string
}

// NewLogTransformer create new log transfer based on config.AppConfig
func NewLogTransformer(config *config.NotifierConfig) *LogTransformer {
	return &LogTransformer{
		LogContent:      "",
		Logfile:         config.LogFile,
		TagID:           config.TagID,
		NoPostMode:      config.NoPostMode,
		Vcs:             config.Vcs,
		DisableCollapse: config.DisableCollapse,
		ShowOverview:    config.ShowOverview,
		Template:        config.Template,
		CustomTemplate:  config.CustomTemplate,
	}
}

func (t *LogTransformer) readFile() error {
	content, err := os.ReadFile(t.Logfile)
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
	var numberOfDifferencesString string
	var numberOfReplaces int
	for _, line := range lines {
		// https://regex101.com/r/XtxJgT/1
		regexNumberOfDifferencesString := regexp.MustCompile(`Number of stacks with differences:.*`)
		matchesNumberOfDifferencesString := regexNumberOfDifferencesString.FindStringSubmatch(line)
		if matchesNumberOfDifferencesString != nil {
			numberOfDifferencesString = matchesNumberOfDifferencesString[0]
		}

		// https://regex101.com/r/0eYw20/1
		regexNumberOfReplaces := regexp.MustCompile(`\(requires replacement\)|\(may cause replacement\)`)
		matchesNumberOfReplaces := regexNumberOfReplaces.FindStringSubmatch(line)
		if len(matchesNumberOfReplaces) > 0 {
			numberOfReplaces++
		}

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
	t.NumberOfDifferencesString = numberOfDifferencesString
	t.NumberReplaces = numberOfReplaces
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
	collapsible := false
	showOverview := false
	// only github and gitlab support collapsable sections
	if t.Vcs == "github" || t.Vcs == "gitlab" {
		collapsible = true
	}
	// can be disable by command line
	if t.DisableCollapse {
		collapsible = false
	}
	// can be activated by command line
	if t.ShowOverview {
		showOverview = true
	}
	template := &commentTemplate{
		TagID:                     t.TagID,
		NumberOfDifferencesString: t.NumberOfDifferencesString,
		NumberReplaces:            t.NumberReplaces,
		Content:                   t.LogContent,
		Backticks:                 "```",
		JobLink:                   "",
		HeaderPrefix:              provider.HeaderPrefix,
		Collapsible:               collapsible,
		ShowOverview:              showOverview,
		Template:                  t.Template,
		customTemplate:            t.CustomTemplate,
	}
	t.LogContent = template.render()
}

func (t *LogTransformer) printFile() {
	logrus.Infof("File contents: %s", t.LogContent)
}

// writeDiffFile is writing the transformed diff to file and appends .diff to filename.
// Additionally, the diff is streamed to stdout
func (t *LogTransformer) writeDiffFile() error {
	if !t.NoPostMode {
		return nil
	}
	filePath := t.Logfile + ".diff"
	// read/write for the owner, and read-only for the group and others
	err := os.WriteFile(filePath, []byte(t.LogContent), 0644)
	if err != nil {
		return err
	}
	fmt.Print(t.LogContent)
	return nil
}

// Process log file
// 1. Clean any ANSI chars and XTERM color created from cdk diff command
// 2. Transform additions and removals to markdown diff syntax
// 3. Create unique message header
// 4. truncate content if message is longer than GitHub API can handle
// 5. write diff as file and to stdout when no-post-mode is activated
func (t *LogTransformer) Process() {
	err := t.readFile()
	if err != nil {
		logrus.Fatal(err)
	}
	t.removeAnsiCode()
	t.transformDiff()
	t.addHeader()
	t.truncate()
	err = t.writeDiffFile()
	if err != nil {
		logrus.Fatal(err)
	}
}
