package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

// AppConfig application configuration initialized by cobra arguments or environment variabels
type AppConfig struct {
	LogFile      string
	TagID        string
	ProjectID    int
	GitlabToken  string
	MergeRequest int
	DeleteNote   bool
	GitlabUrl    string
}

// ValidationError indicated a missing configuration either CLI argument or environment variable
type ValidationError struct {
	CliArg string
	EnvVar string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("missing argument. Set --%s argument or env var %s", e.CliArg, e.EnvVar)
}

const (
	// EnvGitlabToken Name of environment variable for Gitlab token
	EnvGitlabToken = "GITLAB_TOKEN"
	// EnvMergeRequestID Name of environment variable for pull request url
	EnvMergeRequestID = "CI_MERGE_REQUEST_IID"
	// EnvGitlabUrl Name of environment variable for Gitlab Base Url
	EnvGitlabUrl = "GITLAB_BASE_URL"
	// EnvGitlabPid Name of environment variable for Gitlab Project ID
	EnvGitlabPid = "CI_MERGE_REQUEST_PROJECT_ID"
)

// Init will create default AppConfig with following priority
// 1. Environment Variables CI_JOB_TOKEN, CI_MERGE_REQUEST_ID, CI_PROJECT_NAME, GITLAB_BASE_URL, CI_MERGE_REQUEST_PROJECT_ID
// 2. CLI args
// returns ValidationError if required field where not set
func (a *AppConfig) Init() error {
	if a.GitlabToken == "" {
		a.GitlabToken = readFromEnv(EnvGitlabToken)
	}
	if a.ProjectID == 0 {
		var err error

		pidStr := readFromEnv(EnvGitlabPid)
		pidInt64, err := strconv.ParseInt(pidStr, 10, 0)
		if err != nil {
			logrus.Errorf("Could not parse env %s with value '%v' to int", EnvMergeRequestID, pidStr)
			panic(err)
		}
		a.ProjectID = int(pidInt64)
	}
	if a.GitlabUrl == "" {
		a.GitlabUrl = readFromEnv(EnvGitlabUrl)
	}
	if a.MergeRequest == 0 {
		prNumber, err := readMergeRequestFromEnv()
		if err != nil {
			return err
		}
		a.MergeRequest = prNumber
	}
	// validate
	if a.ProjectID == 0 {
		return &ValidationError{"gitlab-pid", EnvGitlabPid}
	}
	if a.GitlabToken == "" {
		return &ValidationError{"gitlab-token", EnvGitlabToken}
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

func readMergeRequestFromEnv() (int, error) {
	url := os.Getenv(EnvMergeRequestID)
	if url == "" {
		logrus.Warnf("env var %s is not set or empty", EnvMergeRequestID)
		return 0, nil
	}
	elements := strings.Split(url, "/")
	prNumber := elements[len(elements)-1]
	val, err := strconv.ParseInt(prNumber, 10, 0)
	if err != nil {
		logrus.Errorf("Could not parse env %s with value '%v' to int", EnvMergeRequestID, url)
		return 0, err
	}
	if val != 0 {
		logrus.Debugf("Reading env var %s with value '%d'", EnvMergeRequestID, val)
	}
	return int(val), nil
}
