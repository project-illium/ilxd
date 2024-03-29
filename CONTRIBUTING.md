# Contribute

Ilxd is an open source project. We love contributions! This document outlines some contribution guidelines so that your contributions can be included in the codebase.

## Issues

Issues should be used primarily for bug reports and directly actionable features. Discussion of the protocol or new features should take place on the Github [discussions page](https://github.com/project-illium/ilxd/discussions).

## Go Guidelines
You must run `gofmt` before each commit. The ci tests will fail if you do not run it. Most IDEs have the ability to set `gofmt` to run on save or at specified times.

All commits are checked with [golangci-lint](https://github.com/golangci/golangci-lint) using the [.golangci.yml](.golangci.yml) config in the repo.

## Tests
If you add new code, please submit a unit test with it. We might not accept the PR without it. Additionally, you are expected to make the appropriate changes to existing tests if they are affected by your commits.

## Pull Requests
If your PR isn't ready to merge make sure you specify this somehow. For example by placing [WIP] in the PR title. Ideally you should include a `task list` in the PR message to track the progress of the PR.

The PR must be approved by more than one member of the team with write access prior to merging. 

## Comments
To keep things consistent comment fragments should start with a capital letter and end with no period. If the comment is one or more full sentences (a sentence has at least a subject and a verb) then the sentences should end with a period.

## Commits
Please keep all of your commits [atomic](https://www.freshconsulting.com/atomic-commits/).

Also, be sure to follow the [seven rules of a great git commit message](http://chris.beams.io/posts/git-commit/).

```
1. Separate subject from body with a blank line
2. Limit the subject line to 50 characters
3. Capitalize the subject line
4. Do not end the subject line with a period
5. Use the imperative mood in the subject line
6. Wrap the body at 72 characters
7. Use the body to explain what and why vs. how
```

Finally, please [sign](https://help.github.com/articles/signing-commits-using-gpg/) your commits. 
