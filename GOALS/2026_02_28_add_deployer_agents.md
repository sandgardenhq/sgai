---
flow: |
  "backend-go-developer" -> "go-readability-reviewer"
  "backend-go-developer" -> "stpa-analyst"
  "go-readability-reviewer" -> "stpa-analyst"
  "general-purpose" -> "stpa-analyst"
  "react-developer" -> "react-reviewer"
  "react-reviewer" -> "stpa-analyst"
  "project-critic-council"
  "skill-writer"
models:
  "coordinator": "anthropic/claude-opus-4-6 (max)"
  "backend-go-developer": "anthropic/claude-opus-4-6"
  "go-readability-reviewer": "anthropic/claude-opus-4-6"
  "general-purpose": "anthropic/claude-opus-4-6"
  "react-developer": "anthropic/claude-opus-4-6"
  "react-reviewer": "anthropic/claude-opus-4-6"
  "stpa-analyst": "anthropic/claude-opus-4-6"
  "project-critic-council": ["anthropic/claude-opus-4-6","anthropic/claude-sonnet-4-6", "anthropic/claude-opus-4-5"]
  "skill-writer": "anthropic/claude-opus-4-6 (max)"
completionGateScript: make test
---

- [x] I need agents capable of setting up and deploying applications
- [x] `vercel-deployer` refer to https://vercel.com/docs/deployments
- [x] `cloudflare-worker-deployer` refer to https://developers.cloudflare.com/workers/configuration/versions-and-deployments/
- [x] `exe-dev-deployer` refer to https://exe.dev/docs/proxy and https://exe.dev/docs/faq/copy-files and https://exe.dev/docs/section/8-cli-reference
