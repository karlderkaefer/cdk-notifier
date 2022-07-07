package gitlab

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/napalm684/cdk-notifier/config"
	"github.com/sirupsen/logrus"
	"github.com/xanzy/go-gitlab"
)

const (
	// HeaderPrefix default prefix for note message
	HeaderPrefix = "## cdk diff for"
)

// NotesService interface for required Gitlab actions with API
type NotesService interface {
	ListMergeRequestNotes(pid interface{}, mergeRequest int, opt *gitlab.ListMergeRequestNotesOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Note, *gitlab.Response, error)
	DeleteMergeRequestNote(pid interface{}, mergeRequest, note int, options ...gitlab.RequestOptionFunc) (*gitlab.Response, error)
	UpdateMergeRequestNote(pid interface{}, mergeRequest, note int, opt *gitlab.UpdateMergeRequestNoteOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Note, *gitlab.Response, error)
	CreateMergeRequestNote(pid interface{}, mergeRequest int, opt *gitlab.CreateMergeRequestNoteOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Note, *gitlab.Response, error)
}

// Client Gitlab client configuration
type Client struct {
	Notes  NotesService
	Pid    interface{}
	Client *gitlab.Client

	Url            string
	Token          string
	TagID          string
	NoteContent    string
	MergeRequestID int
	DeleteNotes    bool
}

func NewGitlabClient(config *config.AppConfig, notesMock NotesService) *Client {
	gitlabClient := &Client{
		Pid:            config.ProjectID,
		Url:            config.GitlabUrl,
		TagID:          config.TagID,
		MergeRequestID: config.MergeRequest,
		DeleteNotes:    config.DeleteNote,
		Token:          config.GitlabToken,
	}

	if notesMock != nil {
		gitlabClient.Notes = notesMock
	} else {
		var err error
		gitlabClient.Client, err = gitlab.NewClient(config.GitlabToken,
			gitlab.WithBaseURL(config.GitlabUrl))
		if err != nil {
			panic(err)
		}
		gitlabClient.Notes = gitlabClient.Client.Notes
	}
	return gitlabClient
}

func (gc *Client) Authenticate() {
	var err error
	gc.Client, err = gitlab.NewClient(gc.Token,
		gitlab.WithBaseURL(gc.Url))
	if err != nil {
		panic(err)
	}
}

func (gc *Client) ListMergeRequestNotes() ([]*gitlab.Note, error) {
	opt := &gitlab.ListMergeRequestNotesOptions{
		ListOptions: gitlab.ListOptions{PerPage: 100},
	}
	notes, _, err := gc.Notes.ListMergeRequestNotes(gc.Pid, gc.MergeRequestID, opt)
	if err != nil {
		return nil, err
	}
	return notes, nil
}

func (gc *Client) GetMergeRequestNote() (*gitlab.Note, error) {
	notes, err := gc.ListMergeRequestNotes()
	if err != nil {
		return nil, err
	}
	for _, note := range notes {
		if strings.Contains(note.Body, gc.getHeaderTagID()) {
			logrus.Debugf("Found existing note for %s", gc.TagID)
			return note, nil
		}
	}
	logrus.Debugf("Could not find existing note for %s", gc.TagID)
	return nil, nil
}

func (gc *Client) CreateMergeRequestNote() error {
	note, err := gc.GetMergeRequestNote()
	if err != nil {
		return err
	}
	if note != nil {
		if gc.DeleteNotes && !gc.hasChanges() {
			_, err := gc.Notes.DeleteMergeRequestNote(gc.Pid, gc.MergeRequestID, note.ID)
			if err != nil {
				return err
			}
			logrus.Infof("Deleted note with id %d and tag id %s because no changes detected", note.ID, gc.TagID)
			return nil
		}
		editedNote, _, err := gc.Notes.UpdateMergeRequestNote(gc.Pid, gc.MergeRequestID, note.ID, &gitlab.UpdateMergeRequestNoteOptions{Body: &gc.NoteContent})
		if err != nil {
			return err
		}
		logrus.Infof("Updated note with id %d and tag id %s", editedNote.ID, gc.TagID)
		return nil
	}
	if !gc.hasChanges() {
		logrus.Infof("There is no diff detected for tag id %s. Skip posting diff.", gc.TagID)
		return nil
	}
	newNote, _, err := gc.Notes.CreateMergeRequestNote(gc.Pid, gc.MergeRequestID, &gitlab.CreateMergeRequestNoteOptions{Body: &gc.NoteContent})
	if err != nil {
		return err
	}
	logrus.Infof("Created note with id %d and tag id %s", newNote.ID, gc.TagID)
	return nil
}

func (gc *Client) getHeaderTagID() string {
	return fmt.Sprintf("%s %s", HeaderPrefix, gc.TagID)
}

func (gc *Client) hasChanges() bool {
	regex := regexp.MustCompile(`(?m)(Policy Changes|Resources\n|Statement Changes)`)
	return regex.MatchString(gc.NoteContent)
}
