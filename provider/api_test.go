package provider

import (
	"errors"
	"testing"

	"github.com/karlderkaefer/cdk-notifier/config"
	"github.com/stretchr/testify/assert"
)

type testCaseCreateService struct {
	vcs           string
	expectedType  interface{}
	expectedError error
	description   string
}

func TestCreateNotifierService(t *testing.T) {
	testCase := []testCaseCreateService{
		{
			vcs:           "droneCi",
			expectedType:  nil,
			expectedError: errors.New("unsupported Version Control System: droneCi"),
			description:   "test VCS is not set",
		},
		{
			vcs:           config.VcsBitbucket,
			expectedType:  &BitbucketProvider{},
			expectedError: nil,
			description:   "test bitbucket",
		},
		{
			vcs:           config.VcsGithub,
			expectedType:  &GithubClient{},
			expectedError: nil,
			description:   "test github",
		},
		{
			vcs:           config.VcsGitlab,
			expectedType:  &GitlabClient{},
			expectedError: nil,
			description:   "test gitlab",
		},
	}
	for _, c := range testCase {
		t.Logf("%s", c.description)
		notifierConfig := config.NotifierConfig{
			Token: "",
			Vcs:   c.vcs,
		}
		svc, err := CreateNotifierService(nil, notifierConfig)
		if c.expectedError != nil {
			assert.Error(t, err)
		} else {
			assert.NotNil(t, svc)
			assert.IsType(t, c.expectedType, svc)
		}
	}

}
