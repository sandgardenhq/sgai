---
name: how-to-use-code-snippets
description: Use when you need to find code snippets by language and query. When looking for code examples, patterns, or snippets in a specific programming language. When you need to search for code snippets quickly. When searching for code patterns or best practices.
---

# How to Use Code Snippets

## Overview

The sgai_find_snippets tool allows you to search for code snippets using a specific programming language and a query string. It returns relevant code examples that match your search criteria, helping you find patterns, best practices, and implementations quickly.

## When to Use

Use sgai_find_snippets when:
- You need code examples for a specific language feature
- You're looking for patterns or best practices in code
- You want to find snippets for common programming tasks
- You need to reference existing code snippets
- Searching manually would be time-consuming

## Quick Reference

| Parameter | Description | Example |
|-----------|-------------|---------|
| language | Programming language | 'python', 'javascript', 'java' |
| query | Search query | 'sort list', 'array methods', 'file io' |

## Implementation

To use sgai_find_snippets, call the tool with the language and query parameters.

Example:

sgai_find_snippets language='python' query='list comprehension'

This will return Python code snippets related to list comprehensions.

## Common Mistakes

- Using incorrect language codes (e.g., 'py' instead of 'python', 'js' instead of 'javascript')
- Too vague or broad queries (e.g., 'code' instead of 'sort array')
- Not specifying the language when it's not obvious from context
- Expecting exact matches instead of relevant examples

## Real-World Impact

Using sgai_find_snippets helps you find relevant code examples quickly, saving time on manual searches or writing code from scratch. It improves productivity by providing instant access to proven patterns and implementations.
