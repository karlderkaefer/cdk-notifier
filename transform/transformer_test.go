package transform

import (
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/karlderkaefer/cdk-notifier/config"
	"github.com/stretchr/testify/assert"
)

func TestLogTransformer_ReadFile(t *testing.T) {
	transformer := &LogTransformer{
		Logfile: "../data/cdk-diff1.log",
	}
	err := transformer.readFile()
	assert.NoError(t, err)
	transformer.removeAnsiCode()
	transformer.printFile()
}

type TestObject struct {
	input    string
	expected string
}

func TestLogTransformer_RemoveAnsiCode(t *testing.T) {
	cases := []TestObject{
		{
			input:    "\u001B[32m[+]\u001B[39m",
			expected: "[+]",
		},
		{
			input:    "\u001B[31mhelloworld\u001B[39m",
			expected: "helloworld",
		},
	}
	for _, c := range cases {
		logTransformer := &LogTransformer{
			LogContent: c.input,
		}
		logTransformer.removeAnsiCode()
		assert.Equal(t, c.expected, logTransformer.LogContent)
	}
}

func TestLogTransformer_TransformDiff(t *testing.T) {
	cases := []TestObject{
		{
			input:    "[+] line1 \n ---[+] line2",
			expected: "+[+] line1 \n+---[+] line2",
		},
		{
			input:    "[+] line1 \n --[[-[-] line2",
			expected: "+[+] line1 \n---[[-[-] line2",
		},
		{
			input:    "│ + │ ${SpmMainInitScript/ProviderHandler/ServiceRole.Arn}            │ Allow  │ sts:AssumeRole                │ Service:lambda.amazonaws.com                                    │           │",
			expected: "+ + │ ${SpmMainInitScript/ProviderHandler/ServiceRole.Arn}            │ Allow  │ sts:AssumeRole                │ Service:lambda.amazonaws.com                                    │           │",
		},
		{
			input:    "│ - │ ${SpmMainInitScript/ProviderHandler/ServiceRole.Arn}            │ Allow  │ sts:AssumeRole                │ Service:lambda.amazonaws.com                                    │           │",
			expected: "- - │ ${SpmMainInitScript/ProviderHandler/ServiceRole.Arn}            │ Allow  │ sts:AssumeRole                │ Service:lambda.amazonaws.com                                    │           │",
		},
		{
			input:    " │   ├─ [-] Removed: .query_cache_size",
			expected: "-│   ├─ [-] Removed: .query_cache_size",
		},
	}
	for _, c := range cases {
		logTransformer := NewLogTransformer(&config.NotifierConfig{})
		logTransformer.LogContent = c.input
		logTransformer.transformDiff()
		assert.Equal(t, c.expected, logTransformer.LogContent)
	}
}

type TemplateTest struct {
	transformer LogTransformer
	expected    string
	contains    string
}

func TestLogTransformer_AddHeader(t *testing.T) {
	cases := []TemplateTest{
		// empty VCS should not have collapsible section
		{
			transformer: LogTransformer{
				LogContent: "+[+] helloworld",
				TagID:      "some title",
			},
			expected: "\n## cdk diff for some title \n\n```diff\n+[+] helloworld\n```\n",
		},
		// bitbucket should not have collapisble section
		{
			transformer: LogTransformer{
				LogContent: "+[+] helloworld",
				TagID:      "some title",
			},
			expected: "\n## cdk diff for some title \n\n```diff\n+[+] helloworld\n```\n",
		},
		// github should have collapsible section
		{
			transformer: LogTransformer{
				LogContent: "+[+] helloworld",
				TagID:      "some github diff",
				Vcs:        "github",
			},
			expected: "\n## cdk diff for some github diff \n<details>\n<summary>Click to expand</summary>\n\n```diff\n+[+] helloworld\n```\n</details>\n",
		},
		// when using vcs github but setting disable-collapse it should have no collapsible section
		{
			transformer: LogTransformer{
				LogContent:      "+[+] helloworld",
				TagID:           "some github diff",
				Vcs:             "github",
				DisableCollapse: true,
			},
			expected: "\n## cdk diff for some github diff \n\n```diff\n+[+] helloworld\n```\n",
		},
	}
	for _, c := range cases {
		c.transformer.addHeader()
		assert.Equal(t, c.expected, c.transformer.LogContent)
	}
}

func TestLogTransformer_TransformDiffAddHeader(t *testing.T) {
	cases := []TemplateTest{
		//when displaying overview for number of replaces
		{
			transformer: LogTransformer{
				LogContent:      "[~] AWS::DynamoDB::Table ddb-table ddbtable7F3F6F3F replace\n └─ [~] TableName (requires replacement)\n-    ├─ [-] ddb-second-table\n+    └─ [+] ddb-second-table2",
				TagID:           "some github diff",
				Vcs:             "github",
				DisableCollapse: true,
				ShowOverview:    true,
			},
			contains: "⚠️ Number of resources that require replacement: 1",
		},
	}
	for _, c := range cases {
		c.transformer.initProcessorsChain()
		c.transformer.transformDiff()
		c.transformer.addHeader()
		assert.Contains(t, c.transformer.LogContent, c.contains)
	}
}

func TestLogTransformer_WriteDiffFile(t *testing.T) {
	file := "../data/cdk-diff1.log"
	fileDiff := "../data/cdk-diff1.log.diff"
	transformer := &LogTransformer{
		LogContent: "+[+] helloworld",
		Logfile:    file,
		TagID:      "small",
		NoPostMode: false,
	}

	defer os.Remove(fileDiff)

	err := transformer.writeDiffFile()
	assert.NoError(t, err)
	assert.NoFileExistsf(t, fileDiff, "Expect diff file not be found when no post mode not activated")

	transformer.NoPostMode = true
	err = transformer.writeDiffFile()
	assert.NoError(t, err)
	assert.FileExistsf(t, fileDiff, "Expect diff file to be found")

	transformer.Logfile = "/tmp/nonexisting-dir/nofile"
	err = transformer.writeDiffFile()
	assert.Error(t, err)
}

type TruncateTest struct {
	runeCount      int
	expectedLength int
	exceeds        bool
}

func randomStringRunes(n int) string {
	rand.Seed(time.Now().UnixNano())
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func TestLogTransform_Truncate(t *testing.T) {
	testCases := []TruncateTest{
		{
			runeCount:      0,
			expectedLength: 0,
			exceeds:        false,
		},
		{
			runeCount:      100,
			expectedLength: 100,
			exceeds:        false,
		},
		{
			runeCount:      65000,
			expectedLength: 65000,
			exceeds:        false,
		},
		{
			runeCount:      78999,
			expectedLength: 65000,
			exceeds:        true,
		},
		{
			runeCount:      878999,
			expectedLength: 65000,
			exceeds:        true,
		},
	}
	transformer := &LogTransformer{}

	for _, c := range testCases {
		truncatedLog := "\n...truncated"
		transformer.LogContent = randomStringRunes(c.runeCount)
		transformer.truncate()
		if c.exceeds {
			assert.Equal(t, c.expectedLength+len(truncatedLog), len(transformer.LogContent))
		} else {
			assert.Equal(t, c.expectedLength, len(transformer.LogContent))
		}
	}
}

func TestNewLogTransformer(t *testing.T) {
	c := &config.NotifierConfig{
		LogFile:    "../data/cdk-nochanges.log",
		TagID:      "small",
		NoPostMode: false,
	}
	transformer := NewLogTransformer(c)
	assert.NotNil(t, transformer)
	assert.Equal(t, transformer.LogContent, "")
	assert.Equal(t, transformer.TagID, "small")
	assert.Equal(t, transformer.NoPostMode, false)

	transformer.Process()
	assert.Contains(t, transformer.LogContent, "Stack SuiteRedisStack\nThere were no differences")

}
func TestOverviewSection(t *testing.T) {
	c := &config.NotifierConfig{
		LogFile:      "../data/cdk-diff-number-diff-replace.log",
		TagID:        "small",
		NoPostMode:   false,
		ShowOverview: true,
	}
	transformer := NewLogTransformer(c)
	assert.NotNil(t, transformer)
	assert.Equal(t, transformer.LogContent, "")
	assert.Equal(t, transformer.TagID, "small")
	assert.Equal(t, transformer.ShowOverview, true)

	transformer.Process()
	assert.Contains(t, transformer.NumberOfDifferencesString, "Number of stacks with differences: 1")
	assert.Equal(t, transformer.NumberReplaces, 5)

}
