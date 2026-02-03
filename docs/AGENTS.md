# SGAI Agents Reference

SGAI (Software Generation AI) uses specialized agents to accomplish different tasks in a software development workflow. Each agent has a specific role, expertise, and set of capabilities. Agents communicate with each other through a messaging system and can be coordinated by the coordinator agent to accomplish complex, multi-step goals.

---

## agent-sdk-verifier-py

Verifies that Python Claude Agent SDK applications are properly configured, follow SDK best practices, and are ready for deployment or testing. This agent inspects Python Agent SDK apps for correct SDK usage, adherence to official documentation recommendations, proper environment setup, and security configurations. Use this agent after creating or modifying a Python Agent SDK application to ensure it meets all requirements before deployment.

---

## agent-sdk-verifier-ts

Verifies that TypeScript Claude Agent SDK applications are properly configured, follow SDK best practices, and are ready for deployment or testing. This agent inspects TypeScript Agent SDK apps for correct SDK usage, proper TypeScript configuration, type checking compliance, and adherence to official documentation recommendations. Invoke this agent after creating or modifying a TypeScript Agent SDK application to validate configuration and catch issues before runtime.

---

## backend-go-developer

Expert Go backend developer for building production-quality APIs, CLI tools, and services with idiomatic Go patterns. This agent writes, tests, and refactors Go code following official Go conventions and best practices from Effective Go. It handles HTTP/API development, database operations, testing, and uses modern Go features (Go 1.21+) including generics, the slices package, and iterators. The agent works closely with the go-readability-reviewer for code quality assurance and must address all review feedback before completing work.

---

## c4-code

Expert C4 Code-level documentation specialist that analyzes code directories to create comprehensive C4 code-level documentation. This agent extracts function signatures, arguments, dependencies, and code structure at the most granular level of the C4 model. It supports multiple programming paradigms (OOP, functional, procedural) and generates Mermaid diagrams for code relationships. Use this agent when documenting code at the lowest C4 level for individual directories and code modules, forming the foundation for higher-level C4 documentation.

---

## c4-component

Expert C4 Component-level documentation specialist that synthesizes C4 Code-level documentation into Component-level architecture. This agent identifies component boundaries, defines interfaces, maps relationships between components, and creates component diagrams. It groups related code files into logical components based on domain, technical, or organizational boundaries. Use this agent when synthesizing code-level documentation into logical components for architectural understanding.

---

## c4-container

Expert C4 Container-level documentation specialist that synthesizes Component-level documentation into Container-level architecture. This agent maps components to deployment units (Docker, Kubernetes, cloud services), documents container interfaces as OpenAPI/Swagger specifications, and creates container diagrams showing high-level technology choices. Use this agent when synthesizing components into deployment containers and documenting system deployment architecture.

---

## c4-context

Expert C4 Context-level documentation specialist that creates high-level system context diagrams and documentation. This agent identifies personas, maps user journeys, documents system features, and captures external dependencies. It synthesizes container and component documentation into stakeholder-friendly context diagrams that show the system, its users, and external systems. Use this agent when creating the highest-level C4 system context documentation that non-technical stakeholders can understand.

---

## cli-output-style-adjuster

Adjusts source code CLI output style for minimal, plain-text output following Unix philosophy principles. This agent scans source code files and applies style transformations to ensure outputs are clean, lowercase, emoji-free, use plain ASCII characters, are silent on success, and properly direct errors to stderr. It works across multiple programming languages (Go, Python, JavaScript, Rust, etc.) and processes recently changed files based on version control diffs.

---

## coordinator

The project manager of the SGAI Software Factory that orchestrates the entire workflow. This agent evaluates GOAL.md and PROJECT_MANAGEMENT.md, delegates tasks to specialized agents, manages checkbox completion in GOAL.md, and ensures the project progresses through brainstorming, work gate approval, and code cleanup phases. The coordinator is read-only for code - it never writes code itself but dispatches work to appropriate specialist agents. It is the sole agent that can communicate with the human partner via questions and the only agent that can mark the workflow as complete.

---

## general-purpose

General-purpose agent for researching complex questions, searching for code, and executing multi-step tasks that no other specialized agent can handle. This agent has full access to read, write, edit, and manage code across any language, run shell commands, and use all development tools. It approaches problems systematically: analyzing requirements, exploring codebases, planning work, executing carefully, and verifying results. Use this agent for tasks that don't fit other specialized agents or require cross-domain expertise.

---

## go-readability-reviewer

Reviews Go code for readability, idioms, and best practices following Go Code Review Comments and the Google Go Style Guide. This is a read-only reviewer that cannot modify files - it provides detailed feedback with line numbers and sends fix recommendations to the backend-go-developer agent via messaging. The agent uses a comprehensive checklist covering formatting, naming, error handling, concurrency, interfaces, documentation, type safety, and modern Go idioms. Every issue identified is mandatory and must be addressed before work can proceed.

---

## htmx-picocss-frontend-developer

Frontend developer specializing in building modern, lightweight web interfaces using HTMX and PicoCSS without heavy JavaScript frameworks. This agent creates fast, accessible, and maintainable web applications with semantic HTML, partial page updates via HTMX attributes, and PicoCSS's classless styling approach.

It enforces a strict no-custom-JavaScript policy (except for idiomorph extension setup) and expects idiomorph-based swaps for auto-refresh flows so UI state is preserved (for example, open/closed `<details>` sections and scroll position). It uses Playwright for visual verification, and ensures accessibility with proper contrast ratios.

Use this agent for building interactive web UIs that need HTMX's AJAX capabilities.

---

## htmx-picocss-frontend-reviewer

The "UI OCD Web Agent" - a hyper-perfectionist frontend reviewer for interfaces built with HTMX and PicoCSS. This agent obsessively reviews visual consistency, predictable interaction patterns, cohesive information architecture, and code readability. It verifies that every interactive element has proper hover/focus/loading/error states, layouts are responsive, accessibility requirements are met, and Playwright tests cover UI behavior. Any rough edge is considered a bug. Use this agent to review and polish HTMX/PicoCSS interfaces to production quality.

---

## openai-sdk-verifier-py

Verifies that Python OpenAI Agents SDK applications are properly configured and follow best practices. This agent checks package installation, Python version compatibility, syntax errors, import statements, agent configuration, tool definitions, environment variables, and run configuration. It supports Basic Agents, Voice Agents (with audio pipeline verification), and Realtime Agents (with WebSocket configuration). Use this agent after creating or modifying a Python OpenAI Agents SDK application to ensure it's ready for deployment.

---

## openai-sdk-verifier-ts

Verifies that TypeScript OpenAI Agents SDK applications are properly configured and follow best practices. This agent checks package installation, TypeScript configuration (tsconfig.json), type checking with `tsc --noEmit`, import statements, agent configuration, Zod-based tool definitions, and environment setup. It supports standard agents, voice agents, and realtime agents with WebSocket/WebRTC transports. Use this agent after creating or modifying a TypeScript OpenAI Agents SDK application.

---

## project-critic-council

A multi-model council that strictly evaluates whether GOAL.md items are truly complete. Multiple models collaborate in a debate-style evaluation, examining checked items against actual evidence (test results, code review, file contents). The council reaches consensus through structured communication, then requests checkbox reverts through the coordinator if work was not genuinely complete. This agent enforces extremely strict standards - "mostly done" or "should work" does not count as complete. It is the last line of defense against incomplete work being marked complete.

---

## retrospective-applier

Reads SUGGESTIONS.md (or IMPROVEMENTS.md) files and applies only the approved suggestions by delegating to appropriate agents. This agent parses approved suggestions, determines whether they are skills or snippets, then delegates creation to the skill-writer or snippet-writer agents respectively. It reports a summary of what was created after processing all approved items. Use this agent to apply retrospective improvements after human review and approval.

---

## retrospective-code-analyzer

Mines code from session diffs to identify valuable code snippets that should be added to SGAI's snippet library. This agent reviews code produced during a session and identifies reusable infrastructure patterns (HTTP handlers, database connections, error handling, test helpers) that would benefit future SGAI users across ANY project - not patterns specific to the application being developed. It checks against existing snippets to avoid duplicates and outputs findings to IMPROVEMENTS.draft.md with priority rankings.

---

## retrospective-refiner

Deduplicates, polishes, and formats IMPROVEMENTS.draft.md into the final IMPROVEMENTS.md with checkbox approval format for human review. This agent validates SGAI relevance (filtering out application-specific improvements), merges similar improvements, verifies uniqueness against existing skills/snippets/agents, prioritizes by impact, and produces a clean approval format with checkboxes. The final output is written to both the retrospective directory and project root for visibility.

---

## retrospective-session-analyzer

Analyzes exported session JSON transcripts to identify skill gaps, struggle patterns, and improvement opportunities for SGAI itself. This agent looks for missing knowledge signals (failed skill searches, trial-and-error debugging), successful workarounds that could be formalized as skills, and agent behavior issues that suggest improvements. It performs deep verification against existing skills to prevent duplicate suggestions and focuses exclusively on improvements that benefit SGAI infrastructure, not the application being developed.

---

## shell-script-coder

Expert shell script developer specializing in writing production-quality shell scripts. This agent creates POSIX-compliant scripts for maximum portability, uses bash-specific features when appropriate, implements proper argument parsing with edge case handling, and ensures robust error handling with appropriate exit codes. It follows best practices like quoting variables, using `"$@"` for arguments, and providing meaningful error messages. Use this agent when you need to create new shell scripts.

---

## shell-script-reviewer

Reviews shell script quality for correctness, portability, security, and best practices. This is a read-only reviewer that cannot modify files or execute commands. It analyzes scripts against criteria including logical correctness, POSIX compatibility, proper variable quoting, input validation, command injection prevention, secure temporary file handling, and meaningful error messages. The agent provides structured feedback with specific line references and a PASS/NEEDS WORK verdict.

---

## skill-writer

Creates new skills from approved suggestions with mandatory validation using the testing-skills-with-subagents process. This agent writes properly formatted SKILL.md files following conventions, then runs a RED-GREEN-REFACTOR testing cycle to ensure the skill works under pressure and resists rationalization. Skills are not considered complete until they pass testing - the agent iterates until the skill is bulletproof. Skills are created in the sgai overlay directory for distribution to all SGAI users.

---

## snippet-writer

Creates new code snippets from approved suggestions following established conventions. This agent writes clean, reusable code snippets with proper header comments documenting purpose, usage, and examples. It uses TODO comments for customizable parts, ensures proper formatting for the target language, and follows language-specific idioms. Snippets are created in the sgai overlay directory for distribution and must pass quality checks including syntax validity and proper documentation.

---

## stpa-analyst

STPA (System Theoretic Process Analysis) hazard analyst for software, physical, and AI systems. This agent treats safety as a control problem and guides users through the 4 STPA steps: defining purpose (losses, hazards, constraints), modeling control structures with Graphviz diagrams, identifying unsafe control actions (4 types), and tracing loss scenarios through causal pathways. It uses interactive questioning to gather system information and documents findings in PROJECT_MANAGEMENT.md. Use this agent for systematic hazard analysis of any system where safety is a concern.

---

## webmaster

Website developer specializing in building marketing sites, landing pages, and institutional websites (not web applications). This agent creates simple Go backends with HTML templates, embedded assets, and Let's Encrypt HTTPS support. It is proficient in Bootstrap, Tailwind CSS, PicoCSS, and vanilla CSS, choosing the appropriate framework based on project needs. The agent emphasizes SEO best practices (meta tags, semantic HTML, Open Graph), mobile-first responsive design, and accessibility. Use this agent for content-driven websites that prioritize presentation and conversion, not complex interactive applications.
