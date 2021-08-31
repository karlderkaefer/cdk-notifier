package cmd

import (
	"github.com/karlderkaefer/cdk-notifier/config"
	"github.com/karlderkaefer/cdk-notifier/github"
	"github.com/karlderkaefer/cdk-notifier/transform"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io"
	"os"
)

var (
	v             string
	logFile       string
	repoName      string
	repoOwner     string
	githubToken   string
	tagId         string
	pullRequestId int
	deleteComment bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cdk-notifier",
	Short: "Post CDK diff log to Github Pull Request",
	Long:  "Post CDK diff log to Github Pull Request",
	Run: func(cmd *cobra.Command, args []string) {
		appConfig := &config.AppConfig{
			LogFile:       logFile,
			TagId:         tagId,
			RepoName:      repoName,
			RepoOwner:     repoOwner,
			PullRequest:   pullRequestId,
			DeleteComment: deleteComment,
			GithubToken:   githubToken,
		}
		err := appConfig.Init()
		if err != nil {
			logrus.Fatal(err)
		}
		logrus.Debugf("got app config: %#v", appConfig)

		transformer := transform.NewLogTransformer(appConfig)
		transformer.Process()

		gc := github.NewGithubConfig(appConfig)
		gc.Context = cmd.Context()
		gc.CommentContent = transformer.LogContent
		gc.Authenticate()
		err = gc.PostComment()
		if err != nil {
			logrus.Fatalln(err)
		}
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		err := setUpLogs(os.Stdout, v)
		if err != nil {
			return err
		}
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&v, "verbosity", "v", logrus.InfoLevel.String(), "Log level (debug, info, warn, error, fatal, panic)")
	rootCmd.PersistentFlags().StringVarP(&repoName, "github-repo", "r", "", "Name of github repository without organisation. If not set will lookup for env var $CIRCLE_PROJECT_REPONAME")
	rootCmd.PersistentFlags().StringVarP(&repoOwner, "github-owner", "o", "", "Name of gitub owner. If not set will lookup for env var $CIRCLE_PROJECT_USERNAME")
	rootCmd.PersistentFlags().StringVar(&githubToken, "github-token", "", "Github token used to post comments to PR")
	rootCmd.PersistentFlags().IntVarP(&pullRequestId, "pull-request-id", "p", 23, "Id of github pull request. If not set will lookup for env var $CIRCLE_PR_NUMBER")
	rootCmd.PersistentFlags().StringVarP(&logFile, "log-file", "l", "./data/cdk-small.log", "path to cdk log file")
	rootCmd.PersistentFlags().StringVarP(&tagId, "tag-id", "t", "stack", "unique identifier for stack within pipeline")
	rootCmd.PersistentFlags().BoolVarP(&deleteComment, "delete", "d", true, "delete comments when no changes are detected for a specific tag id")
}

func setUpLogs(out io.Writer, level string) error {
	logrus.SetOutput(out)
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		return err
	}
	logrus.SetLevel(lvl)
	return nil
}
