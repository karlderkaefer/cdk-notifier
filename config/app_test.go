package config

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
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
	ci             string
	vcs            string
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
			input:    "https://git.something.company.com/future/project/pull/9",
			expected: 9,
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
		_ = os.Setenv(EnvCiCircleCiPullRequestID, c.input)
		p := PullRequest{}
		err := p.LoadFromURL()
		actual := p.Number
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
			vcs: VcsGithub,
			ci:  CiCircleCi,
			envVars: map[string]string{
				EnvCiCircleCiPullRequestID: "23",
				EnvGithubToken:             "some-token",
				EnvCiCircleCiRepoName:      "Uepsilon",
				EnvCiCircleCiRepoOwner:     "pansenentertainment",
			},
			expectedConfig: NotifierConfig{
				LogFile:       "./cdk.log",
				RepoName:      "Uepsilon",
				RepoOwner:     "pansenentertainment",
				Token:         "some-token",
				PullRequestID: 23,
				Ci:            CiCircleCi,
			},
			err: nil,
		},
		{
			description: "test misssing pull request id will cause no error",
			inputConfig: NotifierConfig{
				LogFile:       "./cdk.log",
				DeleteComment: true,
			},
			envVars: map[string]string{
				EnvCiCircleCiRepoName:  "Uepsilon",
				EnvCiCircleCiRepoOwner: "pansenentertainment",
				EnvGithubToken:         "some-token",
			},
			vcs: VcsGithub,
			ci:  CiCircleCi,
			expectedConfig: NotifierConfig{
				LogFile:       "./cdk.log",
				DeleteComment: true,
				RepoName:      "Uepsilon",
				RepoOwner:     "pansenentertainment",
				Token:         "some-token",
				PullRequestID: 0,
				Ci:            CiCircleCi,
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
				Ci:        CiCircleCi,
			},
			envVars: map[string]string{
				EnvCiCircleCiPullRequestID: "23as",
			},
			vcs: VcsGithub,
			ci:  CiCircleCi,
			expectedConfig: NotifierConfig{
				LogFile:       "./cdk.log",
				RepoName:      "Uepsilon",
				RepoOwner:     "pansenentertainment",
				Token:         "some-token",
				PullRequestID: 0,
				Ci:            CiCircleCi,
			},
			err: &strconv.NumError{
				Func: "Atoi",
				Num:  "23as",
				Err:  strconv.ErrSyntax,
			},
		},
		{
			description: "test bitbucket ci config override",
			inputConfig: NotifierConfig{
				LogFile:   "./cdk.log",
				RepoName:  "Uepsilon",
				RepoOwner: "pansenentertainment",
				Token:     "some-token",
				Ci:        CiCircleCi,
			},
			envVars: map[string]string{
				EnvCiBitbucketRepoOwner: "pansenentertainment",
				EnvCiBitbucketPrId:      "12",
				EnvCiBitbucketRepoName:  "Uepsilon",
				EnvBitbucketToken:       "bitbucket-token",
			},
			vcs: VcsGithub,
			ci:  CiBitbucket,
			expectedConfig: NotifierConfig{
				LogFile:       "./cdk.log",
				RepoName:      "Uepsilon",
				RepoOwner:     "pansenentertainment",
				Token:         "bitbucket-token",
				PullRequestID: 12,
				Ci:            CiBitbucket,
			},
			err: nil,
		},
		{
			description: "test missing CI override",
			inputConfig: NotifierConfig{
				LogFile:   "./cdk.log",
				RepoName:  "Uepsilon",
				RepoOwner: "pansenentertainment",
				Token:     "some-token",
			},
			envVars: map[string]string{
				EnvCiBitbucketRepoOwner: "pansenentertainment",
				EnvCiBitbucketPrId:      "12",
				EnvCiBitbucketRepoName:  "Uepsilon",
				EnvBitbucketToken:       "bitbucket-token",
			},
			vcs: VcsGithub,
			ci:  "",
			expectedConfig: NotifierConfig{
				LogFile:       "./cdk.log",
				RepoName:      "Uepsilon",
				RepoOwner:     "pansenentertainment",
				Token:         "bitbucket-token",
				PullRequestID: 12,
				Ci:            "",
			},
			err: nil,
		},
		{
			description: "test missing parameter github token",
			inputConfig: NotifierConfig{
				LogFile:       "./cdk.log",
				DeleteComment: true,
			},
			envVars: map[string]string{
				EnvCiCircleCiPullRequestID: "23",
				EnvCiCircleCiRepoName:      "Uepsilon",
				EnvCiCircleCiRepoOwner:     "pansenentertainment",
			},
			vcs: VcsGithub,
			ci:  CiCircleCi,
			expectedConfig: NotifierConfig{
				LogFile:       "./cdk.log",
				DeleteComment: true,
				RepoName:      "Uepsilon",
				RepoOwner:     "pansenentertainment",
				Token:         "",
				PullRequestID: 23,
				Ci:            CiCircleCi,
			},
			err: &ValidationError{"token", []string{"TOKEN", EnvGithubToken, EnvBitbucketToken, EnvGitlabToken}},
		},
		{
			description: "test missing parameter repo name",
			inputConfig: NotifierConfig{
				LogFile:       "./cdk.log",
				DeleteComment: true,
			},
			envVars: map[string]string{
				EnvCiCircleCiPullRequestID: "23",
				EnvGithubToken:             "some-token",
				EnvCiCircleCiRepoOwner:     "pansenentertainment",
			},
			vcs: VcsGithub,
			ci:  CiCircleCi,
			expectedConfig: NotifierConfig{
				LogFile:       "./cdk.log",
				DeleteComment: true,
				RepoName:      "",
				RepoOwner:     "pansenentertainment",
				Token:         "some-token",
				PullRequestID: 23,
				Ci:            CiCircleCi,
			},
			err: &ValidationError{"repo", []string{"REPO_NAME", EnvCiCircleCiRepoName, EnvCiBitbucketRepoName, EnvCiGitlabRepoName}},
		},
		{
			description: "test missing parameter repo owner",
			inputConfig: NotifierConfig{
				LogFile:       "./cdk.log",
				DeleteComment: true,
			},
			envVars: map[string]string{
				EnvCiCircleCiPullRequestID: "23",
				EnvGithubToken:             "some-token",
				EnvCiCircleCiRepoName:      "Uepsilon",
			},
			vcs: VcsGithub,
			ci:  CiCircleCi,
			expectedConfig: NotifierConfig{
				LogFile:       "./cdk.log",
				DeleteComment: true,
				RepoName:      "Uepsilon",
				RepoOwner:     "",
				Token:         "some-token",
				PullRequestID: 23,
				Ci:            CiCircleCi,
			},
			err: &ValidationError{"owner", []string{"REPO_OWNER", EnvCiCircleCiRepoOwner, EnvCiBitbucketRepoOwner, EnvCiGitlabRepoOwner}},
		},
	}
	for _, c := range testCasesInit {
		t.Log(c.description)
		for k, v := range c.envVars {
			fmt.Printf("set env variable %s: %s\n", k, v)
			_ = os.Setenv(k, v)
		}
		viper.Set("ci_system", c.ci)
		viper.Set("vcs", c.vcs)
		err := c.inputConfig.Init()
		assert.Equal(t, c.err, err, c.description)
		assert.Equal(t, c.expectedConfig, c.inputConfig, c.description)
		for k := range c.envVars {
			_ = os.Unsetenv(k)
		}
	}
}
