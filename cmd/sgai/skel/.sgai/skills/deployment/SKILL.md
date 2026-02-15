---
name: deployment
description: MANDATORY when GOAL.md mentions deploying, shipping, releasing, or publishing built artifacts to any environment - reads deployment instructions from the .deploy/ directory, enforces pre-deployment verification, artifact integrity checks, deployment execution safety, post-deployment validation, and rollback planning; prevents deploying without tests, skipping rollback plans, or bypassing environment validation. When GOAL.md says deploy, ship, release, publish, or push artifact to production/staging/server.
---

# Deployment

## Overview

Deployment of built artifacts requires mandatory safety gates. This skill reads deployment instructions from the `.deploy/` directory and enforces a structured safety process around them.

**Three-part model:**

| Concern | Location | Role |
|---------|----------|------|
| **What** to deploy | `GOAL.md` | Declares deployment intent ("deploy the app to staging") |
| **How** to deploy | `.deploy/` | Contains deployment commands, targets, health checks, rollback procedures |
| **Safely** deploying | This skill | Enforces safety gates, checklists, and rollback planning |

**Core principle:** No artifact leaves the build environment without passing every gate. No deployment proceeds without a rollback plan. All deployment instructions come from `.deploy/`.

**Violating the letter of this rule is violating the spirit of this rule.**

**Announce at start:** "I'm using the deployment skill to ensure safe artifact deployment."

## Activation Rule

**This skill is MANDATORY** when GOAL.md contains any deployment intent. Scan GOAL.md for:

- **Deployment verbs:** deploy, ship, release, publish, push, distribute, deliver, roll out
- **Artifact nouns:** binary, container, image, package, bundle, executable, build output, artifact
- **Target nouns:** server, cloud, production, staging, environment, cluster, host, instance, registry

If ANY of these appear in GOAL.md in the context of deploying a built artifact, this skill **must** be loaded and followed. No exceptions.

**Not applicable when:**
- GOAL.md only mentions building (no deployment target)
- GOAL.md only mentions testing (no deployment step)
- Changes are documentation-only

## Process

### Step 0: Load Deployment Instructions

**Before any gates, read `.deploy/` to understand the deployment procedure.**

Read the contents of the `.deploy/` directory at the repository root. From it, extract:

- **What artifact** is being deployed (name, build command, output path)
- **Where** it is going (target environment, host, registry)
- **How** to deploy it (commands, scripts, pipelines)
- **How** to verify it (health check endpoints, smoke test commands)
- **How** to roll back (rollback commands, previous version restore procedure)

If `.deploy/` contains environment-specific subdirectories, identify which environment GOAL.md is targeting and read the relevant subdirectory.

**If `.deploy/` does not exist:**

1. Scan the project structure and GOAL.md to understand the deployment context
2. Draft a `.deploy/README.md` with deployment instructions based on what you can infer from the project (build system, artifact type, any deployment hints in the codebase)
3. Present the draft to the user for review
4. **STOP. Do not proceed with deployment until the user confirms the `.deploy/` contents are correct.**

The skill never guesses deployment commands at execution time. All deployment commands must be documented in `.deploy/` and confirmed by the user before any deployment proceeds.

```
.deploy/ check:
  directory exists:     [YES/DRAFTING]
  instructions loaded:  [YES/NO]
  artifact identified:  [YES/NO]
  target identified:    [YES/NO]
  deploy command found: [YES/NO]
  rollback found:       [YES/NO]
  health check found:   [YES/NO]

ALL must show YES before proceeding to Step 1.
DRAFTING = draft .deploy/README.md, present to user, STOP until confirmed.
```

### Step 1: Pre-Deployment Gate

**No deployment proceeds until ALL gates pass.**

- [ ] **Tests pass** - Run the full test suite. Fresh run, not cached results.
- [ ] **Build succeeds** - Build the artifact from clean state. Exit code 0.
- [ ] **Linter clean** - No lint errors or warnings in shipped code.
- [ ] **completionGateScript passes** - If GOAL.md defines a `completionGateScript`, run it and verify it passes.
- [ ] **Version/tag is set** - Artifact version matches intended release.
- [ ] **No debug configuration** - Debug flags, dev endpoints, test credentials are removed from the artifact.

```
Gate check:
  tests:              [PASS/FAIL]
  build:              [PASS/FAIL]
  lint:               [PASS/FAIL]
  completionGate:     [PASS/FAIL/N/A]
  version:            [SET/MISSING]
  debug config:       [CLEAN/FOUND]

ALL must show PASS/SET/CLEAN before proceeding.
Any FAIL/MISSING/FOUND = STOP. Fix before continuing.
```

### Step 2: Artifact Verification

Verify the artifact at the path specified in `.deploy/`.

- [ ] **Artifact exists** - The built artifact is present at the path documented in `.deploy/`.
- [ ] **Artifact is correct version** - Version string/tag in artifact matches what was intended.
- [ ] **Artifact integrity** - Record checksum (SHA256) of the artifact for verification.
- [ ] **Artifact size sanity** - Size is within expected range (not suspiciously small or large).
- [ ] **No dev/test artifacts** - Only production artifacts are included in the deployment package.

### Step 3: Environment Validation

Validate the target environment documented in `.deploy/`. Use the validation commands from `.deploy/` if provided.

- [ ] **Target environment is accessible** - Can reach the deployment target (SSH, API, registry).
- [ ] **Required services available** - Dependencies (database, cache, queue) are running.
- [ ] **Configuration is correct** - Environment variables, secrets, and config files match the target environment.
- [ ] **Sufficient resources** - Disk space, memory, CPU are adequate.
- [ ] **No active incidents** - Target environment is not currently experiencing issues.

```
Environment check:
  target reachable:   [YES/NO]
  services up:        [YES/NO]
  config validated:   [YES/NO]
  resources ok:       [YES/NO]
  no incidents:       [YES/NO]

ALL must show YES before proceeding.
```

### Step 4: Rollback Plan

**MANDATORY. No deployment without a documented rollback.**

Use the rollback procedure from `.deploy/` as the basis. Verify it is complete and current.

- [ ] **Current state recorded** - Document what is currently deployed (version, config, state).
- [ ] **Rollback procedure documented** - `.deploy/` contains step-by-step instructions to revert.
- [ ] **Rollback tested or verified possible** - Confirm the rollback mechanism works.
- [ ] **Rollback trigger defined** - What conditions trigger a rollback (error rate, health check failure, etc.).
- [ ] **Time estimate** - How long will rollback take?

If `.deploy/` does not include rollback instructions, **STOP**. Add rollback instructions to `.deploy/` and get user confirmation before proceeding.

### Step 5: Deployment Execution

Execute the deployment command documented in `.deploy/`.

- [ ] **Announce deployment** - Notify stakeholders that deployment is starting.
- [ ] **Execute deployment command** - Run the exact command from `.deploy/`. Do not improvise.
- [ ] **Monitor for errors** - Watch deployment output for failures.
- [ ] **Log deployment** - Record what was deployed, when, and by whom.

```
Deployment log:
  artifact:    [name and version]
  target:      [environment from .deploy/]
  command:     [command from .deploy/]
  started:     [timestamp]
  completed:   [timestamp]
  status:      [SUCCESS/FAILED]
  deployed by: [agent/user]
```

**If deployment fails:** Stop. Do NOT retry blindly. Investigate the failure. If the failure is understood and fixable, fix and restart from Step 1.

### Step 6: Post-Deployment Verification

Run the health checks and verification steps documented in `.deploy/`.

- [ ] **Service is running** - The deployed artifact is executing.
- [ ] **Health checks pass** - All health/readiness endpoints from `.deploy/` return healthy.
- [ ] **Key functionality works** - Smoke test critical paths as documented in `.deploy/`.
- [ ] **No error spikes** - Error rates are within normal range.
- [ ] **Performance acceptable** - Response times and throughput are within SLA.

```
Post-deployment verification:
  service running:    [YES/NO]
  health checks:      [PASS/FAIL]
  smoke tests:        [PASS/FAIL]
  error rate:         [NORMAL/ELEVATED]
  performance:        [OK/DEGRADED]

ALL must show YES/PASS/NORMAL/OK.
Any NO/FAIL/ELEVATED/DEGRADED = Execute rollback plan from .deploy/.
```

## The `.deploy/` Directory

`.deploy/` is a free-form directory at the repository root. It is the single source of truth for how deployment works for this project. The skill makes no assumptions about its internal structure beyond: the contents must be readable and must describe the deployment procedure.

### Example Structures

**Simple project:**
```
.deploy/
  README.md           # all deployment instructions in one file
```

**Multi-environment:**
```
.deploy/
  README.md           # shared instructions, overview
  staging/
    README.md         # staging-specific commands and targets
  production/
    README.md         # production-specific commands and targets
```

**Script-based:**
```
.deploy/
  README.md           # overview and manual steps
  deploy.sh           # deployment script
  rollback.sh         # rollback script
  health-check.sh     # health check script
```

**Container-based:**
```
.deploy/
  README.md           # overview
  Dockerfile          # container build
  docker-compose.yml  # orchestration
  k8s/
    deployment.yaml
    service.yaml
```

### Drafting `.deploy/` for a New Project

When `.deploy/` does not exist and this skill is activated, draft a `.deploy/README.md` with the following structure, filled in from project context:

```markdown
# Deployment Instructions

## Artifact
- Name: [inferred from project]
- Build command: [inferred from build system]
- Output path: [inferred from build output]

## Target
- Host: [ask user if not inferable]
- Method: [scp, docker push, kubectl apply, etc.]
- Deploy command: [the actual command to run]

## Health Check
- URL or command: [health endpoint or verification command]
- Expected result: [HTTP 200, exit code 0, etc.]

## Rollback
- Procedure: [step-by-step rollback commands]
- Time estimate: [duration]
- Trigger: [conditions that require rollback]

## Environment Validation
- Reachability: [command to verify target is accessible]
- Services: [command to verify dependencies are running]
```

After drafting, present it to the user and **STOP until confirmed**.

## Rules

1. **No deployment without `.deploy/`** - If `.deploy/` does not exist or is empty, draft it and wait for user confirmation. Never deploy without documented instructions.
2. **No deployment without passing tests** - Fresh test run, not "tests passed yesterday." Run them NOW.
3. **No deployment without rollback plan** - If you cannot roll back, you cannot deploy. Period.
4. **No deployment without environment validation** - Deploying to an unreachable or unhealthy environment wastes time and risks data.
5. **No debug artifacts in production** - Debug builds, test credentials, dev endpoints must not reach production.
6. **No skipping post-deployment verification** - Deployment is not done until post-verification passes. Walking away after `deploy` is negligence.
7. **No retry without investigation** - Failed deployments must be investigated before retrying. Blind retries compound problems.
8. **Rollback on any post-deployment failure** - If post-deployment verification fails, execute the rollback plan from `.deploy/` immediately. Do not "wait and see."
9. **No improvised deployment commands** - Execute what `.deploy/` says. If `.deploy/` is wrong, fix `.deploy/` first, then redeploy.

## Rationalization Table

| Excuse | Reality |
|--------|---------|
| "Tests passed last time" | Run them NOW. Code may have changed. Dependencies may have changed. |
| "It's just a small change" | Small changes break production. The size of the change does not reduce the risk. |
| "We can fix it in production" | Fix it BEFORE deploying. Production fixes under pressure create more bugs. |
| "Rollback is obvious" | Document it in `.deploy/`. Under stress, "obvious" becomes "forgotten." |
| "Staging is the same as production" | Validate production separately. Configuration, data, and load differ. |
| "We don't have time for all these checks" | You don't have time for a production outage. Checks take minutes, outages take hours. |
| "It works on my machine" | Your machine is not production. Verify in the target environment. |
| "The team is waiting" | The team prefers a working deployment over a fast broken one. |
| "We'll verify after" | After deployment is too late if it's broken. Verify BEFORE and AFTER. |
| "This deployment is low risk" | All deployments carry risk. Low-risk assessments are often wrong. |
| "CI passed, so it's fine" | CI is necessary but not sufficient. Environment-specific issues exist. |
| "We can always roll back" | Not if you didn't plan and test the rollback. |
| "Skip staging, deploy to prod directly" | Staging exists for a reason. Every environment skip increases risk. |
| "Human partner already validated the environment" | Validate independently. Trust but verify. Someone else's check is not your check. |
| "Automated backups handle rollback" | Automated backups are not a rollback plan. A rollback plan has documented procedure, trigger conditions, and time estimate. Backups are a tool, not a plan. |
| "I checked the environment last deployment" | Stale checks are not checks. Validate fresh every time. State changes between deployments. |
| "This is an emergency, skip the process" | The process IS the emergency response. It's designed to take 10-15 minutes. Skipping it risks turning a 30-minute outage into a multi-hour incident. |
| "I'll just add a quick health check instead of full verification" | A health check is one step of six. Partial compliance is non-compliance. |
| "The previous engineer said just SCP and restart" | Someone else's shortcut is not your deployment process. Follow the skill. |
| "I know how to deploy this, I don't need .deploy/" | Document it in `.deploy/` anyway. Knowledge in your head is not a deployment procedure. |
| "I'll just run the command I found in the CI config" | CI config is not `.deploy/`. If it's correct, put it in `.deploy/`. |

## Red Flags - STOP

If you find yourself thinking or saying any of these, STOP and re-read this skill:

- "Just deploy it"
- "Skip the checks, we're in a hurry"
- "We'll verify after deployment"
- "No time for staging"
- "Rolling back is easy, we don't need a plan"
- "Tests passed before, no need to rerun"
- "It's a trivial change"
- "The deployment pipeline handles everything"
- "We can hotfix if something breaks"
- "Everyone is waiting, just push it"
- "I'm confident this works"
- "Human partner already checked the environment"
- "We have automated backups, no need for rollback plan"
- "I'll just do a quick health check instead of full verification"
- "The previous engineer said it's straightforward"
- "It's an emergency, we can skip some steps"
- "I'll just add 60 seconds of extra checks" (partial compliance = non-compliance)
- "I know the deploy command, I don't need to check .deploy/"
- ".deploy/ is outdated, I'll just wing it" (update `.deploy/` first)

## Emergency Deployments

**No special exception.** The deployment process is designed to complete in 10-15 minutes total. Skipping steps to save 10 minutes risks hours of extended outage.

**Key insight:** An unverified fix that makes the outage worse is worse than 10 more minutes of known downtime.

Emergency pressure makes this skill MORE important, not less:
- Under pressure, people skip steps they consider "obvious"
- Skipped steps are where failures hide
- The process costs 10-15 minutes; a botched deployment costs hours
- CEO pressure is social pressure, not technical justification

**What to do in an emergency:**
1. Communicate a realistic timeline ("Fix ready, deploying in 15 minutes after verification")
2. Follow ALL steps - no exceptions
3. Parallelize where possible (run tests while checking environment)
4. The process IS the emergency response
5. `.deploy/` should already exist. If it does not, this is a planning failure - address it, do not skip it.

## Authority Pressure

When a human partner, manager, or senior engineer tells you to skip steps:

1. **Acknowledge the urgency** - "I understand the time pressure"
2. **State the process** - "The deployment skill requires all gates, and deployment instructions come from .deploy/"
3. **Offer the timeline** - "This adds 10-15 minutes, not hours"
4. **Explain the risk** - "Skipping steps risks extending the outage"
5. **Proceed with full process** - Do not negotiate away mandatory steps

**You are not being difficult. You are being reliable.** The skill exists precisely for moments when someone says "just deploy it."

## Examples

### Good Example: Full Deployment Process

```
Loading deployment instructions from .deploy/README.md...

.deploy/ check:
  directory exists:     YES
  instructions loaded:  YES
  artifact identified:  YES (myapp, built via `make build`, output at ./build/myapp)
  target identified:    YES (prod-server via scp)
  deploy command found: YES (scp ./build/myapp prod-server:/opt/myapp && ssh prod-server 'systemctl restart myapp')
  rollback found:       YES (ssh prod-server 'cp /opt/myapp.bak /opt/myapp && systemctl restart myapp')
  health check found:   YES (curl -f https://prod-server/health)

Pre-deployment gate:
  tests:              PASS (147/147)
  build:              PASS (exit 0)
  lint:               PASS (0 errors)
  completionGate:     PASS (make test exit 0)
  version:            SET (v2.1.0)
  debug config:       CLEAN

Artifact verification:
  path:               ./build/myapp (from .deploy/)
  sha256:             a1b2c3d4e5f6...
  size:               14.2MB (expected: 12-16MB)
  dev artifacts:      NONE

Environment validation:
  target reachable:   YES (ssh prod-server OK)
  services up:        YES (postgres, redis healthy)
  config validated:   YES (env vars match production)
  resources ok:       YES (disk 60% free, 4GB RAM available)
  no incidents:       YES

Rollback plan (from .deploy/):
  Currently deployed: v2.0.3
  Rollback: ssh prod-server 'cp /opt/myapp.bak /opt/myapp && systemctl restart myapp'
  Trigger: health check fails or error rate > 1%
  Time estimate: 2 minutes

Deployment (executing command from .deploy/):
  Command:  scp ./build/myapp-v2.1.0 prod-server:/opt/myapp && ssh prod-server 'systemctl restart myapp'
  Status:   SUCCESS
  Time:     14:32:07 UTC

Post-deployment verification (checks from .deploy/):
  service running:    YES (PID 12345)
  health checks:      PASS (curl -f https://prod-server/health returns 200)
  smoke tests:        PASS (login, dashboard, API)
  error rate:         NORMAL (0.02%)
  performance:        OK (p99: 180ms)

Deployment complete. v2.1.0 is live.
```

### Good Example: Drafting `.deploy/` for a New Project

```
GOAL.md says: "Deploy the API server to the staging environment."

Checking for .deploy/ directory... NOT FOUND.

Scanning project structure:
  - Go project with Makefile
  - Build target: make build → ./bin/api-server
  - Dockerfile present
  - No existing deployment scripts

Drafting .deploy/README.md:

  # Deployment Instructions

  ## Artifact
  - Name: api-server
  - Build command: make build
  - Output path: ./bin/api-server

  ## Target
  - Host: [NEEDS USER INPUT]
  - Method: [NEEDS USER INPUT - scp, docker push, etc.]
  - Deploy command: [NEEDS USER INPUT]

  ## Health Check
  - URL or command: [NEEDS USER INPUT]
  - Expected result: HTTP 200

  ## Rollback
  - Procedure: [NEEDS USER INPUT]
  - Time estimate: [NEEDS USER INPUT]
  - Trigger: health check failure or error rate spike

  ## Environment Validation
  - Reachability: [NEEDS USER INPUT]
  - Services: [NEEDS USER INPUT]

STOPPED. Presenting draft to user for review and completion.
Deployment will not proceed until .deploy/ is confirmed.
```

### Bad Example: Skipping `.deploy/`

```
WRONG:

"I can see from the Makefile how to deploy, I'll just run it."
→ Document it in .deploy/ first. Undocumented deployment commands are not deployment procedures.

"Tests passed yesterday, deploying now."
→ Tests must be run fresh. Yesterday's results are stale.

"Deploying to production, will check if it works after."
→ Post-deployment verification is mandatory, but so is pre-deployment verification.

"No rollback plan needed, it's a simple update."
→ Every deployment needs a rollback plan in .deploy/. No exceptions.

".deploy/ exists but it's from last month, probably still correct."
→ Verify .deploy/ is current before every deployment. Stale instructions are wrong instructions.
```

## Checklist

Before marking deployment complete, verify:

- [ ] `.deploy/` loaded: instructions read and understood
- [ ] Pre-deployment gate: all checks passed (fresh run)
- [ ] Artifact verified: exists at path from `.deploy/`, correct version, integrity checked
- [ ] Environment validated: accessible, services up, config correct per `.deploy/`
- [ ] Rollback plan: documented in `.deploy/`, tested/verified, trigger defined
- [ ] Deployment executed: command from `.deploy/` ran, monitored, logged
- [ ] Post-deployment verified: service up, health checks from `.deploy/` pass, no errors
- [ ] Rollback plan retained: keep until next successful deployment

## Integration

**Pairs with:**
- **verification-before-completion** - Provides the general verification discipline this skill specializes for deployment
- **finishing-a-development-branch** - Handles the development completion before deployment begins
- **code-cleanup** - Ensures clean code before building deployment artifacts
