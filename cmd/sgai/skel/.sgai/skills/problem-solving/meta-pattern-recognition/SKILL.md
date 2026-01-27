---
name: Meta-Pattern Recognition
description: Spot patterns appearing in 3+ domains to find universal principles
when_to_use: Same issue in different parts of codebase. Pattern feels familiar across projects. "Haven't I solved this before?" Different teams solving similar problems. Recurring solution shapes.
version: 1.0.0
---

# Meta-Pattern Recognition

## Overview

When the same pattern appears in 3+ domains, it's probably a universal principle worth extracting.

**Core principle:** Find patterns in how patterns emerge.

## Quick Reference

| Pattern Appears In | Abstract Form | Where Else? |
|-------------------|---------------|-------------|
| CPU/DB/HTTP/DNS caching | Store frequently-accessed data closer | LLM prompt caching, CDN |
| Layering (network/storage/compute) | Separate concerns into abstraction levels | Architecture, organization |
| Queuing (message/task/request) | Decouple producer from consumer with buffer | Event systems, async processing |
| Pooling (connection/thread/object) | Reuse expensive resources | Memory management, resource governance |

## Process

1. **Spot repetition** - See same shape in 3+ places
2. **Extract abstract form** - Describe independent of any domain
3. **Identify variations** - How does it adapt per domain?
4. **Check applicability** - Where else might this help?

## Example

**Pattern spotted:** Rate limiting in API throttling, traffic shaping, circuit breakers, admission control

**Abstract form:** Bound resource consumption to prevent exhaustion

**Variation points:** What resource, what limit, what happens when exceeded

**New application:** LLM token budgets (same pattern - prevent context window exhaustion)

## Red Flags You're Missing Meta-Patterns

- "This problem is unique" (probably not)
- Multiple teams independently solving "different" problems identically
- Reinventing wheels across domains
- "Haven't we done something like this?" (yes, find it)

## Remember

- 3+ domains = likely universal
- Abstract form reveals new applications
- Variations show adaptation points
- Universal patterns are battle-tested