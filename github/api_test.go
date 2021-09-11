package github

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/go-github/v37/github"
	"github.com/karlderkaefer/cdk-notifier/config"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
)

func initLogger() {
	logrus.SetLevel(7)
}

type MockGithubPullRequestService struct {
	Comments []*github.PullRequestComment
	Response *github.Response
	Error    error
}

type MockPullRequestService struct {
	comments []*github.IssueComment
}

func (m MockPullRequestService) getCommentById(id int64) *github.IssueComment {
	for _, comment := range m.comments {
		fmt.Println("checking ", comment)
		if comment.ID == &id {
			fmt.Println("should be her")
			return comment
		}
	}
	return nil
}

func (m *MockPullRequestService) DeleteComment(ctx context.Context, owner string, repo string, commentID int64) (*github.Response, error) {
	return nil, nil
}

func (m *MockPullRequestService) EditComment(ctx context.Context, owner string, repo string, commentID int64, comment *github.IssueComment) (*github.IssueComment, *github.Response, error) {
	for i, localComment := range m.comments {
		fmt.Printf("incoming comment: %v}\nincoming id: %v\ncomment: %v\n", comment, commentID, localComment)
		if *localComment.ID == commentID {
			m.comments[i] = &github.IssueComment{
				ID:   &commentID,
				Body: comment.Body,
			}
			return m.comments[i], nil, nil
		}
	}
	return nil, nil, errors.New(fmt.Sprintf("Could not find comment with id %d in database %v", commentID, m.comments))
}

func (m *MockPullRequestService) CreateComment(ctx context.Context, owner string, repo string, number int, comment *github.IssueComment) (*github.IssueComment, *github.Response, error) {
	m.comments = append(m.comments, comment)
	return comment, nil, nil
}

func (m *MockPullRequestService) ListComments(ctx context.Context, owner string, repo string, number int, opts *github.IssueListCommentsOptions) ([]*github.IssueComment, *github.Response, error) {
	return m.comments, nil, nil
}

func TestUpdateExistingComment(t *testing.T) {
	initLogger()
	commentsMock := []*github.IssueComment{
		{
			ID:   github.Int64(1),
			Body: github.String("## cdk diff for example\nhello-world"),
		},
	}

	mock := &MockPullRequestService{comments: commentsMock}
	client := NewCithubClient(context.Background(), &config.AppConfig{TagId: "example"}, mock)

	// test update existing comment
	client.CommentContent = "## cdk diff for example\nUpdated"
	found, err := client.FindComment()
	assert.NoError(t, err)
	assert.NotNil(t, found)
	err = client.PostComment()
	assert.NoError(t, err)
	assert.Len(t, mock.comments, 1, "Expect one member in mock database.")
	comment, err := client.FindComment()
	assert.NoError(t, err)
	assert.NotNil(t, comment)
	assert.Equal(t, client.CommentContent, comment.GetBody(), "Expected updated body for comment")
	t.Logf("updated comment: %v", comment)
}

func TestGithubConfig_FindComment(t *testing.T) {
	commentsMock := []*github.IssueComment{
		{
			ID:   github.Int64(1),
			Body: github.String("## cdk diff for real-tag\nhello-word"),
		},
		{
			ID:   github.Int64(2),
			Body: github.String("## cdk diff for not-real-tag\nsome-description"),
		},
	}
	mock := &MockPullRequestService{comments: commentsMock}
	client := NewCithubClient(context.Background(), &config.AppConfig{TagId: "real-tag"}, mock)

	comment, err := client.FindComment()
	assert.NoError(t, err)
	assert.NotNil(t, comment)
	assert.Equal(t, commentsMock[0], comment)

	client.TagId = "non-existing-tag"
	comment, err = client.FindComment()
	assert.NoError(t, err)
	assert.Nil(t, comment)

}

func TestGithubClient_ListComments(t *testing.T) {
	initLogger()
	maxLength := 12
	var commentsMock []*github.IssueComment
	for i := 1; i <= maxLength; i++ {
		commentsMock = append(commentsMock, &github.IssueComment{
			ID:   github.Int64(int64(i)),
			Body: github.String(fmt.Sprintf("## cdk diff for example-%d", i)),
		})
	}

	mock := &MockPullRequestService{comments: commentsMock}
	client := NewCithubClient(context.Background(), &config.AppConfig{TagId: "example"}, mock)

	comments, err := client.ListComments()
	t.Logf("%v", comments)
	assert.NoError(t, err)
	assert.Len(t, comments, maxLength, "Expected number of initial comment to be %d", maxLength)
	assert.Len(t, mock.comments, maxLength, "Expect %d members in mock database", maxLength)
	assert.Equal(t, commentsMock, comments)

	mock.comments = nil

}
