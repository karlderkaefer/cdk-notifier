package provider

import (
	"context"
	"fmt"

	"github.com/karlderkaefer/cdk-notifier/config"
	gitlab "github.com/xanzy/go-gitlab"
)

// gitlabMaxCommentLength is the maximum number of chars allowed by Gitlab in a
// single comment.
const GitlabMaxCommentLength = 1000000

// NotesService interface for required Gitlab actions with API
type GitlabNotesService interface {
	ListMergeRequestNotes(pid interface{}, mergeRequest int, opt *gitlab.ListMergeRequestNotesOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Note, *gitlab.Response, error)
	DeleteMergeRequestNote(pid interface{}, mergeRequest, note int, options ...gitlab.RequestOptionFunc) (*gitlab.Response, error)
	UpdateMergeRequestNote(pid interface{}, mergeRequest, note int, opt *gitlab.UpdateMergeRequestNoteOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Note, *gitlab.Response, error)
	CreateMergeRequestNote(pid interface{}, mergeRequest int, opt *gitlab.CreateMergeRequestNoteOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Note, *gitlab.Response, error)
}

// ProjectsService interface for required Gitlab actions with API
type GitlabProjectsService interface {
	GetProject(pid interface{}, opt *gitlab.GetProjectOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Project, *gitlab.Response, error)
}

// GitlabClient GitLab client configuration
type GitlabClient struct {
	Notes       GitlabNotesService
	Projects    GitlabProjectsService
	Context     context.Context
	Client      *gitlab.Client
	Config      config.NotifierConfig
	ProjectId   string
	NoteContent string
}

func NewGitlabClient(ctx context.Context, config config.NotifierConfig) *GitlabClient {
	c := &GitlabClient{
		Config:  config,
		Context: ctx,
	}
	if ctx == nil {
		c.Context = context.Background()
	}

	c.Client, _ = gitlab.NewClient(config.Token, gitlab.WithBaseURL(config.Url))

	if c.Notes == nil {
		c.Notes = c.Client.Notes
	}

	if c.Projects == nil {
		c.Projects = c.Client.Projects
	}

	return c
}

func convert(n *gitlab.Note) *Comment {
	var comment = &Comment{}
	if n == nil {
		return comment
	}

	comment.Id = int64(n.ID)
	comment.Body = n.Body

	return comment
}

func (gc *GitlabClient) GetProjectId() (string, error) {
	if gc.ProjectId != "" {
		return gc.ProjectId, nil
	}

	project, _, err := gc.Projects.GetProject(gc.Config.RepoOwner+"/"+gc.Config.RepoName, nil)

	if err != nil {
		return "", err
	}

	gc.ProjectId = fmt.Sprint(project.ID)
	return gc.ProjectId, nil
}

func (gc *GitlabClient) CreateComment() (*Comment, error) {
	projectId, err := gc.GetProjectId()
	if err != nil {
		return nil, err
	}
	note, _, err := gc.Notes.CreateMergeRequestNote(projectId, gc.Config.PullRequestID, &gitlab.CreateMergeRequestNoteOptions{Body: &gc.NoteContent})
	return convert(note), err
}

func (gc *GitlabClient) UpdateComment(id int64) (*Comment, error) {
	projectId, err := gc.GetProjectId()
	if err != nil {
		return nil, err
	}
	editedNote, _, err := gc.Notes.UpdateMergeRequestNote(projectId, gc.Config.PullRequestID, int(id), &gitlab.UpdateMergeRequestNoteOptions{Body: &gc.NoteContent})
	return convert(editedNote), err
}

func (gc *GitlabClient) DeleteComment(id int64) error {
	projectId, err := gc.GetProjectId()
	if err != nil {
		return err
	}
	_, err = gc.Notes.DeleteMergeRequestNote(projectId, gc.Config.PullRequestID, int(id))
	return err
}

func (gc *GitlabClient) GetCommentContent() string {
	return gc.NoteContent
}

func (gc *GitlabClient) PostComment() (CommentOperation, error) {
	return postComment(gc, gc.Config)
}

func (gc *GitlabClient) SetCommentContent(content string) {
	gc.NoteContent = content
}

func (gc *GitlabClient) ListComments() ([]Comment, error) {
	projectId, err := gc.GetProjectId()
	if err != nil {
		return nil, err
	}
	opt := &gitlab.ListMergeRequestNotesOptions{
		ListOptions: gitlab.ListOptions{PerPage: 100},
	}
	notes, _, err := gc.Notes.ListMergeRequestNotes(projectId, gc.Config.PullRequestID, opt)
	if err != nil {
		return nil, err
	}
	var result []Comment
	for _, note := range notes {
		result = append(result, *convert(note))
	}
	return result, nil
}
