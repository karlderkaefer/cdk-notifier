package config

import (
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"os"
	"strconv"
	"testing"
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
			input:    "https://github.com/pansenentertainment/uepsilon/pull/1361",
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
			input:    "",
			expected: 0,
			err:      &strconv.NumError{},
		},
		{
			input:    "https://github.com/pansenentertainment/uepsilon/pull",
			expected: 0,
			err:      &strconv.NumError{},
		},
	}

	for _, c := range testCases {
		_ = os.Setenv(ENV_PULL_REQUEST_ID, c.input)
		actual, err := readPullRequestFromEnv()
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
				ENV_PULL_REQUEST_ID: "23",
				ENV_GITHUB_TOKEN:    "some-token",
				ENV_REPO_NAME:       "Uepsilon",
				ENV_REPO_OWNER:      "pansenentertainment",
			},
			expectedConfig: AppConfig{
				LogFile:     "./cdk.log",
				RepoName:    "Uepsilon",
				RepoOwner:   "pansenentertainment",
				GithubToken: "some-token",
				PullRequest: 23,
			},
			err: nil,
		},
		{
			description: "test override of env vars with cli arguments",
			inputConfig: AppConfig{
				LogFile:     "./cdk.log",
				RepoName:    "changedRepo",
				GithubToken: "changedToken",
				RepoOwner:   "changedOwner",
				PullRequest: 12,
			},
			envVars: map[string]string{
				ENV_PULL_REQUEST_ID: "23",
				ENV_GITHUB_TOKEN:    "some-token",
				ENV_REPO_NAME:       "Uepsilon",
				ENV_REPO_OWNER:      "pansenentertainment",
			},
			expectedConfig: AppConfig{
				LogFile:     "./cdk.log",
				RepoName:    "changedRepo",
				GithubToken: "changedToken",
				RepoOwner:   "changedOwner",
				PullRequest: 12,
			},
			err: nil,
		},
		{
			description: "test missing github token values by env variables",
			inputConfig: AppConfig{
				LogFile:       "./cdk.log",
				DeleteComment: true,
			},
			envVars: map[string]string{
				ENV_PULL_REQUEST_ID: "23",
				ENV_REPO_NAME:       "Uepsilon",
				ENV_REPO_OWNER:      "pansenentertainment",
			},
			expectedConfig: AppConfig{
				LogFile:       "./cdk.log",
				DeleteComment: true,
				RepoName:      "Uepsilon",
				RepoOwner:     "pansenentertainment",
				GithubToken:   "",
				PullRequest:   23,
			},
			err: &ConfigValidationError{"github-token", ENV_GITHUB_TOKEN},
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
		for k, _ := range c.envVars {
			_ = os.Unsetenv(k)
		}
	}
}
