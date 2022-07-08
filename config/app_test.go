package config

import (
	"os"
	"strconv"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

type testCase struct {
	input    string
	expected int
	err      error
}

type testCaseInit struct {
	description    string
	inputConfig    AppConfig
	envVars        map[string]string
	expectedConfig AppConfig
	err            error
}

func TestAppConfig_ReadPullRequestFromEnv(t *testing.T) {
	logrus.SetLevel(7)
	testCases := []testCase{
		{
			input:    "https://gitlab.com/svause/somerepo/-/merge_requests/1361",
			expected: 1361,
			err:      nil,
		},
		{
			input:    "/1361",
			expected: 1361,
			err:      nil,
		},
		{
			input:    "1361",
			expected: 1361,
			err:      nil,
		},
		{
			// not setting pull request id should not throw error
			input:    "",
			expected: 0,
			err:      nil,
		},
		{
			input:    "https://gitlab.com/svause/somerepo/-/merge_requests/",
			expected: 0,
			err:      &strconv.NumError{},
		},
	}

	for _, c := range testCases {
		_ = os.Setenv(EnvMergeRequestID, c.input)
		actual, err := readMergeRequestFromEnv()
		assert.IsType(t, c.err, err)
		assert.Equal(t, c.expected, actual)
	}
}

func TestAppConfig_Init(t *testing.T) {
	testCasesInit := []testCaseInit{
		{
			description: "test set values by env variables",
			inputConfig: AppConfig{
				LogFile: "./cdk.log",
			},
			envVars: map[string]string{
				EnvMergeRequestID: "23",
				EnvGitlabToken:    "some-token",
				EnvGitlabPid:      "1",
				EnvGitlabUrl:      "https://gitlab.com/",
			},
			expectedConfig: AppConfig{
				LogFile:      "./cdk.log",
				ProjectID:    1,
				GitlabToken:  "some-token",
				MergeRequest: 23,
				GitlabUrl:    "https://gitlab.com/",
			},
			err: nil,
		},
		{
			description: "test override of env vars with cli arguments",
			inputConfig: AppConfig{
				LogFile:      "./cdk.log",
				GitlabToken:  "changedToken",
				MergeRequest: 12,
				ProjectID:    2,
			},
			envVars: map[string]string{
				EnvMergeRequestID: "23",
				EnvGitlabToken:    "some-token",
				EnvGitlabPid:      "1",
				EnvGitlabUrl:      "https://gitlab.com/",
			},
			expectedConfig: AppConfig{
				LogFile:      "./cdk.log",
				GitlabToken:  "changedToken",
				MergeRequest: 12,
				ProjectID:    2,
				GitlabUrl:    "https://gitlab.com/",
			},
			err: nil,
		},
		{
			description: "test missing gitlab token values by env variables",
			inputConfig: AppConfig{
				LogFile:    "./cdk.log",
				DeleteNote: true,
			},
			envVars: map[string]string{
				EnvMergeRequestID: "23",
				EnvGitlabPid:      "1",
				EnvGitlabUrl:      "https://gitlab.com/",
			},
			expectedConfig: AppConfig{
				LogFile:      "./cdk.log",
				DeleteNote:   true,
				ProjectID:    1,
				GitlabToken:  "",
				MergeRequest: 23,
				GitlabUrl:    "https://gitlab.com/",
			},
			err: &ValidationError{"gitlab-token", EnvGitlabToken},
		},
		{
			description: "test misssing merge request id will cause no error",
			inputConfig: AppConfig{
				LogFile:    "./cdk.log",
				DeleteNote: true,
			},
			envVars: map[string]string{
				EnvGitlabPid:   "1",
				EnvGitlabToken: "some-token",
				EnvGitlabUrl:   "https://gitlab.com/",
			},
			expectedConfig: AppConfig{
				LogFile:      "./cdk.log",
				DeleteNote:   true,
				ProjectID:    1,
				GitlabToken:  "some-token",
				MergeRequest: 0,
				GitlabUrl:    "https://gitlab.com/",
			},
			err: nil,
		},
		{
			description: "test parse int error",
			inputConfig: AppConfig{
				LogFile:     "./cdk.log",
				ProjectID:   1,
				GitlabToken: "some-token",
			},
			envVars: map[string]string{
				EnvMergeRequestID: "23as",
				EnvGitlabUrl:      "https://gitlab.com/",
			},
			expectedConfig: AppConfig{
				LogFile:      "./cdk.log",
				ProjectID:    1,
				GitlabToken:  "some-token",
				MergeRequest: 0,
				GitlabUrl:    "https://gitlab.com/",
			},
			err: &strconv.NumError{
				Func: "ParseInt",
				Num:  "23as",
				Err:  strconv.ErrSyntax,
			},
		},
	}
	for _, c := range testCasesInit {
		t.Log(c.description)
		for k, v := range c.envVars {
			_ = os.Setenv(k, v)
		}
		err := c.inputConfig.Init()
		assert.Equal(t, c.err, err)
		assert.Equal(t, c.expectedConfig, c.inputConfig)
		for k := range c.envVars {
			_ = os.Unsetenv(k)
		}
	}
}
