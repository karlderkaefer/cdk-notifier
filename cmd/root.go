package cmd

import (
	"fmt"
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
	tagID         string
	pullRequestID int
	deleteComment bool
	// Version cdk-notifier application version
	Version string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "cdk-notifier",
	Short:   "Post CDK diff log to Github Pull Request",
	Long:    "Post CDK diff log to Github Pull Request",
	Version: Version,
	Run: func(cmd *cobra.Command, args []string) {
		appConfig := &config.AppConfig{
			LogFile:       logFile,
			TagID:         tagID,
			RepoName:      repoName,
			RepoOwner:     repoOwner,
			PullRequest:   pullRequestID,
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

		gc := github.NewGithubClient(cmd.Context(), appConfig, nil)
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
	usageRepo := fmt.Sprintf("Name of github repository without organisation. If not set will lookup for env var '%s'", config.EnvRepoName)
	usageOwner := fmt.Sprintf("Name of gitub owner. If not set will lookup for env var '%s'", config.EnvRepoOwner)
	usageToken := fmt.Sprintf("Github token used to post comments to PR. If not set will lookup for env var '%s'", config.EnvGithubToken)
	usagePr := fmt.Sprintf("Id or URL of github pull request. If not set will lookup for env var '%s'", config.EnvPullRequestID)
	rootCmd.PersistentFlags().StringVarP(&repoName, "github-repo", "r", "", usageRepo)
	rootCmd.PersistentFlags().StringVarP(&repoOwner, "github-owner", "o", "", usageOwner)
	rootCmd.PersistentFlags().StringVar(&githubToken, "github-token", "", usageToken)
	rootCmd.PersistentFlags().IntVarP(&pullRequestID, "pull-request-id", "p", 0, usagePr)
	rootCmd.PersistentFlags().StringVarP(&logFile, "log-file", "l", "./cdk.log", "path to cdk log file")
	rootCmd.PersistentFlags().StringVarP(&tagID, "tag-id", "t", "stack", "unique identifier for stack within pipeline")
	rootCmd.PersistentFlags().BoolVarP(&deleteComment, "delete", "d", true, "delete comments when no changes are detected for a specific tag id")
	if Version == "" {
		rootCmd.Version = "dev"
	}
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
