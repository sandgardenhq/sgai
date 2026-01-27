---
name: Incorporate Snippets
description: Guide for adding new code snippets to the library
when_to_use: When you need to add a new code snippet to the sgai_find_snippets library, when creating files without proper metadata, when using incorrect filenames or extensions, when adding code without extensive comments
version: 1.1.0
---

# Incorporate Snippets

## Overview

This skill guides the process of incorporating new code snippets into the sgai_find_snippets library. It ensures snippets are properly formatted and stored for future retrieval.

## When to Use

Use this skill when:
- Adding a new code snippet to the sgai_find_snippets library
- Creating snippet files without frontmatter metadata
- Using filenames with spaces, special characters, or wrong extensions
- Adding code without extensive explanatory comments
- Missing required fields like language, description, or when_to_use

## The Process

### Phase 1: Gather Snippet Information
- Ask for the programming language
- Ask for the snippet name (brief, descriptive)
- Ask for a single-line description
- Ask for when to use the snippet
- Ask for the code snippet itself

### Phase 2: Validate Snippet
- Check that all required fields are provided
- Sanitize filename: convert to kebab-case, remove special characters
- Ensure the code is syntactically correct for the language using language-specific validation
- Verify the snippet follows best practices
- Check for filename conflicts and resolve (append -2, -3, etc.)

### Phase 3: Save Snippet
- Generate the filename: kebab-case-name.extension
- Create the .sgai/snippets/{language}/ directory if missing
- Write the frontmatter and code in the correct format
- Confirm successful addition

## Syntax Validation by Language

| Language | Validation Command | Notes |
|----------|-------------------|-------|
| JavaScript/TypeScript | `node -c <filename>` | Checks syntax |
| Python | `python -m py_compile <filename>` | Compiles to check syntax |
| Go | `go build -o /dev/null <filename>` | Builds to check syntax |
| Rust | `rustc --emit=dep-info <filename>` | Checks syntax |
| Java | `javac <filename>` | Compiles to check syntax |

## Filename Generation

- Convert name to kebab-case: "Error Handling" → "error-handling"
- Remove special characters: "Error Handling!" → "error-handling"
- Add proper extension based on language
- If filename exists, append number: error-handling.go, error-handling-2.go, etc.
- Ensure directory exists, create if missing

## File Format
Snippets must follow this exact format:

```
---
name: Snippet Name
description: Single line description
when_to_use: When to use this snippet
---

/* Extensive comment explaining the code */
actual_code_goes_here()
```

## Common Mistakes

- Creating files without frontmatter metadata
- Using filenames with spaces or special characters
- Missing extensive comments explaining the code
- Not validating syntax before adding
- Forgetting to create the language directory
- Not checking for filename conflicts

## Important Notes
- Filenames should be kebab-case versions of the name
- Extensions should match the language (e.g., .go, .js, .py)
- Comments should be comprehensive but concise
- Test snippets before adding if possible
