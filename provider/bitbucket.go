package provider

import (
	"context"
	"errors"

	"github.com/karlderkaefer/cdk-notifier/config"
	"github.com/sirupsen/logrus"
)

// maxCommentLength is the maximum number of chars allowed by Bitbucket in a
// single comment.
const BitbucketMaxCommentLength = 32768

var (
	errContentEmpty = errors.New("comment content should not be empty")
)

type BitbucketProvider struct {
	Service        IBitbucketRepositoryService
	Context        context.Context
	Client         *BitbucketClient
	Config         config.NotifierConfig
	CommentContent string
}

func NewBitbucketProvider(ctx context.Context, config config.NotifierConfig) (b *BitbucketProvider) {
	b = &BitbucketProvider{
		Context: ctx,
		Client:  NewBitbucketClient(config.TokenUser, config.Token),
		Config:  config,
	}
	if b.Context == nil {
		b.Context = context.Background()
	}
	b.Service = b.Client.Repositories
	return b
}

func (c *BitbucketComment) transform() *Comment {
	var comment = &Comment{}
	if c == nil {
		return comment
	}
	if c.Id != nil {
		comment.Id = *c.Id
	}
	if c.Content != nil {
		comment.Body = c.Content.Raw
	}
	if c.Links != nil {
		comment.Link = c.Links.Html.Href
	}
	return comment
}

func (b *BitbucketProvider) CreateComment() (*Comment, error) {
	if b.CommentContent == "" {
		return nil, errContentEmpty
	}
	comment, _, err := b.Service.CreateComment(
		b.Context,
		b.Config.RepoOwner,
		b.Config.RepoName,
		int64(b.Config.PullRequestID),
		NewBitbucketComment(b.CommentContent),
	)
	if err != nil {
		return nil, err
	}
	return comment.transform(), nil
}

func (b *BitbucketProvider) UpdateComment(id int64) (*Comment, error) {
	comment, _, err := b.Service.EditComment(
		b.Context,
		b.Config.RepoOwner,
		b.Config.RepoName,
		int64(b.Config.PullRequestID),
		id,
		NewBitbucketComment(b.CommentContent),
	)
	if err != nil {
		return nil, err
	}
	return comment.transform(), nil
}

func (b *BitbucketProvider) DeleteComment(id int64) error {
	_, _, err := b.Service.DeleteComment(
		b.Context,
		b.Config.RepoOwner,
		b.Config.RepoName,
		int64(b.Config.PullRequestID),
		id,
	)
	if err != nil {
		return err
	}
	logrus.Debugf("deleted comment with id %d\n", id)
	return nil
}

func (b *BitbucketProvider) SetCommentContent(content string) {
	b.CommentContent = content
}

func (b *BitbucketProvider) GetCommentContent() string {
	return b.CommentContent
}

func (b *BitbucketProvider) PostComment() (CommentOperation, error) {
	return postComment(b, b.Config)
}

func (b *BitbucketProvider) ListComments() ([]Comment, error) {
	opts := &ListCommentOptions{
		// filter out deleted comments
		Query: "deleted=false",
		// filter out content only to save bandwidth
		Fields: "values.id,values.content.raw,values.links.html.href",
		// set maximum pagelength so I don't need to bother with paging
		PageLength: 100,
	}
	comments, _, err := b.Service.ListComments(
		b.Context,
		b.Config.RepoOwner,
		b.Config.RepoName,
		int64(b.Config.PullRequestID),
		opts,
	)
	if err != nil {
		return nil, err
	}
	var result []Comment
	if comments != nil && comments.Values != nil {
		for _, comment := range comments.Values {
			result = append(result, *comment.transform())
		}
	}
	return result, nil
}
