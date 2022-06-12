package config

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
	"strconv"
	"strings"
)

// ValidationError indicated a missing configuration either CLI argument or environment variable
type ValidationError struct {
	CliArg string
	EnvVar string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("missing argument. Set --%s argument or env var %s", e.CliArg, e.EnvVar)
}

const (
	// EnvGithubToken Name of environment variable for github token
	EnvGithubToken = "GITHUB_TOKEN"
	// EnvGithubPullRequestID Name of environment variable for pull request url
	EnvGithubPullRequestID = "CIRCLE_PULL_REQUEST"
	// EnvGithubRepoName Name of environment variable for GitHub repo
	EnvGithubRepoName = "CIRCLE_PROJECT_REPONAME"
	// EnvGithubRepoOwner Name of environment variable for GitHub owner
	EnvGithubRepoOwner = "CIRCLE_PROJECT_USERNAME"
	// EnvBitbucketToken Name of environment variable for bitbucket token
	EnvBitbucketToken = "BITBUCKET_TOKEN"
	// EnvBitbucketUser Name of environment variable for bitbucket user
	EnvBitbucketUser = "BITBUCKET_USER"
	// EnvBitbucketPrId ID of Pull Request - only available on pull request triggered builds
	EnvBitbucketPrId = "BITBUCKET_PR_ID"
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
	bindings := make(map[string]string)
	// CircleCi
	bindings[EnvGithubRepoName] = "REPO_NAME"
	bindings[EnvGithubRepoOwner] = "REPO_OWNER"
	bindings[EnvGithubToken] = "TOKEN"
	// Bitbucket
	bindings[EnvBitbucketToken] = "TOKEN"
	bindings[EnvBitbucketUser] = "TOKEN_USER"
	bindings[EnvBitbucketPrId] = "PR_ID"
	return bindings
}

func (c *NotifierConfig) validate() error {
	if c.PullRequestID == 0 {
		prNumber, err := readPullRequestFromEnv()
		if err != nil {
			return err
		}
		c.PullRequestID = prNumber
	}
	if c.RepoName == "" {
		return &ValidationError{"repo", EnvGithubRepoName}
	}
	if c.RepoOwner == "" {
		return &ValidationError{"owner", EnvGithubRepoOwner}
	}
	if c.Token == "" {
		return &ValidationError{"token", EnvGithubToken}
	}
	return nil
}

func readPullRequestFromEnv() (int, error) {
	url := os.Getenv(EnvGithubPullRequestID)
	if url == "" {
		logrus.Warnf("env var %s is not set or empty", EnvGithubPullRequestID)
		return 0, nil
	}
	elements := strings.Split(url, "/")
	prNumber := elements[len(elements)-1]
	val, err := strconv.ParseInt(prNumber, 10, 0)
	if err != nil {
		logrus.Errorf("Could not parse env %s with value '%v' to int", EnvGithubPullRequestID, url)
		return 0, err
	}
	if val != 0 {
		logrus.Debugf("Reading env var %s with value '%d'", EnvGithubPullRequestID, val)
	}
	return int(val), nil
}
