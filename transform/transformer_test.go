package transform

import (
	"math/rand"
	"os"
	"reflect"
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

	// do not add job link
	os.Setenv("CDK_NOTIFIER_DEACTIVATE_JOB_LINK", "true")
	defer os.Unsetenv("CDK_NOTIFIER_DEACTIVATE_JOB_LINK")

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
	provider       string
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
			provider:       config.VcsGithub,
			exceeds:        false,
		},
		{
			runeCount:      100,
			expectedLength: 100,
			provider:       config.VcsGithub,
			exceeds:        false,
		},
		{
			runeCount:      65000,
			expectedLength: 65000,
			provider:       config.VcsGithub,
			exceeds:        false,
		},
		{
			runeCount:      78999,
			expectedLength: 65536,
			provider:       config.VcsGithub,
			exceeds:        true,
		},
		{
			runeCount:      878999,
			expectedLength: 65536,
			provider:       config.VcsGithub,
			exceeds:        true,
		},
		{
			runeCount:      0,
			expectedLength: 0,
			provider:       config.VcsGitlab,
			exceeds:        false,
		},
		{
			runeCount:      100,
			expectedLength: 100,
			provider:       config.VcsGitlab,
			exceeds:        false,
		},
		{
			runeCount:      65000,
			expectedLength: 65000,
			provider:       config.VcsGitlab,
			exceeds:        false,
		},
		{
			runeCount:      78999,
			expectedLength: 78999,
			provider:       config.VcsGitlab,
			exceeds:        false,
		},
		{
			runeCount:      878999,
			expectedLength: 878999,
			provider:       config.VcsGitlab,
			exceeds:        false,
		},
		{
			runeCount:      1000001,
			expectedLength: 1000000,
			provider:       config.VcsGitlab,
			exceeds:        true,
		},
		{
			runeCount:      423428,
			expectedLength: 32768,
			provider:       config.VcsBitbucket,
			exceeds:        true,
		},
		{
			runeCount:      100,
			expectedLength: 100,
			provider:       config.VcsBitbucket,
			exceeds:        false,
		},
		{
			runeCount:      32769,
			expectedLength: 32768,
			provider:       config.VcsBitbucket,
			exceeds:        true,
		},
		{
			runeCount:      80001,
			expectedLength: 80000,
			provider:       config.VcsGithubEnterprise,
			exceeds:        true,
		},
	}

	for _, c := range testCases {
		transformer := &LogTransformer{
			Vcs: c.provider,
		}
		if c.provider == config.VcsGithubEnterprise {
			transformer.GithubMaxCommentLength = 80000
		}
		transformer.LogContent = randomStringRunes(c.runeCount)
		transformer.truncate()
		assert.Equal(t, c.expectedLength, len(transformer.LogContent))
		if c.exceeds {
			assert.Contains(t, transformer.LogContent, "**Warning**")
		} else {
			assert.NotContains(t, transformer.LogContent, "**Warning**")
		}
	}
}

func TestNewLogTransformer(t *testing.T) {
	c := &config.NotifierConfig{
		LogFile:    "../data/cdk-nochanges.log",
		TagID:      "small",
		NoPostMode: false,
		Vcs:        config.VcsGithub,
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
		Vcs:          config.VcsGithub,
	}
	transformer := NewLogTransformer(c)
	assert.NotNil(t, transformer)
	assert.Equal(t, transformer.LogContent, "")
	assert.Equal(t, transformer.TagID, "small")
	assert.Equal(t, transformer.ShowOverview, true)

	transformer.Process()
	assert.Contains(t, transformer.NumberOfDifferencesString, "Number of stacks with differences: 1")
	assert.Equal(t, transformer.NumberReplaces, 5)

	// test resource diff extractor
	res := transformer.ChangedBaseResource
	assert.Equal(t, 2, res["AWS::DynamoDB::Table"].Count)
	assert.Equal(t, true, res["AWS::DynamoDB::Table"].Replaced)
	assert.Equal(t, 1, res["AWS::RDS::DBInstance"].Count)
	assert.Equal(t, true, res["AWS::RDS::DBInstance"].Replaced)
	assert.Equal(t, 1, res["AWS::RDS::DBInstance"].Count)
	assert.Equal(t, true, res["AWS::RDS::DBInstance"].Replaced)
	assert.Equal(t, 1, res["AWS::RDS::DBParameterGroup"].Count)
	assert.Equal(t, true, res["AWS::RDS::DBParameterGroup"].Replaced)
}

func TestResourceDiffExtractorProcessor_ProcessLine(t *testing.T) {
	tests := []struct {
		name          string
		line          string
		expected      map[string]ResourceMetric
		baseProcessor BaseProcessor
	}{
		{
			name: "WithAddition",
			line: "   [+] AWS::S3::Bucket MyBucket",
			expected: map[string]ResourceMetric{
				"AWS::S3::Bucket": {
					Count:    1,
					Replaced: false,
				},
			},
			baseProcessor: BaseProcessor{},
		},
		{
			name: "WithDeletion",
			line: "   [-] AWS::S3::Bucket MyBucket",
			expected: map[string]ResourceMetric{
				"AWS::S3::Bucket": {
					Count:    1,
					Replaced: false,
				},
			},
			baseProcessor: BaseProcessor{},
		},
		{
			name: "WithModification",
			line: "   [~] AWS::S3::Bucket MyBucket replaced",
			expected: map[string]ResourceMetric{
				"AWS::S3::Bucket": {
					Count:    1,
					Replaced: true,
				},
			},
			baseProcessor: BaseProcessor{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lt := &LogTransformer{
				ChangedBaseResource: make(map[string]ResourceMetric),
			}
			p := &ResourceDiffExtractorProcessor{
				BaseProcessor: tt.baseProcessor,
			}

			p.ProcessLine(tt.line, lt)
			if !reflect.DeepEqual(lt.ChangedBaseResource, tt.expected) {
				t.Errorf("ChangedBaseResource = %v, want %v", lt.ChangedBaseResource, tt.expected)
			}
		})
	}
}
func TestGetJobLink(t *testing.T) {
	// Set up test cases
	os.Setenv("CDK_NOTIFIER_DEACTIVATE_JOB_LINK", "false")
	os.Setenv("CIRCLECI", "false")
	cases := []struct {
		name     string
		envVars  map[string]string
		expected string
	}{
		{
			name: "cirlceci job link",
			envVars: map[string]string{
				"CIRCLECI":         "true",
				"CIRCLE_BUILD_URL": "https://circleci.com/build/456",
			},
			expected: "https://circleci.com/build/456",
		},
		{
			name: "gitlab job link",
			envVars: map[string]string{
				"GITLAB_CI":  "true",
				"CI_JOB_URL": "https://gitlab.com/job/123",
			},
			expected: "https://gitlab.com/job/123",
		},
		{
			name: "bitbucket job link",
			envVars: map[string]string{
				"BITBUCKET_BUILD_NUMBER": "789",
				"BITBUCKET_WORKSPACE":    "workspace",
				"BITBUCKET_REPO_SLUG":    "repo",
			},
			expected: "https://bitbucket.org/workspace/repo/pipelines/results/789",
		},
		{
			name: "github job link",
			envVars: map[string]string{
				"GITHUB_ACTIONS":    "true",
				"GITHUB_SERVER_URL": "https://github.com",
				"GITHUB_REPOSITORY": "owner/repo",
				"GITHUB_RUN_ID":     "12345",
			},
			expected: "https://github.com/owner/repo/actions/runs/12345",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			// Save the current environment
			oldEnv := make(map[string]string)
			for k, v := range tt.envVars {
				oldEnv[k] = os.Getenv(k)
				os.Setenv(k, v)
			}

			// Reset the environment after the test
			t.Cleanup(func() {
				for k, v := range oldEnv {
					os.Setenv(k, v)
				}
			})

			if got := getJobLink(); got != tt.expected {
				t.Errorf("getJobLink() = %v, want %v", got, tt.expected)
			}
		})
	}
}
func TestIgnoreHashesProcessor_ProcessLine(t *testing.T) {
	cases := []struct {
		line           string
		expectedTotal  int
		expectedHashes int
	}{
		{
			line:           "+ some line",
			expectedTotal:  1,
			expectedHashes: 0,
		},
		{
			line:           "- some line",
			expectedTotal:  1,
			expectedHashes: 0,
		},
		{
			line:           "+ some line with hash abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			expectedTotal:  1,
			expectedHashes: 1,
		},
		{
			line:           "- some line with hash abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			expectedTotal:  1,
			expectedHashes: 1,
		},
		{
			line:           "+ some line with invalid hash",
			expectedTotal:  1,
			expectedHashes: 0,
		},
		{
			line:           "- some line with invalid hash",
			expectedTotal:  1,
			expectedHashes: 0,
		},
		{
			line:           "some other line",
			expectedTotal:  0,
			expectedHashes: 0,
		},
		{
			line:           "-       [-]   \"Fn::Sub\": \"123456789012.dkr.ecr.eu-central-1.${AWS::URLSuffix}/cdk-hnb659fds-container-assets-123456789012-eu-central-1:88f53e8e790ee348fe371bfe2dd7365d2cc15be096da0c12d4b0d8bf47aff35d3",
			expectedTotal:  1,
			expectedHashes: 1,
		},
	}

	processor := &IgnoreHashesProcessor{
		BaseProcessor: BaseProcessor{},
	}

	lt := &LogTransformer{
		TotalChanges: 0,
		HashChanges:  0,
	}

	for _, c := range cases {
		lt.TotalChanges = 0
		lt.HashChanges = 0
		processor.ProcessLine(c.line, lt)
		assert.Equal(t, c.expectedTotal, lt.TotalChanges)
		assert.Equal(t, c.expectedHashes, lt.HashChanges)
	}
}
