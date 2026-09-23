# Set up this GitHub CLI fork

This fork adds repository-specific GitHub account selection. The normal GitHub
CLI release does not include this fork's change.

## Install the forked CLI

Build this repository with the Go version declared in `go.mod`:

```sh
git clone https://github.com/whoant/cli.git
cd cli
make install prefix="$HOME/.local"
export PATH="$HOME/.local/bin:$PATH"
command -v gh
gh version
```

Put `$HOME/.local/bin` before other `gh` installations in your shell's permanent
`PATH` setting if you want this fork to be the default in new terminals. You can
also use `./bin/gh` from this checkout without installing it.

## Choose an account for each repository

Log in to each account once with the forked `gh` binary. Check that both appear
in `gh auth status`. Then set `github.account` within each Git repository:

```sh
cd /path/to/personal/repo
git config --local github.account whoant
gh api user --jq .login

cd /path/to/work/repo
git config --local github.account YOUR_WORK_USERNAME
gh api user --jq .login
```

`gh api`, `gh pr`, `gh issue`, and `gh repo` use the selected account's stored
token for their target host. The active account shown by `gh auth status` does
not change. `GH_TOKEN` and `GITHUB_TOKEN` take precedence for github.com;
`GH_ENTERPRISE_TOKEN` and `GITHUB_ENTERPRISE_TOKEN` take precedence for
GitHub Enterprise hosts. If `github.account` is unset, the active account is
used. To remove a repository's choice, run:

```sh
git config --local --unset github.account
```

The account must already have a stored token for the command's target host.
If it does not, the command reports an error naming the account and host.

## Merge updates from the original GitHub CLI

After this fork's changes are merged into `trunk`, the fork has its own commits.
Update it by merging `cli/cli`'s `trunk` into the fork's `trunk`:

```sh
cd /path/to/your/cli-checkout
git remote add upstream https://github.com/cli/cli.git # run once
git switch trunk
git pull --ff-only origin trunk
git fetch upstream trunk
git merge --no-edit upstream/trunk
git -c credential.helper='!gh auth git-credential' push origin trunk
```

The account used to push must have write access to `whoant/cli`. If Git reports
a merge conflict, resolve it before pushing. Do not force-sync the fork with
upstream, because that would discard this fork's commits.

Updating the GitHub repository does not replace the `gh` binary on your
computer. Rebuild and reinstall after merging new upstream commits:

```sh
make install prefix="$HOME/.local"
gh version
```
