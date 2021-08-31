package github

import (
	"context"
	"fmt"
	"github.com/google/go-github/v37/github"
	"github.com/karlderkaefer/cdk-notifier/config"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"strings"
)

type GithubConfig struct {
	Token          string
	Owner          string
	Repo           string
	TagId          string
	CommentContent string
	PullRequestId  int
	Context        context.Context
	Client         *github.Client
	DeleteComments bool
}

func NewGithubConfig(config *config.AppConfig) *GithubConfig {
	return &GithubConfig{
		Owner:          config.RepoOwner,
		Repo:           config.RepoName,
		TagId:          config.TagId,
		PullRequestId:  config.PullRequest,
		DeleteComments: config.DeleteComment,
		Token:          config.GithubToken,
	}
}

func (gc *GithubConfig) Authenticate() {
	token := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: gc.Token},
	)
	tokenClient := oauth2.NewClient(gc.Context, token)
	gc.Client = github.NewClient(tokenClient)
}

func (gc *GithubConfig) ListComments() ([]*github.IssueComment, error) {
	opt := &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}
	comments, _, err := gc.Client.Issues.ListComments(gc.Context, gc.Owner, gc.Repo, gc.PullRequestId, opt)
	if err != nil {
		return nil, err
	}
	return comments, nil
}

func (gc *GithubConfig) FindComment() (*github.IssueComment, error) {
	comments, err := gc.ListComments()
	if err != nil {
		return nil, err
	}
	for _, comment := range comments {
		id := fmt.Sprintf("## cdk diff for %s", gc.TagId)
		if strings.Contains(comment.GetBody(), id) {
			logrus.Debugf("Found existing comment for %s", gc.TagId)
			return comment, nil
		}
	}
	logrus.Debugf("Could not find existing comment for %s", gc.TagId)
	return nil, nil
}

func (gc *GithubConfig) PostComment() error {
	comment, err := gc.FindComment()
	if err != nil {
		return err
	}
	if comment != nil {
		if gc.DeleteComments && strings.Contains(gc.CommentContent, "There were no differences") {
			_, err := gc.Client.Issues.DeleteComment(gc.Context, gc.Owner, gc.Repo, comment.GetID())
			if err != nil {
				return err
			}
			logrus.Infof("Deleted comment with id %d and tag id %s because no changes detected", comment.ID, gc.TagId)
			return nil
		}
		editedComment, _, err := gc.Client.Issues.EditComment(gc.Context, gc.Owner, gc.Repo, *comment.ID, &github.IssueComment{Body: &gc.CommentContent})
		if err != nil {
			return err
		}
		logrus.Infof("Updated comment with id %d and tag id %s", editedComment.ID, gc.TagId)
		return nil
	}
	if strings.Contains(gc.CommentContent, "There were no differences") {
		logrus.Infof("There is no diff detected for tag id %s. Skip posting diff.", gc.TagId)
		return nil
	}
	newComment, _, err := gc.Client.Issues.CreateComment(gc.Context, gc.Owner, gc.Repo, gc.PullRequestId, &github.IssueComment{Body: &gc.CommentContent})
	if err != nil {
		return err
	}
	logrus.Infof("Created comment with id %d and tag id %s", newComment.ID, gc.TagId)
	return nil
}
