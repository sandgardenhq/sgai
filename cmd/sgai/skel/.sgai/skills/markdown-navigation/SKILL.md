---
name: Markdown Navigation
description: Tool for navigating large markdown files, providing outline and section extraction
when_to_use: When dealing with large markdown files, need to get outline of sections, extract specific section content. Symptoms - large markdown files, need to navigate structure without reading whole file, extract sections for focused reading, documentation navigation, changelog parsing.
version: 1.0.0
languages: all
---

# Markdown Navigation

## Overview

Provides a tool to quickly get the outline of section titles in a markdown file or extract the body of a specific section.

**Core principle:** Use simple bash script to parse markdown headers and content for efficient navigation.

## When to Use

- Need hierarchical outline of all sections in a markdown file
- Need to extract content of a specific section without manual parsing
- Working with large documentation files
- When grep and read tools are insufficient for structured navigation
- Symptoms: struggling to find sections in long markdown, manually scrolling through files, need focused reading of specific parts

## Core Pattern

Use the `markdown-viewer.sh` script:

- `./markdown-viewer.sh file.md` for outline
- `./markdown-viewer.sh file.md "Section Name"` for section body

## Quick Reference

| Command | Description |
|---------|-------------|
| `./markdown-viewer.sh file.md` | Print hierarchical outline of sections |
| `./markdown-viewer.sh file.md "Section"` | Print content of specified section |

## Implementation

The tool is implemented as a bash script located at `markdown-viewer.sh` in this skill directory.

It uses `grep -E` and `sed` to parse markdown headers and extract content ranges.

## Common Mistakes

- Section name must match exactly (case sensitive, includes spaces)
- For outline mode, provide no second argument
- Script requires executable permissions (`chmod +x`)
- File path must be correct and accessible

## Real-World Impact

Allows efficient navigation of large markdown files like documentation, changelogs, or project READMEs, saving time on manual searching and improving productivity when working with structured text.
