package provider

import (
	"context"
	"errors"
	"testing"

	"github.com/karlderkaefer/cdk-notifier/config"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// mockNotifierService simulates the NotifierService interface for testing postComment.
type mockNotifierService struct {
	commentExists     bool
	returnFindErr     bool
	deleteErr         error
	updateErr         error
	createErr         error
	commentContent    string
	deletedCommentId  int64
	updatedCommentId  int64
	createdCommentId  int64
}

func (m *mockNotifierService) CreateComment() (*Comment, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	m.createdCommentId = 789
	return &Comment{Id: m.createdCommentId, Body: m.commentContent}, nil
}

func (m *mockNotifierService) UpdateComment(id int64) (*Comment, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	m.updatedCommentId = id
	return &Comment{Id: id, Body: m.commentContent}, nil
}

func (m *mockNotifierService) DeleteComment(id int64) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	m.deletedCommentId = id
	return nil
}

func (m *mockNotifierService) SetCommentContent(content string) {
	m.commentContent = content
}

func (m *mockNotifierService) GetCommentContent() string {
	return m.commentContent
}

// PostComment is not used directly here because postComment calls findComment() itself.
func (m *mockNotifierService) PostComment() (CommentOperation, error) {
	return API_COMMENT_NOTHING, nil
}

// ListComments is called within findComment()
func (m *mockNotifierService) ListComments() ([]Comment, error) {
	if m.returnFindErr {
		return nil, errors.New("mock: error on findComment")
	}
	if m.commentExists {
		return []Comment{{Id: 123, Body: "## cdk diff for myTag\nSomething"}}, nil
	}
	return []Comment{}, nil
}

// TestPostCommentDataDriven tests all branches of postComment with data-driven style.
func TestPostCommentDataDriven(t *testing.T) {
    logrus.SetLevel(logrus.DebugLevel)

    tests := []struct {
        name          string
        ms            mockNotifierService
        cfg           config.NotifierConfig
        wantOp        CommentOperation
        wantErr       bool
        wantDeletedID int64
        wantUpdatedID int64
        wantCreatedID int64
    }{
        {
            name: "errorOnFindComment",
            ms: mockNotifierService{
                returnFindErr:   true,
                commentExists:   false,
                commentContent:  "Policy Changes",
            },
            cfg:     config.NotifierConfig{},
            wantOp:  API_COMMENT_NOTHING,
            wantErr: true,
        },
        {
            name: "existingCommentDeleteCommentForceDelete",
            ms: mockNotifierService{
                commentExists:  true,
                commentContent: "No changes",
            },
            cfg: config.NotifierConfig{
                DeleteComment:      true,
                ForceDeleteComment: true,
                TagID:              "myTag",
            },
            wantOp:        API_COMMENT_DELETED,
            wantErr:       false,
            wantDeletedID: 123,
        },
        {
            name: "existingCommentDeleteCommentNoDiff",
            ms: mockNotifierService{
                commentExists:  true,
                commentContent: "No changes",
            },
            cfg: config.NotifierConfig{
                DeleteComment:      true,
                ForceDeleteComment: false,
                TagID:              "myTag",
            },
            wantOp:        API_COMMENT_DELETED,
            wantErr:       false,
            wantDeletedID: 123,
        },
        {
            name: "existingCommentHasDiff->Update",
            ms: mockNotifierService{
                commentExists:  true,
                commentContent: "Stack resources\nPolicy Changes",
            },
            cfg:           config.NotifierConfig{TagID: "myTag"},
            wantOp:        API_COMMENT_UPDATED,
            wantUpdatedID: 123, // match updated comment ID
        },
        {
            name: "noCommentNoDiff->Nothing",
            ms: mockNotifierService{
                commentExists:  false,
                commentContent: "no meaningful changes here",
            },
            cfg:    config.NotifierConfig{TagID: "myTag"},
            wantOp: API_COMMENT_NOTHING,
        },
        {
            name: "noCommentHasDiff->Create",
            ms: mockNotifierService{
                commentExists:  false,
                commentContent: "Policy Changes found",
            },
            cfg:           config.NotifierConfig{TagID: "myTag"},
            wantOp:        API_COMMENT_CREATED,
            wantCreatedID: 789, // match created comment ID
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            op, err := postComment(&tt.ms, tt.cfg)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
            assert.Equal(t, tt.wantOp, op)
            assert.Equal(t, tt.wantDeletedID, tt.ms.deletedCommentId, "deleted comment ID")
            assert.Equal(t, tt.wantUpdatedID, tt.ms.updatedCommentId, "updated comment ID")
            assert.Equal(t, tt.wantCreatedID, tt.ms.createdCommentId, "created comment ID")
        })
    }
}

type testCaseCreateService struct {
	vcs           string
	expectedType  interface{}
	expectedError error
	description   string
}

func TestCreateNotifierService(t *testing.T) {
	testCase := []testCaseCreateService{
		{
			vcs:           "droneCi",
			expectedType:  nil,
			expectedError: errors.New("unsupported Version Control System: droneCi"),
			description:   "test VCS is not set",
		},
		{
			vcs:           config.VcsBitbucket,
			expectedType:  &BitbucketProvider{},
			expectedError: nil,
			description:   "test bitbucket",
		},
		{
			vcs:           config.VcsGithub,
			expectedType:  &GithubClient{},
			expectedError: nil,
			description:   "test github",
		},
		{
			vcs:           config.VcsGitlab,
			expectedType:  &GitlabClient{},
			expectedError: nil,
			description:   "test gitlab",
		},
	}
	for _, c := range testCase {
		t.Logf("%s", c.description)
		notifierConfig := config.NotifierConfig{
			Token: "",
			Vcs:   c.vcs,
		}
		svc, err := CreateNotifierService(context.TODO(), notifierConfig)
		if c.expectedError != nil {
			assert.Error(t, err)
		} else {
			assert.NotNil(t, svc)
			assert.IsType(t, c.expectedType, svc)
		}
	}

}

func Test_DiffOutputChanges(t *testing.T) {
	assert.True(t, diffHasChanges(`
Stack OutputStack
Outputs
[+] Output output output: {"Value":"","Export":{"Name":"output"}}
`))
}
