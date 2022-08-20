package transform

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
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
		logTransformer := &LogTransformer{
			LogContent: c.input,
		}
		logTransformer.transformDiff()
		assert.Equal(t, c.expected, logTransformer.LogContent)
	}
}

type TemplateTest struct {
	transformer LogTransformer
	expected    string
}

func TestLogTransformer_AddHeader(t *testing.T) {
	cases := []TemplateTest{
		{
			transformer: LogTransformer{
				LogContent: "+[+] helloworld",
				TagID:      "some title",
			},
			expected: "\n## cdk diff for some title \n```diff\n+[+] helloworld\n```\n",
		},
	}
	for _, c := range cases {
		c.transformer.addHeader()
		assert.Equal(t, c.expected, c.transformer.LogContent)
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

}
