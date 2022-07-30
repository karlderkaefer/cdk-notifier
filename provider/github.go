package provider

import (
	"context"
	"github.com/google/go-github/v37/github"
	"github.com/karlderkaefer/cdk-notifier/config"
	"golang.org/x/oauth2"
)

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
func NewGithubClient(ctx context.Context, config config.NotifierConfig) *GithubClient {
	c := &GithubClient{
		Config:  config,
		Context: ctx,
	}
	if ctx == nil {
		c.Context = context.Background()
	}
	token := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: config.Token},
	)
	tokenClient := oauth2.NewClient(c.Context, token)
	c.Client = github.NewClient(tokenClient)

	if c.Issues == nil {
		c.Issues = c.Client.Issues
	}
	return c
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
	return transform(comment), err
}

func (gc *GithubClient) UpdateComment(id int64) (*Comment, error) {
	editedComment, _, err := gc.Issues.EditComment(gc.Context, gc.Config.RepoOwner, gc.Config.RepoName, id, &github.IssueComment{Body: &gc.CommentContent})
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
