THE ONLY ACCEPTABLE PLACE FOR PROJECT_MANAGEMENT.md IS `.sgai/PROJECT_MANAGEMENT.md` -- never place `cmd/sgai/skel/.sgai/PROJECT_MANAGEMENT.md`.

Every time you are asked to make a source code (or prompt) modification  to `/.sgai` you have to make the modification to `cmd/sgai/skel/.sgai` instead.

In term of Go code style, I prefer total absence of inline comments; organize functions and if blocks in a way that they have intention revealing names, and use that instead.

In term of Go code style, I prefer very private functions over public functions; private struct over public structs; local types and structs over global structs; public functions and structs must have godoc comments.

In terms of Go code style, error variable names must use the err prefix pattern: errSpecificName (e.g., errClose, errRead), not the suffix pattern (closeErr, readErr).

Always run the code reviewer when one is available.

Always make sure you use tmux and playwright to test the changes.

For tests:
use listen address `-listen-addr 127.0.0.1:8181`
use directory ./verification

In terms of layout, UI, style, when something doesn't fit a container, use ellipsis with tooltip - refer to https://picocss.com/docs/tooltip


CRITICAL: it must be a pure HTMX and PicoCSS implementation, some light touch CSS to make looks be good is OK; some Javascript to hook up Idiomorph HTMX extension, may be acceptable upon human partner approval; otherwise, NEVER USE CUSTOM JAVASCRIPT.
CRITICAL: use playwright screenshots (and the skill to operate playwright) to verify the application is working correctly.

CRITICAL(code quality): ensure good Go code quality by calling `make lint`

You must use the skill `browser-bug-testing-workflow` - remember to use visual diffs and screenshots to evaluate the problem
You must use the skill `run-long-running-processes-in-tmux`


# Terminology

- Standalone Repository: a repository that has only _one_ `jj workspace` -- itself.
- Root Repository: a repository that has more than one `jj workspace`, and it is the root (it is the one in which `.jj/repo` is a directory and not a file)
- Forked Repository: a repository that is part of a `jj workspace, and it is not the root (it is the one in which `.jj/repo` is a text file, whose content points to the parent).

- Repository Mode: is when a repository is served by SGAI in a way that it can actually run software.
- Forked Mode: is when a root repository has at least one child, it displays the fork (dashboard-style) mode.
**CRITICAL** when a Root Repository run out of children, it must revert back from Forked Mode to Repository Mode.


# Safe Assumptions

"OpenCode" (aka `opencode`) is always installed and available.
