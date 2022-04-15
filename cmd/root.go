package cmd

import (
	"fmt"
	"github.com/karlderkaefer/cdk-notifier/ci"
	"github.com/karlderkaefer/cdk-notifier/config"
	"github.com/karlderkaefer/cdk-notifier/transform"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io"
	"os"
)

var (
	v string
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
		appConfig := &config.NotifierConfig{}
		err := appConfig.Init()
		if err != nil {
			logrus.Fatal(err)
		}
		if appConfig.PullRequestID == 0 {
			err = &config.ValidationError{CliArg: "pull-request-id", EnvVar: config.EnvGithubPullRequestID}
			logrus.Warnf("Skipping... because %s", err)
			return
		}
		logrus.Tracef("got app config: %#v", appConfig)

		transformer := transform.NewLogTransformer(appConfig)
		transformer.Process()

		notifier, err := ci.GetNotifierService(cmd.Context(), appConfig)
		if err != nil {
			logrus.Fatalln(err)
		}
		notifier.SetCommentContent(transformer.LogContent)
		err = notifier.Authenticate()
		if err != nil {
			logrus.Fatalln(err)
		}
		err = notifier.PostComment()
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

	usageRepo := fmt.Sprintf("Name of github repository without organisation. If not set will lookup for env var '%s'", config.EnvGithubRepoName)
	usageOwner := fmt.Sprintf("Name of gitub owner. If not set will lookup for env var '%s'", config.EnvGithubRepoOwner)
	usageToken := fmt.Sprintf("Github token used to post comments to PR. If not set will lookup for env var '%s'", config.EnvGithubToken)
	usagePr := fmt.Sprintf("Id or URL of github pull request. If not set will lookup for env var '%s'", config.EnvGithubPullRequestID)

	rootCmd.Flags().StringP("repo", "r", "", usageRepo)
	rootCmd.Flags().StringP("owner", "o", "", usageOwner)
	rootCmd.Flags().String("token", "", usageToken)
	rootCmd.Flags().StringP("pull-request-id", "p", "", usagePr)
	rootCmd.Flags().StringP("log-file", "l", "", "path to cdk log file")
	rootCmd.Flags().StringP("tag-id", "t", "stack", "unique identifier for stack within pipeline")
	rootCmd.Flags().StringP("delete", "d", "", "delete comments when no changes are detected for a specific tag id")
	rootCmd.Flags().String("vcs", "github", "Version Control System [github|bitbucket]")

	// mapping for viper [mapstruct value, flag name]
	viperMappings := make(map[string]string)
	viperMappings["REPO_NAME"] = "repo"
	viperMappings["REPO_OWNER"] = "owner"
	viperMappings["TOKEN"] = "token"
	viperMappings["PR_ID"] = "pull-request-id"
	viperMappings["LOG_FILE"] = "log-file"
	viperMappings["TAG_ID"] = "tag-id"
	viperMappings["DELETE_COMMENT"] = "delete"
	viperMappings["VERSION_CONTROL_SYSTEM"] = "vcs"

	for k, v := range viperMappings {
		err := viper.BindPFlag(k, rootCmd.Flags().Lookup(v))
		if err != nil {
			logrus.Error(err)
		}
	}

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
