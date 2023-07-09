package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/karlderkaefer/cdk-notifier/config"
	"github.com/stretchr/testify/assert"
)

type MockBitbucketRepositoryService struct {
	sync.RWMutex
	comments []*BitbucketComment
}

func (m *MockBitbucketRepositoryService) ListComments(ctx context.Context, owner string, repo string, prId int64, opts *ListCommentOptions) (*BitbucketComments, *http.Response, error) {
	comments := &BitbucketComments{}
	for _, comment := range m.comments {
		comments.Values = append(comments.Values, *comment)
	}
	return comments, nil, nil
}

func (m *MockBitbucketRepositoryService) EditComment(ctx context.Context, owner string, repo string, prId int64, commentID int64, comment *BitbucketComment) (*BitbucketComment, *http.Response, error) {
	m.Lock()
	defer m.Unlock()
	for i, localComment := range m.comments {
		if *localComment.Id == commentID {
			m.comments[i] = &BitbucketComment{
				Id:      &commentID,
				Content: comment.Content,
				Links:   comment.Links,
			}
			return m.comments[i], nil, nil
		}
	}
	return nil, nil, fmt.Errorf("could not find comment with id %d in database %v", commentID, m.comments)
}

func (m *MockBitbucketRepositoryService) CreateComment(ctx context.Context, owner string, repo string, prId int64, comment *BitbucketComment) (*BitbucketComment, *http.Response, error) {
	m.Lock()
	defer m.Unlock()
	id := int64(len(m.comments))
	comment.Id = &id
	m.comments = append(m.comments, comment)
	return comment, nil, nil
}

func (m *MockBitbucketRepositoryService) DeleteComment(ctx context.Context, owner string, repo string, prId int64, commentID int64) (*BitbucketComment, *http.Response, error) {
	var index int
	var found bool
	for i, comment := range m.comments {
		if *comment.Id == commentID {
			index = i
			found = true
		}
	}
	if !found {
		return nil, nil, fmt.Errorf("could not find comment to delete with id %d", commentID)
	}
	copy(m.comments[index:], m.comments[index+1:])
	m.comments[len(m.comments)-1] = nil
	m.comments = m.comments[:len(m.comments)-1]
	return nil, nil, nil
}

func defaultTestBitbucketProvider(comments []*BitbucketComment) *BitbucketProvider {
	mock := &MockBitbucketRepositoryService{comments: comments}
	return &BitbucketProvider{
		Service:        mock,
		Context:        context.Background(),
		Config:         config.NotifierConfig{TagID: "test-tag", DeleteComment: true},
		CommentContent: "test-content",
	}
}

func TestBitbucketProvider_CreateAndDeleteComment(t *testing.T) {
	initLogger()
	client := defaultTestBitbucketProvider(nil)

	comments, err := client.ListComments()
	assert.NoError(t, err)
	assert.Len(t, comments, 0, "asssuming no comments in beginning")

	// add first comment
	client.SetCommentContent("testBody1")
	comment, err := client.CreateComment()
	assert.NoError(t, err)
	comments, err = client.ListComments()
	assert.NoError(t, err)
	assert.Len(t, comments, 1, "expected to 1 comment in database")
	assert.Equal(t, comment.Body, "testBody1")

	// add second comment
	client.SetCommentContent("testBody2")
	comment, err = client.CreateComment()
	assert.NoError(t, err)
	comments, err = client.ListComments()
	assert.NoError(t, err)
	assert.Len(t, comments, 2, "expected to 2 comment in database")
	assert.Equal(t, comment.Body, "testBody2")

	// delete second comment
	err = client.DeleteComment(1)
	assert.NoError(t, err)
	comments, err = client.ListComments()
	fmt.Println(comments)
	assert.NoError(t, err)
	assert.Len(t, comments, 1, "expected to have 1 comment in database after deletion")

}

type bitbucketTestObject struct {
	input                string
	tag                  string
	expectedError        error
	expectedOperation    CommentOperation
	expectedCommentCount int
}

func TestBitbucketProvider_PostComment(t *testing.T) {
	inputs := []bitbucketTestObject{
		{
			input:                "There were no differences",
			tag:                  "test",
			expectedError:        nil,
			expectedOperation:    API_COMMENT_NOTHING,
			expectedCommentCount: 0,
		},
		{
			input:                "not relevant",
			tag:                  "test",
			expectedError:        nil,
			expectedOperation:    API_COMMENT_NOTHING,
			expectedCommentCount: 0,
		},
		{
			input:                "8 Policy Changes changed",
			tag:                  "test",
			expectedError:        nil,
			expectedOperation:    API_COMMENT_CREATED,
			expectedCommentCount: 1,
		},
		{
			input:                "10 Policy Changes changed",
			tag:                  "test",
			expectedError:        nil,
			expectedOperation:    API_COMMENT_UPDATED,
			expectedCommentCount: 1,
		},
		{
			input:                "10 Policy Changes changed",
			tag:                  "test2",
			expectedError:        nil,
			expectedOperation:    API_COMMENT_CREATED,
			expectedCommentCount: 2,
		},
		{
			input:                "There were no differences",
			tag:                  "test2",
			expectedError:        nil,
			expectedOperation:    API_COMMENT_DELETED,
			expectedCommentCount: 1,
		},
	}
	initLogger()
	client := defaultTestBitbucketProvider(nil)
	for i, o := range inputs {
		client.Config.TagID = o.tag
		content := fmt.Sprintf("%s\n%s", getHeaderTagID(client.Config), o.input)
		client.SetCommentContent(content)
		commentOperation, err := client.PostComment()
		if o.expectedError == nil {
			assert.NoError(t, err)
		}
		assert.Equal(t, o.expectedOperation.String(), commentOperation.String(), "Expect API Operation")
		comments, err := client.ListComments()
		assert.NoError(t, err)
		assert.Len(t, comments, o.expectedCommentCount, "expected comments count to be equal for test object with index %d", i)
	}
}

func TestNewBitbucketProvider(t *testing.T) {
	notifierConfig := config.NotifierConfig{
		Token: "",
	}
	client := NewBitbucketProvider(context.TODO(), notifierConfig)
	comment, err := client.PostComment()
	assert.Error(t, err)
	assert.Equal(t, errors.New("BitBucket API Error: 401 Unauthorized "), err)
	assert.Equal(t, comment, API_COMMENT_NOTHING)
}
