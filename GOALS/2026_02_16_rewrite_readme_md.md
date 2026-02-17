---
flow: |
  "general-purpose"
  "project-critic-council"
models:
  "coordinator": "anthropic/claude-opus-4-6 (max)"
  "general-purpose": "anthropic/claude-opus-4-6"
  "project-critic-council": ["anthropic/claude-opus-4-6", "anthropic/claude-opus-4-6 (max)"]
interactive: yes
---

# Rewrite README.md — Clearer Conceptual Overview and Simplified Instructions

Closes https://github.com/sandgardenhq/sgai/issues/66

Restructure and rewrite `README.md` so new users understand what sgai is and can
run it before encountering advanced configuration. Do NOT change any source
code — this is a documentation-only task. Only `README.md` is modified.

## Critical Corrections

All occurrences of `sgai serve` must become just `sgai`. The `serve` subcommand
still works but is undocumented; the canonical invocation is:

```
sgai                                # Start on localhost:8080
sgai --listen-addr 0.0.0.0:8080    # Start accessible externally
```

Also: `sgai` must be started from the **parent directory** of the project
folders, not from within a project directory.

## Current README Section Order (for reference)

1. Title + 1-line summary
2. Automated Setup with opencode
3. Features
4. Prerequisites
5. Installation
6. Quick Start (macOS)
7. How It Works
8. GOAL.md Reference (with Frontmatter Options subsection)
9. Usage
10. Frontend Development
11. Contributing

## Target README Section Order

Reorganize into this order. Sections marked KEEP must preserve their current
content exactly (after applying the `sgai serve` → `sgai` fix). Sections marked
NEW must be written. Sections marked MODIFY must be edited as described.

1. **Title + expanded summary** (MODIFY) — Keep the title "# Sandgarden AI
   Software Factory". Expand the summary to 2-3 sentences explaining what sgai
   is at a high level: it's an AI software factory that orchestrates multiple
   specialized AI agents to build software from a goal description.

2. **Key Concepts** (NEW) — Introduce the factory mental model with a table:

   | Concept | What it is | Why it matters |
   |---------|-----------|----------------|
   | The Factory | sgai orchestrates multiple AI agents working together like workers in a factory | This is not just "one AI" — it is coordinated teamwork |
   | GOAL.md | The specification that drives everything — your intent, not implementation | You describe *what* you want, not *how* to build it |
   | Agents | Specialized AI instances with specific roles (developer, reviewer, analyst) | Different agents have different expertise, just like a real team |
   | Flow | The DAG that defines how work moves between agents | The `->` syntax defines who reviews whose work |
   | Human-in-the-Loop | You are the factory supervisor, not replaced by AI | You provide guidance when agents need clarification |
   | Coordinator | The foreman who reads your GOAL.md and delegates tasks | Every workflow has a coordinator that manages the other agents |

3. **Installation** (KEEP) — Preserve current content exactly:

   ```sh
   go install github.com/sandgardenhq/sgai/cmd/sgai@latest
   ```

   And the "from source" block with git clone / bun install / make build.

4. **Automated Setup with opencode** (KEEP) — Move here from current position 2.
   Preserve the opencode instructions, the "Before you begin" checklist, and the
   `opencode run` command exactly as they are.

5. **Quick Start (macOS)** (MODIFY) — Keep steps 1 (brew install) and 2
   (opencode auth login) exactly. Simplify step 3: the GOAL.md example must NOT
   include frontmatter. Show only a plain markdown body:

   ```markdown
   # My Project Goal

   Build a REST API with user authentication.

   ## Tasks

   - [ ] Create user registration endpoint
   - [ ] Create login endpoint with JWT
   - [ ] Add password hashing
   ```

   Add a note: "This minimal GOAL.md uses default settings. See the GOAL.md
   Reference section below for advanced options like custom agent flows and
   model assignments."

   Step 4: change `sgai serve` to `sgai`. Keep the localhost:8080 link.

   Add a note about directory structure after step 4:

   > sgai is started from the parent directory containing your project folders.
   > It discovers `GOAL.md` files in subdirectories automatically.

   ```
   workspace/              ← Run `sgai` from here
   ├── project-1/
   │   └── GOAL.md
   ├── project-2/
   │   └── GOAL.md
   └── project-3/
       └── GOAL.md
   ```

6. **How It Works** (MODIFY) — Fix the diagram:

   ```
   GOAL.md → sgai → Monitor in Browser → Iterate
   ```

   Fix step 2: "Run `sgai` to start the web interface" (remove `serve`).
   Keep all 5 steps and the dashboard feature list exactly.

7. **Features** (KEEP) — Move here from current position 3. Preserve the
   bullet list exactly as-is.

8. **Prerequisites** (MODIFY) — Keep the current table but add a third column
   header "Required?" with values:
   - Node.js: Required
   - bun: Required (build only)
   - opencode: Required
   - jj: Required
   - dot (Graphviz): Optional — plain-text SVG fallback
   - gh (GitHub CLI): Optional — merge works without PR creation
   - tmux: Required
   - rg (ripgrep): Required

   Keep the Environment Variables subsection exactly.

9. **GOAL.md Reference** (KEEP) — Preserve the example GOAL.md, the
    Frontmatter Options table, and all content.

10. **Common Flows** (NEW) — Add copy-paste flow examples:

    **Go Backend Only:**
    ```yaml
    flow: |
      "backend-go-developer" -> "go-readability-reviewer"
    ```

    **Go Backend with Safety Analysis:**
    ```yaml
    flow: |
      "backend-go-developer" -> "go-readability-reviewer"
      "go-readability-reviewer" -> "stpa-analyst"
    ```

    **Full-Stack Go + HTMX/PicoCSS:**
    ```yaml
    flow: |
      "backend-go-developer" -> "go-readability-reviewer"
      "go-readability-reviewer" -> "stpa-analyst"
      "htmx-picocss-frontend-developer" -> "htmx-picocss-frontend-reviewer"
      "htmx-picocss-frontend-reviewer" -> "stpa-analyst"
    ```

    **Full-Stack Go + React:**
    ```yaml
    flow: |
      "backend-go-developer" -> "go-readability-reviewer"
      "go-readability-reviewer" -> "stpa-analyst"
      "react-developer" -> "react-reviewer"
      "react-reviewer" -> "stpa-analyst"
    ```

    **Research / Exploration:**
    ```yaml
    flow: |
      "general-purpose"
    ```

11. **Usage** (MODIFY) — Fix both lines:

    ```sh
    sgai                                # Start on localhost:8080
    sgai --listen-addr 0.0.0.0:8080    # Start accessible externally
    ```

12. **Frontend Development** (KEEP) — Preserve all content exactly: the
    description, Frontend Stack table, and Build Commands.

13. **Contributing** (KEEP) — Preserve all content exactly.

## Tasks

- [x] Replace all `sgai serve` references with `sgai` (lines 109, 117, 121, 182, 183)
- [x] Move "Features" after "How It Works"
- [x] Verify the final README reads coherently end-to-end
