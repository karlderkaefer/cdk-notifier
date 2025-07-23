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
	tests := []struct {
		name     string
		inputUrl string
		want     *PullRequest
		wantErr  bool
	}{
		{
			name:     "Valid PR URL",
			inputUrl: "https://github.com/owner/repo/pull/123",
			want: &PullRequest{
				Host:   "github.com",
				Owner:  "owner",
				Repo:   "repo",
				Number: 123,
			},
			wantErr: false,
		},
		{
			name:     "Valid PR URL with custom host",
			inputUrl: "https://mycompany.com/owner/repo/pull/123",
			want: &PullRequest{
				Host:   "mycompany.com",
				Owner:  "owner",
				Repo:   "repo",
				Number: 123,
			},
			wantErr: false,
		},
		{
			name:     "Valid PR number",
			inputUrl: "9",
			want: &PullRequest{
				Host:   "",
				Owner:  "",
				Repo:   "",
				Number: 9,
			},
			wantErr: false,
		},
		{
			name:     "Invalid PR URL",
			inputUrl: "",
			want: &PullRequest{
				Number: 0,
			},
			wantErr: false,
		},
		{
			name:     "Invalid PR URL with null",
			inputUrl: "https://github.com/owner/repo/pull/null",
			want:     nil,
			wantErr:  true,
		},
		{
			name:     "Invalid URL Scheme",
			inputUrl: "::::",
			want:     nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := &PullRequest{}
			err := pr.ConvertUrlToPullRequest(tt.inputUrl)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConvertUrlToPullRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if *pr != *tt.want {
				t.Errorf("ConvertUrlToPullRequest() = %+v, want %+v", pr, tt.want)
			}
		})
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
				Vcs: 	  VcsGithub,
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
				Vcs: 	  VcsGithub,
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
				Vcs: 	  VcsGithub,
			},
			err: fmt.Errorf("unable to extract pull request number from url '%s': %w", "23as", &strconv.NumError{
				Func: "Atoi",
				Num:  "23as",
				Err:  strconv.ErrSyntax,
			}),
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
				Vcs: 	  VcsGithub,
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
				Vcs: 	  VcsGithub,
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
				Vcs: 	  VcsGithub,
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
				Vcs: 	  VcsGithub,
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
				Vcs: 	  VcsGithub,
			},
			err: &ValidationError{"owner", []string{"REPO_OWNER", EnvCiCircleCiRepoOwner, EnvCiBitbucketRepoOwner, EnvCiGitlabRepoOwner}},
		},
		{
			description: "test no post mode does not validate",
			inputConfig: NotifierConfig{
				LogFile:    "./cdk.log",
				TagID:      "no-post",
				NoPostMode: true,
			},
			envVars: map[string]string{},
			vcs:     VcsGithub,
			ci:      CiCircleCi,
			expectedConfig: NotifierConfig{
				LogFile:    "./cdk.log",
				TagID:      "no-post",
				NoPostMode: true,
				Ci:         CiCircleCi,
				Vcs: 	  VcsGithub,
			},
			err: nil,
		},
		{
			description: "test github enterprise host is set from circleci env",
			inputConfig: NotifierConfig{
				LogFile: "./cdk.log",
				RepoName:      "repo",
				RepoOwner:     "owner",
			},
			envVars: map[string]string{
				EnvCiCircleCiPullRequestID: "https://github.your-company.com/owner/repo/pull/1",
				EnvGithubToken:             "some-token",
			},
			ci:  CiCircleCi,
			vcs: VcsGithubEnterprise,
			expectedConfig: NotifierConfig{
				LogFile: "./cdk.log",
				GithubHost: "github.your-company.com",
				Ci: CiCircleCi,
				PullRequestID: 1,
				Vcs: VcsGithubEnterprise,
				RepoName:      "repo",
				RepoOwner:     "owner",
				Token:         "some-token",
			},
			err: nil,
		},
		{
			description: "test set template",
			inputConfig: NotifierConfig{
				LogFile: "./cdk.log",
				Template: "some template",
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
				Template: "some template",
				PullRequestID: 23,
				Ci:            CiCircleCi,
				Vcs: 	  VcsGithub,
			},
			err: nil,
		},
	}
	for _, c := range testCasesInit {
		t.Run(c.description, func(t *testing.T) {
			t.Log(c.description)
			for k, v := range c.envVars {
				fmt.Printf("set env variable %s: %s\n", k, v)
				_ = os.Setenv(k, v)
			}
			viper.Set("ci_system", c.ci)
			viper.Set("version_control_system", c.vcs)
			err := c.inputConfig.Init()
			assert.Equal(t, c.err, err, c.description)
			assert.Equal(t, c.expectedConfig, c.inputConfig, c.description)
			defer func() {
				for k := range c.envVars {
				_ = os.Unsetenv(k)
				}
			}()
		})
	}
}
