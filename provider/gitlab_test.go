package provider

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/karlderkaefer/cdk-notifier/config"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/xanzy/go-gitlab"
)

func initGitlabLogger() {
	logrus.SetLevel(7)
}

type MockMergeRequestService struct {
	sync.RWMutex
	notes []*gitlab.Note
}

type MockProjectService struct {
	sync.RWMutex
}

func (p *MockProjectService) GetProject(pid interface{}, opt *gitlab.GetProjectOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Project, *gitlab.Response, error) {
	project := &gitlab.Project{
		ID: 1,
	}
	return project, nil, nil
}

func (m *MockMergeRequestService) DeleteMergeRequestNote(pid interface{}, mergeRequest, note int, options ...gitlab.RequestOptionFunc) (*gitlab.Response, error) {
	var index int
	var found bool
	for i, currNote := range m.notes {
		if currNote.ID == note {
			index = i
			found = true
		}
	}
	if !found {
		return nil, fmt.Errorf("could not find note to delete with id %d", note)
	}
	copy(m.notes[index:], m.notes[index+1:])
	m.notes[len(m.notes)-1] = nil
	m.notes = m.notes[:len(m.notes)-1]
	return nil, nil
}

func (m *MockMergeRequestService) UpdateMergeRequestNote(pid interface{}, mergeRequest, note int, opt *gitlab.UpdateMergeRequestNoteOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Note, *gitlab.Response, error) {
	m.Lock()
	defer m.Unlock()
	for i, localNote := range m.notes {
		if localNote.ID == note {
			m.notes[i] = &gitlab.Note{
				ID:   note,
				Body: *opt.Body,
			}
			return m.notes[i], nil, nil
		}
	}
	return nil, nil, fmt.Errorf("could not find note with id %d in database %v", note, *opt.Body)
}

func (m *MockMergeRequestService) CreateMergeRequestNote(pid interface{}, mergeRequest int, opt *gitlab.CreateMergeRequestNoteOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Note, *gitlab.Response, error) {
	m.Lock()
	defer m.Unlock()
	note := &gitlab.Note{
		ID:   1,
		Body: *opt.Body,
	}
	m.notes = append(m.notes, note)
	return note, nil, nil
}

func (m *MockMergeRequestService) ListMergeRequestNotes(pid interface{}, mergeRequest int, opt *gitlab.ListMergeRequestNotesOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Note, *gitlab.Response, error) {
	return m.notes, nil, nil
}

func defaultTestGitlabProvider(notes []*gitlab.Note) *GitlabClient {
	mock := &MockMergeRequestService{notes: notes}
	mockProj := &MockProjectService{}
	return &GitlabClient{
		Notes:       mock,
		Projects:    mockProj,
		Context:     context.Background(),
		Config:      config.NotifierConfig{TagID: "test-tag", DeleteComment: true},
		NoteContent: defaultTag,
	}
}

func TestNewGitlabClient(t *testing.T) {
	notifierConfig := config.NotifierConfig{
		Token: "",
		Url:   "https://gitlab.com/",
	}
	client := NewGitlabClient(context.TODO(), notifierConfig)
	comment, err := client.PostComment()
	assert.Error(t, err)
	assert.IsType(t, &gitlab.ErrorResponse{}, err)
	assert.Equal(t, comment, API_COMMENT_NOTHING)
}

func TestGitlabClient_UpdateExistingComment(t *testing.T) {
	initGitlabLogger()
	notesMock := []*gitlab.Note{
		{
			ID:   1,
			Body: fmt.Sprintf("%s %s\n%s", HeaderPrefix, defaultTag, "hello-word"),
		},
	}
	client := defaultTestGitlabProvider(notesMock)
	comments, err := client.ListComments()
	assert.NoError(t, err)
	assert.Len(t, comments, 1, "expect one comment in database")

	// test update existing comment
	client.NoteContent = fmt.Sprintf("%s %s\n%s", HeaderPrefix, defaultTag, "there are Policy Changes detected")
	operation, err := client.PostComment()
	assert.NoError(t, err)
	comments, err = client.ListComments()
	assert.NoError(t, err)
	assert.Len(t, comments, 1, "expect one comment in database after update")
	assert.Equal(t, API_COMMENT_UPDATED.String(), operation.String(), "Expected Update Operation")
}

func TestGitlabClient_CreateComment(t *testing.T) {
	initLogger()
	var notesMock []*gitlab.Note
	client := defaultTestGitlabProvider(notesMock)
	client.SetCommentContent("New Comment")
	assert.Len(t, notesMock, 0)
	comment, err := client.CreateComment()
	assert.NoError(t, err)
	assert.NotNil(t, comment)
	assert.Equal(t, "New Comment", comment.Body)
	comments, err := client.ListComments()
	assert.NoError(t, err)
	assert.Len(t, comments, 1)
}

func TestGitlabClient_DeleteComment(t *testing.T) {
	initLogger()
	notesMock := []*gitlab.Note{
		{
			ID:   1,
			Body: fmt.Sprintf("%s %s\n%s", HeaderPrefix, defaultTag, "hello-word"),
		},
	}
	client := defaultTestGitlabProvider(notesMock)
	comments, err := client.ListComments()
	assert.NoError(t, err)
	assert.Len(t, comments, 1)
	err = client.DeleteComment(int64(1))
	assert.NoError(t, err)
	comments, err = client.ListComments()
	assert.NoError(t, err)
	assert.Len(t, comments, 0)
}

func TestGitlabClient_ListComments(t *testing.T) {
	initLogger()
	maxLength := 12
	var notesMock []*gitlab.Note
	for i := 1; i <= maxLength; i++ {
		notesMock = append(notesMock, &gitlab.Note{
			ID:   i,
			Body: fmt.Sprintf("%s example-%d", HeaderPrefix, i),
		})
	}
	client := defaultTestGitlabProvider(notesMock)
	comments, err := client.ListComments()
	t.Logf("%v", comments)
	assert.NoError(t, err)
	assert.Len(t, comments, maxLength, "Expected number of initial comment to be %d", maxLength)
}

func TestGitlabClient_hasChanges(t *testing.T) {
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
		client := GitlabClient{
			NoteContent: c.input,
		}
		actual := diffHasChanges(client.NoteContent)
		assert.Equal(t, c.expectHasChanges, actual)
	}
}
