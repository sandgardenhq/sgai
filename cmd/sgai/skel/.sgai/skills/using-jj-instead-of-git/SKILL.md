---
name: using-jj-instead-of-git
description: Use jj instead of git for version control operations
when_to_use: When you need to perform version control tasks like committing, branching, merging, etc. Traps git use cases to steer to jj.
version: 2.0.0
languages: all
---

# Using jj instead of Git

## Overview
Use Jujutsu (jj) instead of Git for all version control operations. jj provides a more intuitive and powerful interface while maintaining compatibility with Git repositories.

## When to Use
When you need to:
- Initialize a repository
- Commit changes
- View status or history
- Branch, merge, or rebase
- Push/pull to remotes
- Any git command

Instead of using git, use the jj equivalents.

## Core Pattern
Map git commands to jj commands using this table:

| Git Command | JJ Command | Notes |
|-------------|------------|-------|
| git init | jj git init [--no-colocate] | |
| git clone <source> <dest> | jj git clone <source> <dest> | |
| git status | jj st | |
| git diff | jj diff | |
| git diff HEAD | jj diff | |
| git diff <rev>^ <rev> | jj diff -r <rev> | |
| git diff --from A --to B | jj diff --from A --to B | |
| git add | (automatic) | jj tracks changes automatically |
| git commit | jj commit | |
| git commit -a | jj commit | |
| git commit --amend | jj squash | |
| git commit --amend --only | jj describe @- | Edit previous commit message |
| git log | jj log | |
| git log --oneline --graph | jj log -r ::@ | |
| git log --oneline --graph --all | jj log -r 'all()' | or jj log -r :: |
| git show <rev> | jj show <rev> | |
| git branch | jj bookmark list | or jj b l |
| git branch <name> <rev> | jj bookmark create <name> -r <rev> | |
| git branch -f <name> <rev> | jj bookmark move <name> --to <rev> | or jj b m <name> -t <rev> |
| git branch --delete <name> | jj bookmark delete <name> | |
| git checkout -b topic main | jj new main | |
| git merge A | jj new @ A | |
| git rebase B A | jj rebase -b A -o B | |
| git rebase --onto B A^ <bookmark> | jj rebase -s A -o B | |
| git rebase -i A | jj rebase -r C --before B | For reordering |
| git push | jj git push | |
| git push --all | jj git push --all | |
| git push <remote> <bookmark> | jj git push --bookmark <bookmark> | |
| git fetch | jj git fetch | |
| git pull | jj git fetch | Then jj new <remote bookmark> if needed |
| git remote add <remote> <url> | jj git remote add <remote> <url> | |
| git reset --hard | jj abandon | Abandon current change |
| git reset --hard | jj restore | Make current change empty |
| git reset --soft HEAD~ | jj squash --from @- | Keep diff in working copy |
| git restore <paths> | jj restore <paths> | |
| git stash | jj new @- | Old commit remains as sibling |
| git cherry-pick <source> | jj duplicate <source> -o @ | |
| git rev-parse --show-toplevel | jj workspace root | |
| git blame <file> | jj file annotate <path> | |
| git ls-files --cached | jj file list | |
| git rm --cached <file> | jj file untrack <file> | Must match ignore pattern |
| git revert <rev> | jj revert -r <rev> -B @ | |

For full table, see https://docs.jj-vcs.dev/latest/git-command-table/

## Quick Reference

### Basic Operations
- To see status: jj st
- To see changes: jj diff
- To see history: jj log
- To commit: jj commit -m 'message'
- To amend: jj squash

### Branching (Bookmarks)
- List bookmarks: jj bookmark list (or jj b l)
- Create bookmark: jj bookmark create <name> -r <revision>
- Move bookmark: jj bookmark move <name> --to <revision>
- Delete bookmark: jj bookmark delete <name>

### Working with Changes
- Start new change: jj new
- Start new change from specific revision: jj new <revision>
- Edit description: jj describe
- Squash into parent: jj squash
- Split a change: jj split
- Interactive diff edit: jj diffedit -r <revision>

### Remote Operations
- Fetch from remote: jj git fetch
- Push to remote: jj git push
- Push specific bookmark: jj git push --bookmark <name>

### Undo Operations
- Undo last operation: jj undo
- See operation log: jj op log

## Common Mistakes
- Trying to use git commands - use jj instead
- Forgetting jj tracks files automatically, no need for add
- Forgetting to set -m for jj commit when you want to avoid the editor
- Using -d instead of -o for destination (jj uses -o for --onto)

## Key Differences from Git
- jj tracks changes automatically - no need for "git add"
- Operations don't get interrupted for conflicts - resolve conflicts then jj squash
- Every commit is a "change" that can be edited at any time
- jj has an operation log (jj op log) that tracks all repo operations
- jj undo can undo almost any operation

## Real-World Impact
jj allows concurrent editing, better conflict resolution, and stacked diffs.
