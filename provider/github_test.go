package provider

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/google/go-github/v65/github"
	"github.com/karlderkaefer/cdk-notifier/config"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func initLogger() {
	logrus.SetLevel(7)
}

const defaultTag = "test-tag"

type MockPullRequestService struct {
	sync.RWMutex
	comments []*github.IssueComment
}

func (m *MockPullRequestService) DeleteComment(ctx context.Context, owner string, repo string, commentID int64) (*github.Response, error) {
	var index int
	var found bool
	for i, comment := range m.comments {
		if *comment.ID == commentID {
			index = i
			found = true
		}
	}
	if !found {
		return nil, fmt.Errorf("could not find comment to delete with id %d", commentID)
	}
	copy(m.comments[index:], m.comments[index+1:])
	m.comments[len(m.comments)-1] = nil
	m.comments = m.comments[:len(m.comments)-1]
	return nil, nil
}

func (m *MockPullRequestService) EditComment(ctx context.Context, owner string, repo string, commentID int64, comment *github.IssueComment) (*github.IssueComment, *github.Response, error) {
	m.Lock()
	defer m.Unlock()
	for i, localComment := range m.comments {
		if *localComment.ID == commentID {
			m.comments[i] = &github.IssueComment{
				ID:      &commentID,
				Body:    comment.Body,
				HTMLURL: comment.HTMLURL,
			}
			return m.comments[i], nil, nil
		}
	}
	return nil, nil, fmt.Errorf("could not find comment with id %d in database %v", commentID, m.comments)
}

func (m *MockPullRequestService) CreateComment(ctx context.Context, owner string, repo string, number int, comment *github.IssueComment) (*github.IssueComment, *github.Response, error) {
	m.Lock()
	defer m.Unlock()
	m.comments = append(m.comments, comment)
	return comment, nil, nil
}

func (m *MockPullRequestService) ListComments(ctx context.Context, owner string, repo string, number int, opts *github.IssueListCommentsOptions) ([]*github.IssueComment, *github.Response, error) {
	return m.comments, nil, nil
}

func defaultTestGithubProvider(comments []*github.IssueComment) *GithubClient {
	mock := &MockPullRequestService{comments: comments}
	return &GithubClient{
		Issues:         mock,
		Context:        context.Background(),
		Config:         config.NotifierConfig{TagID: "test-tag", DeleteComment: true},
		CommentContent: defaultTag,
	}
}

func TestNewGithubClient(t *testing.T) {
	notifierConfig := config.NotifierConfig{
		Token: "",
	}
	client, err := NewGithubClient(context.TODO(), notifierConfig)
	assert.NoError(t, err)

	comment, err := client.PostComment()
	assert.Error(t, err)
	assert.IsType(t, &github.ErrorResponse{}, err)
	assert.Equal(t, comment, API_COMMENT_NOTHING)
}

func TestUpdateExistingComment(t *testing.T) {
	initLogger()
	commentsMock := []*github.IssueComment{
		{
			ID:   github.Int64(1),
			Body: github.String(fmt.Sprintf("%s %s\n%s", HeaderPrefix, defaultTag, "hello-word")),
		},
	}
	client := defaultTestGithubProvider(commentsMock)
	comments, err := client.ListComments()
	assert.NoError(t, err)
	assert.Len(t, comments, 1, "expect one comment in database")

	// test update existing comment
	client.CommentContent = fmt.Sprintf("%s %s\n%s", HeaderPrefix, defaultTag, "there are Policy Changes detected")
	operation, err := client.PostComment()
	assert.NoError(t, err)
	comments, err = client.ListComments()
	assert.NoError(t, err)
	assert.Len(t, comments, 1, "expect one comment in database after update")
	assert.Equal(t, API_COMMENT_UPDATED.String(), operation.String(), "Expected Update Operation")
}

func TestGithubClient_CreateComment(t *testing.T) {
	initLogger()
	var commentsMock []*github.IssueComment
	client := defaultTestGithubProvider(commentsMock)
	client.SetCommentContent("New Comment")
	assert.Len(t, commentsMock, 0)
	comment, err := client.CreateComment()
	assert.NoError(t, err)
	assert.NotNil(t, comment)
	assert.Equal(t, "New Comment", comment.Body)
	comments, err := client.ListComments()
	assert.NoError(t, err)
	assert.Len(t, comments, 1)
}

func TestGithubClient_DeleteComment(t *testing.T) {
	initLogger()
	commentsMock := []*github.IssueComment{
		{
			ID:   github.Int64(1),
			Body: github.String(fmt.Sprintf("%s %s\n%s", HeaderPrefix, defaultTag, "hello-word")),
		},
	}
	client := defaultTestGithubProvider(commentsMock)
	comments, err := client.ListComments()
	assert.NoError(t, err)
	assert.Len(t, comments, 1)
	err = client.DeleteComment(int64(1))
	assert.NoError(t, err)
	comments, err = client.ListComments()
	assert.NoError(t, err)
	assert.Len(t, comments, 0)
}

func TestGithubClient_ListComments(t *testing.T) {
	initLogger()
	maxLength := 12
	var commentsMock []*github.IssueComment
	for i := 1; i <= maxLength; i++ {
		commentsMock = append(commentsMock, &github.IssueComment{
			ID:   github.Int64(int64(i)),
			Body: github.String(fmt.Sprintf("%s example-%d", HeaderPrefix, i)),
		})
	}
	client := defaultTestGithubProvider(commentsMock)
	comments, err := client.ListComments()
	t.Logf("%v", comments)
	assert.NoError(t, err)
	assert.Len(t, comments, maxLength, "Expected number of initial comment to be %d", maxLength)

}

type HasChangesTest struct {
	input            string
	expectHasChanges bool
}

func TestClient_hasChanges(t *testing.T) {
	cases := []HasChangesTest{
		{
			input:            readFile("../data/cdk-multistack.log"),
			expectHasChanges: true,
		},
		{
			input:            "Stack core-network\nThere were no differences\nStack corenetwork735961878498apsoutheast21AE73C6D\nThere were no differences\nThere were no Resources differences",
			expectHasChanges: false,
		},
		{
			input:            "Stack core-network\nThere were no differences\nResources\nStack corenetwork735961878498apsoutheast21AE73C6D\nThere were no differences",
			expectHasChanges: true,
		},
	}
	for _, c := range cases {
		client := GithubClient{
			CommentContent: c.input,
		}
		actual := diffHasChanges(client.CommentContent)
		assert.Equal(t, c.expectHasChanges, actual)
	}
}

func readFile(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		logrus.Fatal(err)
	}
	return string(content)
}
