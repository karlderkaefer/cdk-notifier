package config

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"strconv"
	"strings"
)

type AppConfig struct {
	LogFile       string
	TagId         string
	RepoName      string
	RepoOwner     string
	GithubToken   string
	PullRequest   int
	DeleteComment bool
}

type ConfigValidationError struct {
	cliArg string
	envVar string
}

func (e *ConfigValidationError) Error() string {
	return fmt.Sprintf("missing argument. Set --%s argument or env var %s", e.cliArg, e.envVar)
}

const (
	ENV_GITHUB_TOKEN    = "GITHUB_TOKEN"
	ENV_PULL_REQUEST_ID = "CIRCLE_PULL_REQUEST"
	ENV_REPO_NAME       = "CIRCLE_PROJECT_REPONAME"
	ENV_REPO_OWNER      = "CIRCLE_PROJECT_USERNAME"
)

func (a *AppConfig) Init() error {
	if a.RepoName == "" {
		a.RepoName = readFromEnv(ENV_REPO_NAME)
	}
	if a.RepoOwner == "" {
		a.RepoOwner = readFromEnv(ENV_REPO_OWNER)
	}
	if a.GithubToken == "" {
		a.GithubToken = readFromEnv(ENV_GITHUB_TOKEN)
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
		return &ConfigValidationError{"github-repo", ENV_REPO_NAME}
	}
	if a.RepoOwner == "" {
		return &ConfigValidationError{"github-owner", ENV_REPO_OWNER}
	}
	if a.GithubToken == "" {
		return &ConfigValidationError{"github-token", ENV_GITHUB_TOKEN}
	}
	if a.PullRequest == 0 {
		return &ConfigValidationError{"pull-request-id", ENV_PULL_REQUEST_ID}
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
	url := os.Getenv(ENV_PULL_REQUEST_ID)
	elements := strings.Split(url, "/")
	prNumber := elements[len(elements)-1]
	val, err := strconv.ParseInt(prNumber, 10, 0)
	if err != nil {
		logrus.Errorf("Could not parse env %s with value '%v' to int", ENV_PULL_REQUEST_ID, url)
		return 0, err
	}
	if val != 0 {
		logrus.Debugf("Reading env var %s with value '%d'", ENV_PULL_REQUEST_ID, val)
	}
	return int(val), nil
}
