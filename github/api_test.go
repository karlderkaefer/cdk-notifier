package github

import (
	"context"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
)

func initLogger() {
	logrus.SetLevel(7)
}

func TestGithubConfig_FindComment(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping testing in short mode")
	}
	initLogger()
	gc := &GithubConfig{
		Owner:         "signavio",
		Repo:          "cdk-orb",
		TagId:         "test",
		PullRequestId: 23,
		Context:       context.Background(),
		//Logger: zap.S(),
	}
	gc.Authenticate()
	comment, err := gc.FindComment()
	assert.NoError(t, err)
	assert.NotNil(t, comment)
	println(comment.GetBody())
}

func TestGithubConfig_PostComment(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping testing in short mode")
	}
	initLogger()
	gc := &GithubConfig{
		Owner:          "signavio",
		Repo:           "cdk-orb",
		TagId:          "more stuff",
		PullRequestId:  23,
		Context:        context.Background(),
		CommentContent: "## cdk diff for more stuff",
		//Logger: zap.S(),
	}
	gc.Authenticate()
	err := gc.PostComment()
	assert.NoError(t, err)

}
