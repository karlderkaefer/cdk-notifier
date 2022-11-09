package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/karlderkaefer/cdk-notifier/config"
	"github.com/sirupsen/logrus"
)

const (
	API_COMMENT_CREATED CommentOperation = iota
	API_COMMENT_UPDATED
	API_COMMENT_DELETED
	API_COMMENT_NOTHING
	userAgent = "cdk-notifier"
	// HeaderPrefix default prefix for comment message
	HeaderPrefix = "## cdk diff for"
)

type CommentOperation int

func (d CommentOperation) String() string {
	return [...]string{"CREATED", "UPDATED", "DELETED", "NOTHING"}[d]
}

type Comment struct {
	Id   int64
	Body string
	Link string
}

// NotifierService interface for public methods actions required for cdk-notifier
type NotifierService interface {
	CreateComment() (*Comment, error)
	UpdateComment(id int64) (*Comment, error)
	DeleteComment(id int64) error
	SetCommentContent(content string)
	GetCommentContent() string
	PostComment() (CommentOperation, error)
	ListComments() ([]Comment, error)
}

func getHeaderTagID(c config.NotifierConfig) string {
	return fmt.Sprintf("%s %s", HeaderPrefix, c.TagID)
}

func diffHasChanges(log string) bool {
	regex := regexp.MustCompile(`(?m)(Policy Changes|Resources\n|Statement Changes)`)
	return regex.MatchString(log)
}

// CreateNotifierService will create an client instance depending on type of ci parameters
func CreateNotifierService(ctx context.Context, c config.NotifierConfig) (NotifierService, error) {
	switch c.Vcs {
	case config.VcsGithub, config.VcsGithubEnterprise:
		return NewGithubClient(ctx, c)
	case config.VcsBitbucket:
		return NewBitbucketProvider(ctx, c), nil
	case config.VcsGitlab:
		return NewGitlabClient(ctx, c), nil
	default:
		return nil, fmt.Errorf("unspported Version Control System: %s", c.Vcs)
	}
}

// postComment contains business logic how to create, update or delete comments
// PostComment will create GitHub comment if comment does not exist yet bases on FindComment
// If the comment already exist the content will be updated.
// If there are no cdk differences the comment will be deleted depending on DeleteComment config.AppConfig
func postComment(ns NotifierService, config config.NotifierConfig) (CommentOperation, error) {
	comment, err := findComment(ns, config)
	if err != nil {
		return API_COMMENT_NOTHING, err
	}
	if comment != nil {
		// if commit exists but there are no change then delete comment in case DeleteComment is active
		if config.DeleteComment && !diffHasChanges(ns.GetCommentContent()) {
			err = ns.DeleteComment(comment.Id)
			if err != nil {
				logrus.Error(err)
				return API_COMMENT_NOTHING, err
			}
			logrus.Infof("Deleted comment with id %d and tag id %s because no changes detected", comment.Id, config.TagID)
			return API_COMMENT_DELETED, nil
		}
		// if comment exists and there are diff then update existing comment
		comment, err = ns.UpdateComment(comment.Id)
		if err != nil {
			logrus.Error(err)
			return API_COMMENT_NOTHING, err
		}
		logrus.Infof("Updated comment with id %d and tag id %s %v", comment.Id, config.TagID, comment.Link)
		return API_COMMENT_UPDATED, nil
	}
	if !diffHasChanges(ns.GetCommentContent()) {
		logrus.Infof("There is no diff detected for tag id %s. Skip posting diff.", config.TagID)
		return API_COMMENT_NOTHING, nil
	}
	comment, err = ns.CreateComment()
	if err != nil {
		logrus.Error(err)
		return API_COMMENT_NOTHING, err
	}
	logrus.Infof("Created comment with id %d and tag id %s %v", comment.Id, config.TagID, comment.Link)
	return API_COMMENT_CREATED, nil
}

// findComment is finding the comment containing the cdk stack id
func findComment(ns NotifierService, config config.NotifierConfig) (*Comment, error) {
	comments, err := ns.ListComments()
	if err != nil {
		return nil, err
	}
	for _, comment := range comments {
		if strings.Contains(comment.Body, getHeaderTagID(config)) {
			logrus.Debugf("Found existing comment for %s", config.TagID)
			return &comment, nil
		}
	}
	return nil, nil
}
