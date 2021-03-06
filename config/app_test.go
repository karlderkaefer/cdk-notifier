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
			// not setting pull request id should not throw error
			input:    "",
			expected: 0,
			err:      nil,
		},
		{
			input:    "https://github.com/pansenentertainment/uepsilon/pull",
			expected: 0,
			err:      &strconv.NumError{},
		},
	}

	for _, c := range testCases {
		_ = os.Setenv(EnvPullRequestID, c.input)
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
				EnvPullRequestID: "23",
				EnvGithubToken:   "some-token",
				EnvRepoName:      "Uepsilon",
				EnvRepoOwner:     "pansenentertainment",
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
				EnvPullRequestID: "23",
				EnvGithubToken:   "some-token",
				EnvRepoName:      "Uepsilon",
				EnvRepoOwner:     "pansenentertainment",
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
				EnvPullRequestID: "23",
				EnvRepoName:      "Uepsilon",
				EnvRepoOwner:     "pansenentertainment",
			},
			expectedConfig: AppConfig{
				LogFile:       "./cdk.log",
				DeleteComment: true,
				RepoName:      "Uepsilon",
				RepoOwner:     "pansenentertainment",
				GithubToken:   "",
				PullRequest:   23,
			},
			err: &ValidationError{"github-token", EnvGithubToken},
		},
		{
			description: "test misssing pull request id will cause no error",
			inputConfig: AppConfig{
				LogFile:       "./cdk.log",
				DeleteComment: true,
			},
			envVars: map[string]string{
				EnvRepoName:    "Uepsilon",
				EnvRepoOwner:   "pansenentertainment",
				EnvGithubToken: "some-token",
			},
			expectedConfig: AppConfig{
				LogFile:       "./cdk.log",
				DeleteComment: true,
				RepoName:      "Uepsilon",
				RepoOwner:     "pansenentertainment",
				GithubToken:   "some-token",
				PullRequest:   0,
			},
			err: nil,
		},
		{
			description: "test parse int error",
			inputConfig: AppConfig{
				LogFile:     "./cdk.log",
				RepoName:    "Uepsilon",
				RepoOwner:   "pansenentertainment",
				GithubToken: "some-token",
			},
			envVars: map[string]string{
				EnvPullRequestID: "23as",
			},
			expectedConfig: AppConfig{
				LogFile:     "./cdk.log",
				RepoName:    "Uepsilon",
				RepoOwner:   "pansenentertainment",
				GithubToken: "some-token",
				PullRequest: 0,
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
