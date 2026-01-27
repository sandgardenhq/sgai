---
name: Log Analysis for Skill Extraction and How to Update .sgai/skills/REQUESTS.md correctly
description: Analyze workbench logs to identify missing skills and using .sgai/skills/REQUESTS.current.md it will create .sgai/skills/REQUESTS.md
when_to_use: When sgai_find_skills returns no results for a query, when workbench fails to progress due to missing techniques, when iterating the workbench to improve capabilities, when logs show repeated searches for same missing skills, when you want to harvest misses from .sgai/workbench.log, when you see 'no skills found' messages, when manual workarounds are logged for missing techniques, when sgai_find_skills returns empty results arrays

version: 1.4.0

languages: all

---

# Log Analysis for Skill Extraction and How to Update .sgai/skills/REQUESTS.md correctly

## Overview
This skill automates the discovery of missing skills by analyzing workbench logs. It identifies gaps in the skill set where the workbench searched for help but found nothing, enabling systematic improvement of capabilities.

## When to Use
- After sgai_find_skills returns empty results
- When workbench gets stuck on tasks that seem solvable
- During iteration cycles to expand skill coverage
- When logs show patterns of failed skill lookups
- When adding new features that require undocumented techniques

## Core Pattern
1. Parse search logs for queries without matches
2. Correlate with workbench activity for context
3. Define specific skills needed
4. Document requests in standardized format

## Quick Reference
| Step | Action | Tools |
|------|--------|-------|
| 1 | Analyze workbench.log for misses | Read, grep |
| 2 | Extract context from workbench.log | Read, grep |
| 3 | Identify skill gaps | Analysis |
| 4 | Read .sgai/skills/REQUESTS.current.md | Read |
| 5 | Overwrite .sgai/skills/REQUESTS.md with the content from .sgai/skills/REQUESTS.current.md | Edit |
| 6 | Update .sgai/skills/REQUESTS.md | Edit |
| 7 | Validate entries | Review |

## Implementation

### Step 1: Analyze Search Log
- Read .sgai/workbench.log completely
- Extract unique "query" values from JSON lines where "results" is empty array
- For each query, check if `sgai_find_skills [query]` returns skills
- Flag queries with zero matches as potential misses

To extract misses, use Python script:
```python
import json
misses = []
with open('.sgai/workbench.log', 'r') as f:
    for line in f:
        line = line.strip()
        if line:
            try:
                entry = json.loads(line)
                if entry.get('action') == 'sgai_find_skills' and not entry.get('results', []):
                    misses.append(entry['query'])
            except json.JSONDecodeError:
                pass
unique_misses = list(set(misses))
print('Potential misses:', unique_misses)
```

### Step 2: Analyze Workbench Log
- Read .sgai/workbench.log for timestamps
- Find entries near missed query times (e.g., within 15 minutes before/after)
- Extract workbench actions and failures around those periods

To correlate, parse all entries with timestamps, for each miss, find nearby "workbench-activity" entries.

Use Python:
```python
from datetime import datetime, timedelta
import json

# Load all entries
entries = []
with open('.sgai/workbench.log', 'r') as f:
    for line in f:
        line = line.strip()
        if line:
            try:
                entry = json.loads(line)
                if 'timestamp' in entry:
                    entry['parsed_time'] = datetime.fromisoformat(entry['timestamp'].replace('Z', '+00:00'))
                entries.append(entry)
            except:
                pass

# For each miss, find context
for miss in unique_misses:
    miss_time = None
    for entry in entries:
        if entry.get('query') == miss:
            miss_time = entry.get('parsed_time')
            break
    if miss_time:
        context = []
        for entry in entries:
            if 'parsed_time' in entry and abs((entry['parsed_time'] - miss_time).total_seconds()) < 900:  # 15 min
                if entry.get('action') == 'workbench-activity':
                    context.append(entry.get('description', ''))
        print(f'Context for {miss}: {context}')
```

### Step 3: Identify Missing Skills
- Match misses to workbench context
- Define what skill would have prevented the issue
- Check .sgai/skills/REQUESTS.md and existing skills for duplicates

### Step 4: Read .sgai/skills/REQUESTS.current.md
- Learn the structure of requests, also known as missing skills

### Step 5: Copy .sgai/skills/REQUESTS.current.md into .sgai/skills/REQUESTS.md
- Use shell or write to duplicate PRECISELY the content of .sgai/skills/REQUESTS.current.md into .sgai/skills/REQUESTS.md

### Step 6: Update .sgai/skills/REQUESTS.md
Add entries following the format:
```
## [Descriptive Name]
**What I need:** One-line skill description
**When I'd use it:** Specific symptoms and situations
**Why I need this:** What makes this non-obvious
**Added:** YYYY-MM-DD
```

### Step 7: Validate
- Ensure each request is specific and actionable
- Verify symptoms are concrete (not abstract)
- Confirm one skill per request
- Check for completeness of required fields

## Common Mistakes

- Treating all search queries as skill needs (some are file searches)

- Creating vague requests without specific symptoms

- Duplicating existing skills without checking

- Missing context correlation between logs

- Requesting skills for file searches rather than technique searches

- Failing to define specific skills needed, leading to vague requests

## Tools to Use
- Read tool for complete log file access
- Grep tool for pattern matching in logs
- Bash for running sgai_find_skills checks
- Edit tool for .sgai/skills/REQUESTS.md updates

## Real-World Impact
This skill has identified critical gaps like log analysis itself, enabling the workbench to self-improve by discovering what capabilities it lacks.
