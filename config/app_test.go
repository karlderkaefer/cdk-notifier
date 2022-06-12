package config

import (
	"fmt"
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
	inputConfig    NotifierConfig
	envVars        map[string]string
	expectedConfig NotifierConfig
	err            error
}

func TestNotifierConfig_ReadPullRequestFromEnv(t *testing.T) {
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
		_ = os.Setenv(EnvGithubPullRequestID, c.input)
		actual, err := readPullRequestFromEnv()
		assert.IsType(t, c.err, err)
		assert.Equal(t, c.expected, actual)
	}
}

func TestNotifierConfig_Init(t *testing.T) {
	testCasesInit := []testCaseInit{
		{
			description: "test set values by env variables",
			inputConfig: NotifierConfig{
				LogFile: "./cdk.log",
			},
			envVars: map[string]string{
				EnvGithubPullRequestID: "23",
				EnvGithubToken:         "some-token",
				EnvGithubRepoName:      "Uepsilon",
				EnvGithubRepoOwner:     "pansenentertainment",
			},
			expectedConfig: NotifierConfig{
				LogFile:       "./cdk.log",
				RepoName:      "Uepsilon",
				RepoOwner:     "pansenentertainment",
				Token:         "some-token",
				PullRequestID: 23,
			},
			err: nil,
		},
		{
			description: "test missing github token values by env variables",
			inputConfig: NotifierConfig{
				LogFile:       "./cdk.log",
				DeleteComment: true,
			},
			envVars: map[string]string{
				EnvGithubPullRequestID: "23",
				EnvGithubRepoName:      "Uepsilon",
				EnvGithubRepoOwner:     "pansenentertainment",
			},
			expectedConfig: NotifierConfig{
				LogFile:       "./cdk.log",
				DeleteComment: true,
				RepoName:      "Uepsilon",
				RepoOwner:     "pansenentertainment",
				Token:         "",
				PullRequestID: 23,
			},
			err: &ValidationError{"token", EnvGithubToken},
		},
		{
			description: "test misssing pull request id will cause no error",
			inputConfig: NotifierConfig{
				LogFile:       "./cdk.log",
				DeleteComment: true,
			},
			envVars: map[string]string{
				EnvGithubRepoName:  "Uepsilon",
				EnvGithubRepoOwner: "pansenentertainment",
				EnvGithubToken:     "some-token",
			},
			expectedConfig: NotifierConfig{
				LogFile:       "./cdk.log",
				DeleteComment: true,
				RepoName:      "Uepsilon",
				RepoOwner:     "pansenentertainment",
				Token:         "some-token",
				PullRequestID: 0,
			},
			err: nil,
		},
		{
			description: "test parse int error",
			inputConfig: NotifierConfig{
				LogFile:   "./cdk.log",
				RepoName:  "Uepsilon",
				RepoOwner: "pansenentertainment",
				Token:     "some-token",
			},
			envVars: map[string]string{
				EnvGithubPullRequestID: "23as",
			},
			expectedConfig: NotifierConfig{
				LogFile:       "./cdk.log",
				RepoName:      "Uepsilon",
				RepoOwner:     "pansenentertainment",
				Token:         "some-token",
				PullRequestID: 0,
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
			fmt.Printf("set env variable %s: %s\n", k, v)
			_ = os.Setenv(k, v)
		}
		err := c.inputConfig.Init()
		assert.Equal(t, c.err, err, c.description)
		assert.Equal(t, c.expectedConfig, c.inputConfig, c.description)
		for k := range c.envVars {
			_ = os.Unsetenv(k)
		}
	}
}
