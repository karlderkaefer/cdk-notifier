package transform

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLogTransformer_ReadFile(t *testing.T) {
	transformer := &LogTransformer{
		Logfile: "../data/cdk-diff1.log",
	}
	err := transformer.ReadFile()
	assert.NoError(t, err)
	transformer.RemoveAnsiCode()
	transformer.PrintFile()
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
		logTransformer.RemoveAnsiCode()
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
			expected: "│ + │ ${SpmMainInitScript/ProviderHandler/ServiceRole.Arn}            │ Allow  │ sts:AssumeRole                │ Service:lambda.amazonaws.com                                    │           │",
		},
	}
	for _, c := range cases {
		logTransformer := &LogTransformer{
			LogContent: c.input,
		}
		logTransformer.TransformDiff()
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
				TagId:      "some title",
			},
			expected: "\n## cdk diff for some title \n```diff\n+[+] helloworld\n```\n",
		},
	}
	for _, c := range cases {
		c.transformer.AddHeader()
		assert.Equal(t, c.expected, c.transformer.LogContent)
	}
}
