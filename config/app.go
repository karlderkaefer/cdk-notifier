package config

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"strconv"
	"strings"
)

// AppConfig application configuration initialized by cobra arguments or environment variabels
type AppConfig struct {
	LogFile       string
	TagID         string
	RepoName      string
	RepoOwner     string
	GithubToken   string
	PullRequest   int
	DeleteComment bool
}

const (
	// HeaderPrefix default prefix for comment message
	HeaderPrefix = "## cdk diff for"
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
	// EnvPullRequestID Name of environment variable for pull request url
	EnvPullRequestID = "CIRCLE_PULL_REQUEST"
	// EnvRepoName Name of environment variable for GitHub repo
	EnvRepoName = "CIRCLE_PROJECT_REPONAME"
	// EnvRepoOwner Name of environment variable for GitHub owner
	EnvRepoOwner = "CIRCLE_PROJECT_USERNAME"
)

// Init will create default AppConfig with following priority
// 1. Environment Variables GITHUB_TOKEN, CIRCLE_PULL_REQUEST, CIRCLE_PROJECT_REPONAME, CIRCLE_PROJECT_USERNAME
// 2. CLI args
// returns ValidationError if required field where not set
func (a *AppConfig) Init() error {
	if a.RepoName == "" {
		a.RepoName = readFromEnv(EnvRepoName)
	}
	if a.RepoOwner == "" {
		a.RepoOwner = readFromEnv(EnvRepoOwner)
	}
	if a.GithubToken == "" {
		a.GithubToken = readFromEnv(EnvGithubToken)
	}
	if a.PullRequest == 0 {
		prNumber, err := readPullRequestFromEnv()
		if err != nil {
			return err
		}
		a.PullRequest = prNumber
	}
	// validate
	if a.RepoName == "" {
		return &ValidationError{"github-repo", EnvRepoName}
	}
	if a.RepoOwner == "" {
		return &ValidationError{"github-owner", EnvRepoOwner}
	}
	if a.GithubToken == "" {
		return &ValidationError{"github-token", EnvGithubToken}
	}
	return nil
}

func readFromEnv(env string) string {
	val := os.Getenv(env)
	if val != "" {
		logrus.Debugf("Reading env var %s with value '%s'", env, val)
	}
	return val
}

func readPullRequestFromEnv() (int, error) {
	url := os.Getenv(EnvPullRequestID)
	if url == "" {
		logrus.Warnf("env var %s is not set or empty", EnvPullRequestID)
		return 0, nil
	}
	elements := strings.Split(url, "/")
	prNumber := elements[len(elements)-1]
	val, err := strconv.ParseInt(prNumber, 10, 0)
	if err != nil {
		logrus.Errorf("Could not parse env %s with value '%v' to int", EnvPullRequestID, url)
		return 0, err
	}
	if val != 0 {
		logrus.Debugf("Reading env var %s with value '%d'", EnvPullRequestID, val)
	}
	return int(val), nil
}
