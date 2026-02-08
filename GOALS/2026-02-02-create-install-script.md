---
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "backend-go-developer" -> "stpa-analyst"
  "go-readability-reviewer" -> "stpa-analyst"
  "general-purpose" -> "stpa-analyst"
  "htmx-picocss-frontend-developer" -> "htmx-picocss-frontend-reviewer"
  "htmx-picocss-frontend-reviewer" -> "stpa-analyst"
models:
  "coordinator": "anthropic/claude-opus-4-5 (max)"
  "backend-go-developer": "anthropic/claude-opus-4-5"
  "go-readability-reviewer": "anthropic/claude-opus-4-5"
  "general-purpose": "anthropic/claude-opus-4-5"
  "htmx-picocss-frontend-developer": "anthropic/claude-sonnet-4-5"
  "htmx-picocss-frontend-reviewer": "anthropic/claude-opus-4-5"
  "stpa-analyst": "anthropic/claude-opus-4-5"
interactive: yes
---

# Create curl-pipe-bash installer for sgai
Build a cross-platform installation script (`install.sh`) that allows users to install sgai with:

```sh
curl -fsSL https://raw.githubusercontent.com/sandgardenhq/sgai/main/install.sh | bash
```

  ## Requirements
   • Detect OS (macOS, Linux) and architecture (amd64, arm64) automatically
   • Download pre-built binary from GitHub Releases for the detected platform
   • Install to /usr/local/bin or ~/.local/bin with graceful permission handling
   • Verify binary checksum before installation
   • Support --version <tag> flag (default: latest)
   • Support --uninstall flag
   • Use strict error handling (set -euo pipefail) and cleanup on exit
   • Provide clear success/failure messages with next steps

  ## Out of Scope

   • Windows support
   • Automatic prerequisite installation (opencode, jj, graphviz)
   • GitHub Release automation (separate task)

  ## Acceptance Criteria

   • [x] Works on macOS Intel, macOS ARM, Linux amd64, Linux arm64
   • [x] Fails gracefully with clear message on unsupported platforms
   • [x] Idempotent — safe to run multiple times
   • [x] Prompts before overwriting existing installation