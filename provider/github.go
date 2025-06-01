package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/go-github/v72/github"
	"github.com/karlderkaefer/cdk-notifier/config"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

// maxCommentLength is the maximum number of chars allowed in a single comment
// by GitHub.
const GithubMaxCommentLength = 65536

// GithubIssuesService interface for required GitHub actions with API
type GithubIssuesService interface {
	ListComments(ctx context.Context, owner string, repo string, number int, opts *github.IssueListCommentsOptions) ([]*github.IssueComment, *github.Response, error)
	DeleteComment(ctx context.Context, owner string, repo string, commentID int64) (*github.Response, error)
	EditComment(ctx context.Context, owner string, repo string, commentID int64, comment *github.IssueComment) (*github.IssueComment, *github.Response, error)
	CreateComment(ctx context.Context, owner string, repo string, number int, comment *github.IssueComment) (*github.IssueComment, *github.Response, error)
}

// GithubClient GitHub client configuration
type GithubClient struct {
	Issues         GithubIssuesService
	Context        context.Context
	Client         *github.Client
	Config         config.NotifierConfig
	CommentContent string
}

// NewGithubClient create new GitHub client. Can also consume a mocked IssueService
func NewGithubClient(ctx context.Context, cfg config.NotifierConfig) (*GithubClient, error) {
	var err error

	c := &GithubClient{
		Config:  cfg,
		Context: ctx,
	}
	if ctx == nil {
		c.Context = context.Background()
	}
	token := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: cfg.Token},
	)
	tokenClient := oauth2.NewClient(c.Context, token)

	switch cfg.Vcs {
	case config.VcsGithubEnterprise:
		c.Client, err = github.NewClient(tokenClient).WithEnterpriseURLs(
			fmt.Sprintf("https://%s/api/v3", cfg.GithubHost),
			fmt.Sprintf("https://%s/api/upload", cfg.GithubHost),
		)
		logrus.Infof("Using GitHub Enterprise Client: %s", cfg.GithubHost)
	default:
		c.Client = github.NewClient(tokenClient)
	}

	if c.Issues == nil {
		c.Issues = c.Client.Issues
	}
	return c, err
}

func transform(i *github.IssueComment) *Comment {
	var comment = &Comment{}
	if i.ID != nil {
		comment.Id = *i.ID
	}
	if i.Body != nil {
		comment.Body = *i.Body
	}
	if i.HTMLURL != nil {
		comment.Link = *i.HTMLURL
	}
	return comment
}

func (gc *GithubClient) CreateComment() (*Comment, error) {
	comment, _, err := gc.Issues.CreateComment(gc.Context, gc.Config.RepoOwner, gc.Config.RepoName, gc.Config.PullRequestID, &github.IssueComment{Body: &gc.CommentContent})
	if err != nil {
		return nil, err
	}
	if comment == nil {
		return nil, errors.New("Comment is nil. Please check your GitHub token.")
	}
	return transform(comment), err
}

func (gc *GithubClient) UpdateComment(id int64) (*Comment, error) {
	editedComment, _, err := gc.Issues.EditComment(gc.Context, gc.Config.RepoOwner, gc.Config.RepoName, id, &github.IssueComment{Body: &gc.CommentContent})
	if err != nil {
		return nil, err
	}
	if editedComment == nil {
		return nil, errors.New("Comment is nil. Please check your GitHub token.")
	}
	return transform(editedComment), err
}

func (gc *GithubClient) DeleteComment(id int64) error {
	_, err := gc.Issues.DeleteComment(gc.Context, gc.Config.RepoOwner, gc.Config.RepoName, id)
	return err
}

func (gc *GithubClient) GetCommentContent() string {
	return gc.CommentContent
}

func (gc *GithubClient) PostComment() (CommentOperation, error) {
	return postComment(gc, gc.Config)
}

func (gc *GithubClient) SetCommentContent(content string) {
	gc.CommentContent = content
}

func (gc *GithubClient) ListComments() ([]Comment, error) {
	opt := &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}
	comments, _, err := gc.Issues.ListComments(gc.Context, gc.Config.RepoOwner, gc.Config.RepoName, gc.Config.PullRequestID, opt)
	if err != nil {
		return nil, err
	}
	var result []Comment
	for _, comment := range comments {
		result = append(result, *transform(comment))
	}
	return result, nil
}
