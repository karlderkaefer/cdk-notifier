package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/napalm684/cdk-notifier/config"
	"github.com/napalm684/cdk-notifier/gitlab"
	"github.com/napalm684/cdk-notifier/transform"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	v              string
	baseURL        string
	logFile        string
	gitlabToken    string
	tagID          string
	mergeRequestID int
	deleteNote     bool
	projectID      int
	// Version cdk-notifier application version
	Version string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "cdk-notifier",
	Short:   "Post CDK diff log to Gitlab Merge Request",
	Long:    "Post CDK diff log to Gitlab Merge Request",
	Version: Version,
	Run: func(cmd *cobra.Command, args []string) {
		appConfig := &config.AppConfig{
			LogFile:      logFile,
			TagID:        tagID,
			MergeRequest: mergeRequestID,
			DeleteNote:   deleteNote,
			GitlabToken:  gitlabToken,
			GitlabUrl:    baseURL,
			ProjectID:    projectID,
		}
		err := appConfig.Init()
		if err != nil {
			logrus.Fatal(err)
		}
		if appConfig.MergeRequest == 0 {
			err = &config.ValidationError{CliArg: "pull-request-id", EnvVar: config.EnvMergeRequestID}
			logrus.Warnf("Skipping... because %s", err)
			return
		}
		logrus.Tracef("got app config: %#v", appConfig)

		transformer := transform.NewLogTransformer(appConfig)
		transformer.Process()

		gc := gitlab.NewGitlabClient(appConfig, nil)
		gc.NoteContent = transformer.LogContent
		gc.Authenticate()
		err = gc.CreateMergeRequestNote()
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
	usageUrl := fmt.Sprintf("Base url for gitlab account. If not set will lookup for env var '%s'", config.EnvGitlabUrl)
	usageToken := fmt.Sprintf("Gitlab token used to post comments to PR. If not set will lookup for env var '%s'", config.EnvGitlabToken)
	usageMr := fmt.Sprintf("Id of gitlab merge request. If not set will lookup for env var '%s'", config.EnvMergeRequestID)
	usagePid := fmt.Sprintf("Project Id of gitlab project that has merge request. If not set will lookup for env var '%s'", config.EnvGitlabPid)
	rootCmd.PersistentFlags().StringVarP(&baseURL, "url", "u", "https://gitlab.com/", usageUrl)
	rootCmd.PersistentFlags().StringVar(&gitlabToken, "gitlab-token", "", usageToken)
	rootCmd.PersistentFlags().IntVarP(&mergeRequestID, "merge-request-id", "m", 0, usageMr)
	rootCmd.PersistentFlags().StringVarP(&logFile, "log-file", "l", "./cdk.log", "path to cdk log file")
	rootCmd.PersistentFlags().StringVarP(&tagID, "tag-id", "t", "stack", "unique identifier for stack within pipeline")
	rootCmd.PersistentFlags().BoolVarP(&deleteNote, "delete", "d", true, "delete notes when no changes are detected for a specific tag id")
	rootCmd.PersistentFlags().IntVarP(&projectID, "gitlab-pid", "p", 0, usagePid)
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
