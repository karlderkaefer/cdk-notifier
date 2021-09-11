package github

import (
	"context"
	"fmt"
	"github.com/google/go-github/v37/github"
	"github.com/karlderkaefer/cdk-notifier/config"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"regexp"
	"strings"
)

const (
	// HeaderPrefix default prefix for comment message
	HeaderPrefix = "## cdk diff for"
)

// IssuesService interface for required GitHub actions with API
type IssuesService interface {
	ListComments(ctx context.Context, owner string, repo string, number int, opts *github.IssueListCommentsOptions) ([]*github.IssueComment, *github.Response, error)
	DeleteComment(ctx context.Context, owner string, repo string, commentID int64) (*github.Response, error)
	EditComment(ctx context.Context, owner string, repo string, commentID int64, comment *github.IssueComment) (*github.IssueComment, *github.Response, error)
	CreateComment(ctx context.Context, owner string, repo string, number int, comment *github.IssueComment) (*github.IssueComment, *github.Response, error)
}

// NotifierService interface for public methods of GitHub actions required for cdk-notifier
type NotifierService interface {
	ListComments() ([]*github.IssueComment, error)
	FindComment() (*github.IssueComment, error)
	PostComment() error
}

// Client GitHub client configuration
type Client struct {
	Issues  IssuesService
	Context context.Context
	Client  *github.Client

	Token          string
	Owner          string
	Repo           string
	TagID          string
	CommentContent string
	PullRequestID  int
	DeleteComments bool
}

// NewGithubClient create new github client. Can also consume a mocked IssueService
func NewGithubClient(ctx context.Context, config *config.AppConfig, issuesMock IssuesService) *Client {
	githubClient := &Client{
		Owner:          config.RepoOwner,
		Repo:           config.RepoName,
		TagID:          config.TagID,
		PullRequestID:  config.PullRequest,
		DeleteComments: config.DeleteComment,
		Token:          config.GithubToken,
	}
	if ctx == nil {
		githubClient.Context = context.Background()
	} else {
		githubClient.Context = ctx
	}
	if issuesMock != nil {
		githubClient.Issues = issuesMock
	} else {
		cred := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: config.GithubToken},
		)
		tokenClient := oauth2.NewClient(ctx, cred)
		githubClient.Client = github.NewClient(tokenClient)
		githubClient.Issues = githubClient.Client.Issues
	}
	return githubClient
}

// Authenticate authenticate client with github token
func (gc *Client) Authenticate() {
	token := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: gc.Token},
	)
	tokenClient := oauth2.NewClient(gc.Context, token)
	gc.Client = github.NewClient(tokenClient)
}

// ListComments GitHub API implementation to list all comments of pull request
func (gc *Client) ListComments() ([]*github.IssueComment, error) {
	opt := &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}
	comments, _, err := gc.Issues.ListComments(gc.Context, gc.Owner, gc.Repo, gc.PullRequestID, opt)
	if err != nil {
		return nil, err
	}
	return comments, nil
}

// FindComment find the comment which body content start with config.HeaderPrefix "## cdk diff for".
func (gc *Client) FindComment() (*github.IssueComment, error) {
	comments, err := gc.ListComments()
	if err != nil {
		return nil, err
	}
	for _, comment := range comments {
		if strings.Contains(comment.GetBody(), gc.getHeaderTagID()) {
			logrus.Debugf("Found existing comment for %s", gc.TagID)
			return comment, nil
		}
	}
	logrus.Debugf("Could not find existing comment for %s", gc.TagID)
	return nil, nil
}

// PostComment will create GitHub comment if comment does not exist yet bases on FindComment
// If the comment already exist the content will be updated.
// If there are no cdk differences the comment will be deleted depending on DeleteComment config.AppConfig
func (gc *Client) PostComment() error {
	comment, err := gc.FindComment()
	if err != nil {
		return err
	}
	if comment != nil {
		if gc.DeleteComments && !gc.hasChanges() {
			_, err := gc.Issues.DeleteComment(gc.Context, gc.Owner, gc.Repo, comment.GetID())
			if err != nil {
				return err
			}
			logrus.Infof("Deleted comment with id %d and tag id %s because no changes detected", comment.ID, gc.TagID)
			return nil
		}
		editedComment, _, err := gc.Issues.EditComment(gc.Context, gc.Owner, gc.Repo, *comment.ID, &github.IssueComment{Body: &gc.CommentContent})
		if err != nil {
			return err
		}
		logrus.Infof("Updated comment with id %d and tag id %s %v", editedComment.ID, gc.TagID, getIssueCommentURL(editedComment))
		return nil
	}
	if !gc.hasChanges() {
		logrus.Infof("There is no diff detected for tag id %s. Skip posting diff.", gc.TagID)
		return nil
	}
	newComment, _, err := gc.Issues.CreateComment(gc.Context, gc.Owner, gc.Repo, gc.PullRequestID, &github.IssueComment{Body: &gc.CommentContent})
	if err != nil {
		return err
	}
	logrus.Infof("Created comment with id %d and tag id %s %v", newComment.ID, gc.TagID, getIssueCommentURL(newComment))
	return nil
}

func (gc *Client) getHeaderTagID() string {
	return fmt.Sprintf("%s %s", HeaderPrefix, gc.TagID)
}

func (gc *Client) hasChanges() bool {
	regex := regexp.MustCompile(`(?m)(Policy Changes|Resources\n|Statement Changes)`)
	return regex.MatchString(gc.CommentContent)
}

func getIssueCommentURL(comment *github.IssueComment) string {
	if comment == nil || comment.HTMLURL == nil {
		return ""
	}
	return *comment.HTMLURL
}
