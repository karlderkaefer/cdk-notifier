# cdk-notifier
lightweight CLI tool to parse a CDK log file and post changes to pull request requests.
Can be used to get more confidence on approving pull requests because reviewer will be aware of changes done to your environments.

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
#  -d, --delete                delete comments when no changes are detected for a specific tag id (default true)
#  -o, --github-owner string   Name of gitub owner. If not set will lookup for env var $CIRCLE_PROJECT_USERNAME
#  -r, --github-repo string    Name of github repository without organisation. If not set will lookup for env var $CIRCLE_PROJECT_REPONAME
#      --github-token string   Github token used to post comments to PR
#  -h, --help                  help for cdk-notifier
#  -l, --log-file string       path to cdk log file (default "./data/cdk-small.log")
#  -p, --pull-request-id int   Id of github pull request. If not set will lookup for env var $CIRCLE_PR_NUMBER (default 23)
#  -t, --tag-id string         unique identifier for stack within pipeline (default "stack")
#  -v, --verbosity string      Log level (debug, info, warn, error, fatal, panic) (default "info")
#      --version               version for cdk-notifier
```

## Usage
First create the output of cdk diff to file. You can stream cdk diff to stdout and to file with following.
This tool most like runs in a CI environment. To avoid [printing millions of lines](https://github.com/aws/aws-cdk/issues/8893#issuecomment-654296389) you add progress flag.
```bash
cdk diff --progress=events &> >(tee cdk.log)
```
cdk-notifier will then analyze and transform the log by
* remove ASCII colors
* prepare additions and deletion for GitHub markdown diff
* truncate log if exceeding [max length of body for comment](https://github.community/t/maximum-length-for-the-comment-body-in-issues-and-pr/148867/2)
and then send
  
cdk-notifier will post the processed log of cdk diff to PR if there are changes.
If a diff comment for tag-id exists and no changes are detected then comment will delete. 
You can control this behavior with `--delete false`.

```bash
cdk-notfier --github-owner some-org --githhub-repo some-repo --github-token 1234 --log-file ./cdk.log --tag-id my-stack
```
The `tag-id` has to be unique within one pipeline. It's been used to identify the comment to update or delete.

## Running on CirlceCI
If you run cdk-notifier on CircleCi you dont need to set owner, repo or token. 
CircleCi will provide default variables which will read in by cdk-notifier when cli arg is not set.
```bash
CIRCLE_PR_NUMBER
CIRCLE_PROJECT_REPONAME
```
but you need the add environment variable for github token or set CLi arg `github-token`.
```bash
GITHUB_TOKEN
```

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


