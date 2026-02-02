---
name: Code Cleanup
description: Clean up generated code by questioning why things are there and cross-referencing with GOAL.md, even when tempted to skip due to time pressure or thinking it's "good enough"
when_to_use: After generating code, when tempted to commit messy code, or when noticing clutter that could be removed.
version: 1.1.0
---

# Code Cleanup

## Overview

Review generated code, question the purpose of each part, cross-reference with GOAL.md, and remove or simplify based on good taste and necessity. Do not skip cleanup even if the code is working - cleanup ensures maintainability and alignment with goals.

## Why Cleanup Matters

Even working code benefits from cleanup:
- Removes clutter that confuses future readers
- Prevents accumulation of technical debt
- Ensures code aligns with project goals
- Makes reviews focus on logic, not trivial fixes
- Improves long-term maintainability

## Red Flags - STOP

Do not skip cleanup if you notice:
- "It's working, not critical"
- "Wait until review"
- "I have plans/dinner/end of day"
- "Review will catch it"
- "Being pragmatic over perfection"
- "Code is functional, tested"
- "Minor issues don't break anything"
- "Exhausted, prevent burnout"
- "Tomorrow's fresh perspective"
- "Low-impact technical debt"

## Process

1. Identify the generated code files in the project.
2. For each file, read the code and analyze each significant component (functions, classes, imports, etc.).
3. Ask "Why is this here?" for each component and check alignment with GOAL.md.
4. If a component is unnecessary, redundant, or can be simplified without affecting functionality, remove or refactor it.
5. Ensure the cleaned code maintains functionality and follows coding best practices.
6. Log the changes made in .sgai/PROJECT_MANAGEMENT.md

## Rationalization Table

| Excuse | Reality |
|--------|---------|
| "It's working, not critical" | Working code still needs cleanup to prevent confusion and debt. |
| "Wait until review" | Reviews should focus on design, not cleanup. Do it now. |
| "I have plans/dinner/end of day" | Cleanup is quick (20 min) and improves quality. |
| "Review will catch it" | Proactive cleanup is better than reactive fixes. |
| "Being pragmatic over perfection" | Cleanup is pragmatic - it saves time later. |
| "Code is functional, tested" | Functional code still benefits from cleanup to maintain standards. |
| "Minor issues don't break anything" | Minor issues accumulate into major problems over time. |
| "Exhausted, prevent burnout" | Short cleanup prevents long-term fatigue from messy code. |
| "Tomorrow's fresh perspective" | Delaying cleanup often means it never happens. |
| "Low-impact technical debt" | All technical debt compounds; start small fixes now. |

## Naming Guidelines: Long Names Are Long

A name has two goals:
- It needs to be *clear*: you need to know what the name refers to.
- It needs to be *precise*: you need to know what it does *not* refer to.

After a name has accomplished those goals, any additional characters are dead weight.

### 1. Omit words that are obvious given the type

If your language has a static type system, users usually know the type of a variable. Redundant type information in names is clutter.

```
// Bad:
String nameString;
List<DateTime> holidayDateList;
Map<Employee, Role> employeeRoleHashMap;

// Better:
String name;
List<DateTime> holidays;
Map<Employee, Role> employeeRoles;
```

For method names, don't describe parameters or types - the parameter list does that:

```
// Bad:
mergeTableCells(List<TableCell> cells)

// Better:
merge(List<TableCell> cells)
```

### 2. Omit words that don't disambiguate the name

Ask yourself for each word: Would removing it cause confusion with something else? If no, remove it.

```
// Bad:
recentlyUpdatedAnnualSalesBid
finalBattleMostDangerousBossMonster

// Better:
bid
boss
```

### 3. Omit words that are known from the surrounding context

A method or field occurs in the context of a class. Take that context for granted.

```
// Bad:
class AnnualHolidaySale {
  int _annualSaleRebate;
  void promoteHolidaySale() { ... }
}

// Better:
class AnnualHolidaySale {
  int _rebate;
  void promote() { ... }
}
```

The more deeply nested a name is, the more surrounding context it has, so shorter names are appropriate.

### 4. Omit words that don't mean much of anything

Remove filler words that add no semantic value. Usual suspects:
- `data`
- `state`
- `amount`
- `value`
- `manager`
- `engine`
- `object`
- `entity`
- `instance`

A good name paints a picture in the mind of the reader. "Manager" doesn't convey what the thing does. Ask: "Would this identifier mean the same thing if I removed the word?" If so, the word doesn't carry its weight.
