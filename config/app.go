package config

import (
	"errors"
	"github.com/sirupsen/logrus"
	"os"
	"strconv"
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

func (a *AppConfig) Init() error {
	if a.RepoName == "" {
		a.RepoName = readFromEnv("CIRCLE_PROJECT_REPONAME")
	}
	if a.RepoOwner == "" {
		a.RepoOwner = readFromEnv("CIRCLE_PROJECT_USERNAME")
	}
	if a.GithubToken == "" {
		a.GithubToken = readFromEnv("GITHUB_TOKEN")
	}
	if a.PullRequest == 0 {
		a.PullRequest = readIntFromEnv("CIRCLE_PR_NUMBER")
	}
	// validate
	if a.RepoName == "" {
		return errors.New("missing argument. Set github-repo argument or env var CIRCLE_PROJECT_REPONAME")
	}
	if a.RepoOwner == "" {
		return errors.New("missing argument. Set github-owner argument or env var CIRCLE_PROJECT_USERNAME")
	}
	if a.GithubToken == "" {
		return errors.New("missing argument. Set github-token argument or env var GITHUB_TOKEN")
	}
	if a.PullRequest == 0 {
		return errors.New("missing argument. Set pull-request-id argument or env var CIRCLE_PR_NUMBER")
	}
	return nil
}

func readFromEnv(env string) string {
	val := os.Getenv(env)
	if val != "" {
		logrus.Debugf("Reading env var %s with value %s", env, val)
	}
	return val
}

func readIntFromEnv(env string) int {
	val, err := strconv.ParseInt(os.Getenv(env), 10, 0)
	if err != nil {
		logrus.Fatalf("Can not parse int from env var %s", env)
	}
	if val != 0 {
		logrus.Debugf("Reading env var %s with value %d", env, val)
	}
	return int(val)
}
