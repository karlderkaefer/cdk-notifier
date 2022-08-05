# cdk-notifier

[![CircleCI](https://circleci.com/gh/circleci/circleci-docs.svg?style=shield)](https://circleci.com/gh/circleci/circleci-docs)
[![codecov](https://codecov.io/gh/karlderkaefer/cdk-notifier/branch/main/graph/badge.svg?token=C0BGW4EUOX)](https://codecov.io/gh/karlderkaefer/cdk-notifier)
[![Go Report Card](https://goreportcard.com/badge/github.com/karlderkaefer/cdk-notifier)](https://goreportcard.com/report/github.com/karlderkaefer/cdk-notifier)

lightweight CLI tool to parse a CDK log file and post changes to pull request requests.
Can be used to get more confidence on approving pull requests because reviewer will be aware of changes done to your
environments.

[Medium Article](https://betterprogramming.pub/improve-your-pull-request-experience-for-aws-cdk-projects-1fd5adb08bb3)

## Install

Install binary with latest release

```bash
curl -L "https://github.com/karlderkaefer/cdk-notifier/releases/latest/download/cdk-notifier_$(uname)_amd64.gz" -o cdk-notifier.gz
gunzip cdk-notifier.gz && chmod +x cdk-notifier && rm -rf cdk-notifier.gz
sudo mv cdk-notifier /usr/local/bin/cdk-notifier
```

Check Version and help

```bash
cdk-notifier --version
# 1.0.1

cdk-notifier --help
#Post CDK diff log to Github Pull Request
#
#Usage:
#  cdk-notifier [flags]
#
#Flags:
#      --ci string                CI System used [circleci|bitbucket|gitlab] (default "circleci")
#  -d, --delete string            delete comments when no changes are detected for a specific tag id
#  -h, --help                     help for cdk-notifier
#  -l, --log-file string          path to cdk log file
#  -o, --owner string             Name of owner. If not set will lookup for env var [REPO_OWNER|CIRCLE_PROJECT_USERNAME|BITBUCKET_REPO_OWNER|CI_PROJECT_NAMESPACE]
#  -p, --pull-request-id string   Id or URL of pull request. If not set will lookup for env var [PR_ID|CIRCLE_PULL_REQUEST|BITBUCKET_PR_ID|CI_MERGE_REQUEST_IID]
#  -r, --repo string              Name of repository without organisation. If not set will lookup for env var [REPO_NAME|CIRCLE_PROJECT_REPONAME|BITBUCKET_REPO_SLUG|CI_PROJECT_NAME],'
#  -t, --tag-id string            unique identifier for stack within pipeline (default "stack")
#      --token string             Authentication token used to post comments to PR. If not set will lookup for env var [TOKEN_USER|GITHUB_TOKEN|BITBUCKET_TOKEN|GITLAB_TOKEN]
#  -u, --user string              Optional set username for token (required for bitbucket)
#      --vcs string               Version Control System [github|bitbucket|gitlab] (default "github")
#  -v, --verbosity string         Log level (debug, info, warn, error, fatal, panic) (default "info")
#      --version                  version for cdk-notifier

```

## Usage

First create the output of cdk diff to file. You can stream cdk diff to stdout and to file with following.
This tool most like runs in a CI environment. To
avoid [printing millions of lines](https://github.com/aws/aws-cdk/issues/8893#issuecomment-654296389) you add progress
flag.

```bash
cdk diff --progress=events &> >(tee cdk.log)
```

cdk-notifier will then analyze and transform the log by

* remove ASCII colors
* prepare additions and deletion for GitHub markdown diff
* truncate log if
  exceeding [max length of body for comment](https://github.community/t/maximum-length-for-the-comment-body-in-issues-and-pr/148867/2)
  and then send

cdk-notifier will post the processed log of cdk diff to PR if there are changes.
If a diff comment for tag-id exists and no changes are detected then comment will delete.
You can control this behavior with `--delete false`.

```bash
cdk-notfier --owner some-org --repo some-repo --token 1234 --log-file ./cdk.log --tag-id my-stack --pull-request-id 12 --vcs github --ci circleci
```

The `tag-id` has to be unique within one pipeline. It's been used to identify the comment to update or delete.

This is an example how the diff would like on github

```bash
cdk-notifier -l data/cdk-small.log -t test
```

![](images/diff.png)

## Support for CI Systems

CDK-Notifier is supporting following Version Control Systems

* github
* bitbucket
* gitlab

If you run CDK-Notifier on CI Systems, you may not need to set flag for `owner`, `repo` or `pull-request-id`.
Those will be read in automatically if not set via cli args. See [priority mapping](#config-priority-mapping).
Following matrix is showing support for automatic mapping for different CI Systems.

| Version Control System | CirlceCi Support   | Bitbucket CI Support | Github CI Support | Gitlab CI Support  |
|------------------------|--------------------|----------------------|-------------------|--------------------|
| github                 | :heavy_check_mark: | :heavy_check_mark:   | :x:               | :x:                |
| bitbucket              | :heavy_check_mark: | :heavy_check_mark:   | :x:               | :x:                |
| gitlab                 | :x:                | :x:                  | :x:               | :heavy_check_mark: |

If you run cdk-notifier on CircleCi you don't need to set owner, repo or token.
CircleCi will provide default variables which will read in by cdk-notifier when cli arg is not set.

Example when running on CircleCi. See [available build variables](https://circleci.com/docs/env-vars#built-in-environment-variables)
```bash
CIRCLE_PR_NUMBER
CIRCLE_PROJECT_REPONAME
CIRCLE_PROJECT_USERNAME
```

Example when running on BitBucket CI. See [available build variables](https://support.atlassian.com/bitbucket-cloud/docs/variables-and-secrets/)
```bash
BITBUCKET_PR_ID
BITBUCKET_REPO_OWNER
BITBUCKET_REPO_SLUG
```

Example when running on Gitlab CI. See [available build variables](https://docs.gitlab.com/ee/ci/variables/predefined_variables.html)
```bash
CI_MERGE_REQUEST_IID
CI_PROJECT_NAMESPACE
CI_PROJECT_NAME
```

Token and usernames will be read in automatically despite on which CI they run. Potentially they override each other in order listed below.

```bash
TOKEN
GITHUB_TOKEN
BITBUCKET_TOKEN
GITLAB_TOKEN
```

## Config Priority Mapping
The config for CDK-Notifier is mapping in following priority (from low to high)
1. Environment Variables of Map Struct
    ```
    REPO_NAME
    REPO_OWNER
    TOKEN
    TOKEN_USER
    PR_ID
    LOG_FILE
    TAG_ID
    DELETE_COMMENT
    VERSION_CONTROL_SYSTEM
    CI_SYSTEM
    ```
2. CI System specific environment variable mapping. See [support-for-ci-systems](#support-for-ci-systems)
3. Default values for CLI args. See `cdk-notifier --help`
4. Values set by CLI e.g. `--token`

## Security
**Disclaimer**: Consider using on private repositories only.

The CDK log does not contain sensitive information by default. However, someone can argue the account id is considered as sensitive information.
CDK-Notifier does also not benefit from the automatic obscuring you may see in CI logs when using secret environment variables. 

## Versioning

Use [Conventional Commit Messages](https://www.conventionalcommits.org/en/v1.0.0/).
[Semantic Release](https://github.com/semantic-release/semantic-release) will release a new version with changelog.

examples:

``` 
# increase patch version
fix: fixing tests

# incease minor version
feat: add configuration

# increase major version:
feat: remove comments api

BREAKING CHANGE: remove comments api

# update docu
docs: update readme
```


