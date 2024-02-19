package transform

import (
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
	ChangedBaseResource       map[string]ResourceMetric
	Template                  string
	CustomTemplate            string
	ProcessorsChain           LineProcessor
}

type ResourceMetric struct {
	Count    int
	Replaced bool
}

// A LineProcessor is responsible to process a single line.
// It can either just extract information from the line or modify it.
type LineProcessor interface {
	// Return the modified line
	ProcessLine(line string, lt *LogTransformer) string
	SetNext(processor LineProcessor)
}

type BaseProcessor struct {
	next LineProcessor
}

func (p *BaseProcessor) SetNext(next LineProcessor) {
	p.next = next
}

func (p *BaseProcessor) ProcessLine(line string, lt *LogTransformer) string {
	if p.next != nil {
		return p.next.ProcessLine(line, lt)
	}
	return line
}

// NewLogTransformer create new log transfer based on config.AppConfig
func NewLogTransformer(config *config.NotifierConfig) *LogTransformer {
	lt := &LogTransformer{
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
	lt.initProcessorsChain()
	return lt
}

func (t *LogTransformer) initProcessorsChain() {
	t.ChangedBaseResource = make(map[string]ResourceMetric)
	stackDiffProcessor := &StackDiffProcessor{}
	numberReplacesProcessor := &NumberReplacesProcessor{}
	diffSymbolProcessor := &DiffSymbolProcessor{}
	resourceDiffExtractorProcessor := &ResourceDiffExtractorProcessor{}
	stackDiffProcessor.SetNext(resourceDiffExtractorProcessor)
	resourceDiffExtractorProcessor.SetNext(numberReplacesProcessor)
	numberReplacesProcessor.SetNext(diffSymbolProcessor)
	t.ProcessorsChain = stackDiffProcessor
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

// Get the number of changed stacks
type StackDiffProcessor struct {
	BaseProcessor
}

func (p *StackDiffProcessor) ProcessLine(line string, lt *LogTransformer) string {
	// https://regex101.com/r/9ORjxP/1
	regexNumberOfDifferencesString := regexp.MustCompile(`Number of stacks with differences:.*`)
	matchesNumberOfDifferencesString := regexNumberOfDifferencesString.FindStringSubmatch(line)
	if matchesNumberOfDifferencesString != nil {
		lt.NumberOfDifferencesString = matchesNumberOfDifferencesString[0]
	}
	return p.BaseProcessor.ProcessLine(line, lt)
}

// Get the count of replaced resources
type NumberReplacesProcessor struct {
	BaseProcessor
}

func (p *NumberReplacesProcessor) ProcessLine(line string, lt *LogTransformer) string {
	// https://regex101.com/r/0eYw20/1
	regexNumberOfReplaces := regexp.MustCompile(`\(requires replacement\)|\(may cause replacement\)`)
	matchesNumberOfReplaces := regexNumberOfReplaces.FindStringSubmatch(line)
	if len(matchesNumberOfReplaces) > 0 {
		lt.NumberReplaces++
	}
	return p.BaseProcessor.ProcessLine(line, lt)
}

// Collect number of AWS base type changes
type ResourceDiffExtractorProcessor struct {
	BaseProcessor
}

func (p *ResourceDiffExtractorProcessor) ProcessLine(line string, lt *LogTransformer) string {
	// https://regex101.com/r/rBmjEp/2
	regex := regexp.MustCompile(`\s*\[(-|\+|~)] (AWS::\w+::\w+).*?(?P<replace>(replace|replaced)?$)`)
	matches := regex.FindStringSubmatch(line)
	if len(matches) > 0 {
		awsBaseResource := matches[2]
		replaced := matches[3] != ""
		resource, exists := lt.ChangedBaseResource[awsBaseResource]
		if exists {
			resource.Count++
			// if replace was already detected, keep it
			resource.Replaced = resource.Replaced || replaced
		} else {
			resource = ResourceMetric{
				Count:    1,
				Replaced: replaced,
			}
		}
		lt.ChangedBaseResource[awsBaseResource] = resource
	}
	return p.BaseProcessor.ProcessLine(line, lt)
}

// Transform additions and removals to markdown diff syntax
type DiffSymbolProcessor struct {
	BaseProcessor
}

func (p *DiffSymbolProcessor) ProcessLine(line string, lt *LogTransformer) string {
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
	return p.BaseProcessor.ProcessLine(modifiedLine, lt)
}

func (t *LogTransformer) transformDiff() {
	lines := strings.Split(t.LogContent, "\n")
	var transformedLines []string

	for _, line := range lines {
		processedLine := t.ProcessorsChain.ProcessLine(line, t)
		transformedLines = append(transformedLines, processedLine)
	}
	t.LogContent = strings.Join(transformedLines, "\n")
}

func (t *LogTransformer) addHeader() {
	collapsible := false
	showOverview := false
	// only github and gitlab support collapsable sections
	if t.Vcs == "github" || t.Vcs == "github-enterprise" || t.Vcs == "gitlab" {
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
		ChangedBaseResource:       t.ChangedBaseResource,
		Content:                   t.LogContent,
		Backticks:                 "```",
		JobLink:                   "",
		HeaderPrefix:              provider.HeaderPrefix,
		Collapsible:               collapsible,
		ShowOverview:              showOverview,
		Template:                  t.Template,
		customTemplate:            t.CustomTemplate,
	}
	content, err := template.render()
	if err != nil {
		logrus.Fatal(err)
	}
	t.LogContent = content
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
// 4. write diff as file and to stdout when no-post-mode is activated
func (t *LogTransformer) Process() {
	err := t.readFile()
	if err != nil {
		logrus.Fatal(err)
	}
	t.removeAnsiCode()
	t.transformDiff()
	t.addHeader()
	err = t.writeDiffFile()
	if err != nil {
		logrus.Fatal(err)
	}
}
