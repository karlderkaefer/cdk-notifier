package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// ValidationError indicated a missing configuration either CLI argument or environment variable
type ValidationError struct {
	CliArg string
	EnvVar []string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("missing argument. Set --%s argument or env var %s", e.CliArg, e.EnvVar)
}

const (
	// EnvGithubToken Name of environment variable for github token
	EnvGithubToken = "GITHUB_TOKEN"
	// EnvBitbucketToken Name of environment variable for bitbucket token
	EnvBitbucketToken = "BITBUCKET_TOKEN"
	// EnvGitlabToken Name of environment variable for Gitlab token
	EnvGitlabToken = "GITLAB_TOKEN"
	// EnvBitbucketUser Name of environment variable for bitbucket user
	EnvBitbucketUser = "BITBUCKET_USER"

	// EnvCiCircleCiPullRequestID Name of environment variable for pull request url
	EnvCiCircleCiPullRequestID = "CIRCLE_PULL_REQUEST"
	// EnvCiCircleCiRepoName Name of environment variable for GitHub repo
	EnvCiCircleCiRepoName = "CIRCLE_PROJECT_REPONAME"
	// EnvCiCircleCiRepoOwner Name of environment variable for GitHub owner
	EnvCiCircleCiRepoOwner = "CIRCLE_PROJECT_USERNAME"

	// EnvCiBitbucketPrId Bitbucket CI variable for pull request id - only available on pull request triggered builds
	EnvCiBitbucketPrId = "BITBUCKET_PR_ID"
	// EnvCiBitbucketRepoOwner Bitbucket CI variable for repo owner
	EnvCiBitbucketRepoOwner = "BITBUCKET_REPO_OWNER"
	// EnvCiBitbucketRepoName Bitbucket CI variable for repo name
	EnvCiBitbucketRepoName = "BITBUCKET_REPO_SLUG"

	// EnvCiGitlabMrId Name of environment variable for Gitlab merge request id
	EnvCiGitlabMrId = "CI_MERGE_REQUEST_IID"
	// EnvCiGitlabUrl Name of environment variable for Gitlab Base Url
	EnvCiGitlabUrl = "GITLAB_BASE_URL"
	// EnvCiGitlabRepoOwner Gitlab CI variable for repo owner
	EnvCiGitlabRepoOwner = "CI_PROJECT_NAMESPACE"
	// EnvCiGitlabRepoName Gitlab CI variable for repo name
	EnvCiGitlabRepoName = "CI_PROJECT_NAME"

	VcsGithub    = "github"
	VcsBitbucket = "bitbucket"
	VcsGitlab    = "gitlab"

	CiCircleCi  = "circleci"
	CiBitbucket = "bitbucket"
	CiGitlab    = "gitlab"
)

// NotifierConfig holds configuration
type NotifierConfig struct {
	LogFile       string `mapstructure:"LOG_FILE"`
	TagID         string `mapstructure:"TAG_ID"`
	RepoName      string `mapstructure:"REPO_NAME"`
	RepoOwner     string `mapstructure:"REPO_OWNER"`
	Token         string `mapstructure:"TOKEN"`
	TokenUser     string `mapstructure:"TOKEN_USER"`
	PullRequestID int    `mapstructure:"PR_ID"`
	DeleteComment bool   `mapstructure:"DELETE_COMMENT"`
	Vcs           string `mapstructure:"VERSION_CONTROL_SYSTEM"`
	Ci            string `mapstructure:"CI_SYSTEM"`
	Url           string `mapstructure:"URL"`
	NoPostMode    bool   `mapstructure:"NO_POST_MODE"`
}

// Init will create default NotifierConfig with following priority
// 1. Environment Variables GITHUB_TOKEN, CIRCLE_PULL_REQUEST, CIRCLE_PROJECT_REPONAME, CIRCLE_PROJECT_USERNAME
// 2. CLI args
// returns ValidationError if required field where not set
func (c *NotifierConfig) Init() error {
	err := c.loadViperConfig()
	if err != nil {
		logrus.Errorln(err)
		return err
	}
	err = c.validate()
	if err != nil {
		logrus.Errorln(err)
		return err
	}
	return nil
}

func (c *NotifierConfig) loadViperConfig() error {
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	for target, source := range createBindings() {
		err := viper.BindEnv(source, target)
		if err != nil {
			return err
		}
	}
	err := viper.Unmarshal(c)
	if err != nil {
		return err
	}
	return nil
}

// create binding to map individual CI environment variables to Config struct fields
func createBindings() map[string]string {
	ci := viper.GetString("ci_system")
	bindings := make(map[string]string)
	switch ci {
	case CiBitbucket:
		bindings[EnvCiBitbucketPrId] = "PR_ID"
		bindings[EnvCiBitbucketRepoName] = "REPO_NAME"
		bindings[EnvCiBitbucketRepoOwner] = "REPO_OWNER"
	case CiCircleCi:
		bindings[EnvCiCircleCiRepoName] = "REPO_NAME"
		bindings[EnvCiCircleCiRepoOwner] = "REPO_OWNER"
	case CiGitlab:
		bindings[EnvCiGitlabMrId] = "PR_ID"
		bindings[EnvCiGitlabRepoName] = "REPO_NAME"
		bindings[EnvCiGitlabRepoOwner] = "REPO_OWNER"
		bindings[EnvCiGitlabUrl] = "URL"
	default:
		logrus.Warnf("Could not detect CI environment from '%s'. Skipping override from CI Env vars", ci)
	}
	// mapping token environment vars regardless of environment since no conflicts expected
	bindings[EnvBitbucketUser] = "TOKEN_USER"
	bindings[EnvBitbucketToken] = "TOKEN"
	bindings[EnvGithubToken] = "TOKEN"
	bindings[EnvGitlabToken] = "TOKEN"
	return bindings
}

func (c *NotifierConfig) validate() error {
	ci := viper.GetString("ci_system")
	if c.PullRequestID == 0 && ci == CiCircleCi {
		prNumber, err := readPullRequestFromEnv()
		if err != nil {
			return err
		}
		c.PullRequestID = prNumber
	}
	if c.RepoName == "" {
		return &ValidationError{"repo", []string{"REPO_NAME", EnvCiCircleCiRepoName, EnvCiBitbucketRepoName, EnvCiGitlabRepoName}}
	}
	if c.RepoOwner == "" {
		return &ValidationError{"owner", []string{"REPO_OWNER", EnvCiCircleCiRepoOwner, EnvCiBitbucketRepoOwner, EnvCiGitlabRepoOwner}}
	}
	if c.Token == "" {
		return &ValidationError{"token", []string{"TOKEN", EnvGithubToken, EnvBitbucketToken, EnvGitlabToken}}
	}
	return nil
}

func readPullRequestFromEnv() (int, error) {
	url := os.Getenv(EnvCiCircleCiPullRequestID)
	if url == "" {
		logrus.Warnf("env var %s is not set or empty", EnvCiCircleCiPullRequestID)
		return 0, nil
	}
	elements := strings.Split(url, "/")
	prNumber := elements[len(elements)-1]
	val, err := strconv.ParseInt(prNumber, 10, 0)
	if err != nil {
		logrus.Errorf("Could not parse env %s with value '%v' to int", EnvCiCircleCiPullRequestID, url)
		return 0, err
	}
	if val != 0 {
		logrus.Debugf("Reading env var %s with value '%d'", EnvCiCircleCiPullRequestID, val)
	}
	return int(val), nil
}
