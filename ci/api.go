package ci

import (
	"context"
	"errors"
	"fmt"
	"github.com/karlderkaefer/cdk-notifier/config"
	"regexp"
)

// NotifierService interface for public methods actions required for cdk-notifier
type NotifierService interface {
	Authenticate() error
	PostComment() error
	SetCommentContent(content string)
}

func getHeaderTagID(c *config.NotifierConfig) string {
	return fmt.Sprintf("%s %s", HeaderPrefix, c.TagID)
}

func diffHasChanges(log string) bool {
	regex := regexp.MustCompile(`(?m)(Policy Changes|Resources\n|Statement Changes)`)
	return regex.MatchString(log)
}

// GetNotifierService will create an client instance depending on type of ci parameters
func GetNotifierService(ctx context.Context, config *config.NotifierConfig) (NotifierService, error) {
	switch config.Vcs {
	case "github":
		return NewGithubClient(ctx, config), nil
	case "bitbucket":
		return nil, errors.New("not Implemented")
	default:
		return nil, fmt.Errorf("unspported Version Control System: %s", config.Vcs)
	}
}
