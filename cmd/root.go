package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/karlderkaefer/cdk-notifier/config"
	"github.com/karlderkaefer/cdk-notifier/provider"
	"github.com/karlderkaefer/cdk-notifier/transform"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	v string
	// Version cdk-notifier application version
	Version string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "cdk-notifier",
	Short:   "Post CDK diff log to Pull Request",
	Long:    "Post CDK diff log to Pull Request",
	Version: Version,
	Run: func(cmd *cobra.Command, args []string) {
		appConfig := &config.NotifierConfig{}
		err := appConfig.Init()
		if err != nil {
			logrus.Fatal(err)
		}

		transformer := transform.NewLogTransformer(appConfig)
		transformer.Process()

		if appConfig.NoPostMode {
			return
		}

		if appConfig.PullRequestID == 0 {
			err = &config.ValidationError{CliArg: "pull-request-id", EnvVar: []string{"PR_ID", config.EnvCiCircleCiPullRequestID, config.EnvCiBitbucketPrId, config.EnvCiGitlabMrId}}
			logrus.Warnf("Skipping... because %s", err)
			return
		}

		notifier, err := provider.CreateNotifierService(cmd.Context(), *appConfig)
		if err != nil {
			logrus.Fatalln(err)
		}
		notifier.SetCommentContent(transformer.LogContent)
		_, err = notifier.PostComment()
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

	usageRepo := fmt.Sprintf("Name of repository without organisation. If not set will lookup for env var [%s|%s|%s],'", "REPO_NAME", config.EnvCiCircleCiRepoName, config.EnvCiBitbucketRepoName)
	usageOwner := fmt.Sprintf("Name of owner. If not set will lookup for env var [%s|%s|%s]", "REPO_OWNER", config.EnvCiCircleCiRepoOwner, config.EnvCiBitbucketRepoOwner)
	usageToken := fmt.Sprintf("Authentication token used to post comments to PR. If not set will lookup for env var [%s|%s|%s|%s]", "TOKEN_USER", config.EnvGithubToken, config.EnvBitbucketToken, config.EnvGitlabToken)
	usagePr := fmt.Sprintf("Id or URL of pull request. If not set will lookup for env var [%s|%s|%s|%s]", "PR_ID", config.EnvCiCircleCiPullRequestID, config.EnvCiBitbucketPrId, config.EnvCiGitlabMrId)

	rootCmd.Flags().StringP("repo", "r", "", usageRepo)
	rootCmd.Flags().StringP("owner", "o", "", usageOwner)
	rootCmd.Flags().String("token", "", usageToken)
	rootCmd.Flags().StringP("pull-request-id", "p", "", usagePr)
	rootCmd.Flags().StringP("log-file", "l", "", "path to cdk log file")
	rootCmd.Flags().StringP("tag-id", "t", "stack", "unique identifier for stack within pipeline")
	rootCmd.Flags().BoolP("delete", "d", true, "delete comments when no changes are detected for a specific tag id")
	rootCmd.Flags().String("vcs", "github", "Version Control System [github|bitbucket|gitlab]")
	rootCmd.Flags().String("ci", "circleci", "CI System used [circleci|bitbucket|gitlab]")
	rootCmd.Flags().StringP("user", "u", "", "Optional set username for token (required for bitbucket)")
	rootCmd.Flags().String("gitlab-url", "https://gitlab.com/", "Optional set gitlab url")
	rootCmd.Flags().Bool("no-post-mode", false, "Optional do not post comment to VCS, instead write additional file and print diff to stdout")

	// mapping for viper [mapstruct value, flag name]
	viperMappings := make(map[string]string)
	viperMappings["REPO_NAME"] = "repo"
	viperMappings["REPO_OWNER"] = "owner"
	viperMappings["TOKEN"] = "token"
	viperMappings["TOKEN_USER"] = "user"
	viperMappings["PR_ID"] = "pull-request-id"
	viperMappings["LOG_FILE"] = "log-file"
	viperMappings["TAG_ID"] = "tag-id"
	viperMappings["DELETE_COMMENT"] = "delete"
	viperMappings["NO_POST_MODE"] = "no-post-mode"
	viperMappings["VERSION_CONTROL_SYSTEM"] = "vcs"
	viperMappings["CI_SYSTEM"] = "ci"
	viperMappings["URL"] = "gitlab-url"

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
