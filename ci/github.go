package ci

import (
	"context"
	"github.com/google/go-github/v37/github"
	"github.com/karlderkaefer/cdk-notifier/config"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"strings"
)

const (
	// HeaderPrefix default prefix for comment message
	HeaderPrefix = "## cdk diff for"
)

// GithubIssuesService interface for required GitHub actions with API
type GithubIssuesService interface {
	ListComments(ctx context.Context, owner string, repo string, number int, opts *github.IssueListCommentsOptions) ([]*github.IssueComment, *github.Response, error)
	DeleteComment(ctx context.Context, owner string, repo string, commentID int64) (*github.Response, error)
	EditComment(ctx context.Context, owner string, repo string, commentID int64, comment *github.IssueComment) (*github.IssueComment, *github.Response, error)
	CreateComment(ctx context.Context, owner string, repo string, number int, comment *github.IssueComment) (*github.IssueComment, *github.Response, error)
}

// NotifierGithubService interface for public methods of GitHub actions required for cdk-notifier
type NotifierGithubService interface {
	ListComments() ([]*github.IssueComment, error)
	FindComment() (*github.IssueComment, error)
	PostComment() error
}

// GithubClient GitHub client configuration
type GithubClient struct {
	Issues         GithubIssuesService
	Context        context.Context
	Client         *github.Client
	Config         *config.NotifierConfig
	CommentContent string
}

// NewGithubClient create new github client. Can also consume a mocked IssueService
func NewGithubClient(ctx context.Context, config *config.NotifierConfig) *GithubClient {
	githubClient := &GithubClient{
		Config: config,
	}
	if ctx == nil {
		githubClient.Context = context.Background()
	} else {
		githubClient.Context = ctx
	}
	return githubClient
}

// SetCommentContent will pre-set the content that will be published to pull request
func (gc *GithubClient) SetCommentContent(content string) {
	gc.CommentContent = content
}

// Authenticate client with GitHub token
func (gc *GithubClient) Authenticate() error {
	token := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: gc.Config.Token},
	)
	tokenClient := oauth2.NewClient(gc.Context, token)
	gc.Client = github.NewClient(tokenClient)
	if gc.Issues == nil {
		gc.Issues = gc.Client.Issues
	}
	return nil
}

func (gc *GithubClient) setGithubIssuesService(issuesMock GithubIssuesService) {
	gc.Issues = issuesMock
}

// ListComments GitHub API implementation to list all comments of pull request
func (gc *GithubClient) ListComments() ([]*github.IssueComment, error) {
	opt := &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}
	comments, _, err := gc.Issues.ListComments(gc.Context, gc.Config.RepoOwner, gc.Config.RepoName, gc.Config.PullRequestID, opt)
	if err != nil {
		return nil, err
	}
	return comments, nil
}

// FindComment find the comment which body content start with config.HeaderPrefix "## cdk diff for".
func (gc *GithubClient) FindComment() (*github.IssueComment, error) {
	comments, err := gc.ListComments()
	if err != nil {
		return nil, err
	}
	for _, comment := range comments {
		if strings.Contains(comment.GetBody(), getHeaderTagID(gc.Config)) {
			logrus.Debugf("Found existing comment for %s", gc.Config.TagID)
			return comment, nil
		}
	}
	logrus.Debugf("Could not find existing comment for %s", gc.Config.TagID)
	return nil, nil
}

// PostComment will create GitHub comment if comment does not exist yet bases on FindComment
// If the comment already exist the content will be updated.
// If there are no cdk differences the comment will be deleted depending on DeleteComment config.AppConfig
func (gc *GithubClient) PostComment() error {
	comment, err := gc.FindComment()
	if err != nil {
		return err
	}
	if comment != nil {
		if gc.Config.DeleteComment && !diffHasChanges(gc.CommentContent) {
			_, err := gc.Issues.DeleteComment(gc.Context, gc.Config.RepoOwner, gc.Config.RepoName, comment.GetID())
			if err != nil {
				return err
			}
			logrus.Infof("Deleted comment with id %d and tag id %s because no changes detected", comment.ID, gc.Config.TagID)
			return nil
		}
		editedComment, _, err := gc.Issues.EditComment(gc.Context, gc.Config.RepoOwner, gc.Config.RepoName, *comment.ID, &github.IssueComment{Body: &gc.CommentContent})
		if err != nil {
			return err
		}
		logrus.Infof("Updated comment with id %d and tag id %s %v", editedComment.ID, gc.Config.TagID, getIssueCommentURL(editedComment))
		return nil
	}
	if !diffHasChanges(gc.CommentContent) {
		logrus.Infof("There is no diff detected for tag id %s. Skip posting diff.", gc.Config.TagID)
		return nil
	}
	newComment, _, err := gc.Issues.CreateComment(gc.Context, gc.Config.RepoOwner, gc.Config.RepoName, gc.Config.PullRequestID, &github.IssueComment{Body: &gc.CommentContent})
	if err != nil {
		return err
	}
	logrus.Infof("Created comment with id %d and tag id %s %v", newComment.ID, gc.Config.TagID, getIssueCommentURL(newComment))
	return nil
}

func getIssueCommentURL(comment *github.IssueComment) string {
	if comment == nil || comment.HTMLURL == nil {
		return ""
	}
	return *comment.HTMLURL
}
