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

		if appConfig.SuppressHashChanges {
			logrus.Warnf("Suppressing hash changes detected %d hash changes and %d total changes", transformer.HashChanges, transformer.TotalChanges)
			if transformer.TotalChanges == transformer.HashChanges {
				logrus.Warnf("Skipping... because suppress-hash-changes is set and only hash changes detected")
				return
			}
		}

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
	rootCmd.Flags().String("vcs", "github", "Version Control System [github|github-enterprise|bitbucket|gitlab]")
	rootCmd.Flags().String("ci", "circleci", "CI System used [circleci|bitbucket|gitlab]")
	rootCmd.Flags().StringP("user", "u", "", "Optional set username for token (required for bitbucket)")
	rootCmd.Flags().String("gitlab-url", "https://gitlab.com/", "Optional set gitlab url")
	rootCmd.Flags().String("github-host", "", "Optional set host for GitHub Enterprise")
	rootCmd.Flags().Int("github-max-comment-length", 0, "Optional set max comment length for GitHub Enterprise")
	rootCmd.Flags().Bool("no-post-mode", false, "Optional do not post comment to VCS, instead write additional file and print diff to stdout")
	rootCmd.Flags().Bool("disable-collapse", false, "Collapsible comments are enabled by default for GitHub and GitLab. When set to true it will not use collapsed sections.")
	rootCmd.Flags().Bool("show-overview", false, "[Deprected: use template extended instead] Show Overview are disabled by default. When set to true it will show the number of cdk stacks with diff and  the number of replaced resources in the overview section.")
	rootCmd.Flags().String("template", "default", "Template to use for comment [default|extended|extendedWithResources]")
	rootCmd.Flags().String("custom-template", "", "File path or string input to custom template. When set it will override the template flag.")
	rootCmd.Flags().Bool("suppress-hash-changes", false, "EXPERIMENTAL: when set to true it will ignore changes in hash values")
	rootCmd.Flags().String("suppress-hash-changes-regex", config.DefaultSuppressHashChangesRegex, "Define Regex to suppress hash changes. Only used when suppress-hash-changes is set to true")

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
	viperMappings["DISABLE_COLLAPSE"] = "disable-collapse"
	// TODO show overview deprecated
	viperMappings["SHOW_OVERVIEW"] = "show-overview"
	viperMappings["NOTIFIER_TEMPLATE"] = "template"
	viperMappings["CUSTOM_TEMPLATE"] = "custom-template"
	viperMappings["VERSION_CONTROL_SYSTEM"] = "vcs"
	viperMappings["CI_SYSTEM"] = "ci"
	viperMappings["URL"] = "gitlab-url"
	viperMappings["GITHUB_ENTERPRISE_HOST"] = "github-host"
	viperMappings["GITHUB_ENTERPRISE_MAX_COMMENT_LENGTH"] = "github-max-comment-length"
	viperMappings["SUPPRESS_HASH_CHANGES"] = "suppress-hash-changes"
	viperMappings["SUPPRESS_HASH_CHANGES_REGEX"] = "suppress-hash-changes-regex"

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
