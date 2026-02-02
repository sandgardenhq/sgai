# GOAL.md

`sgai` uses `GOAL.md` in your project directory as the source of truth for the workflow goal and its task checklist.

## Task checkboxes

### Checkbox ownership

Only the **coordinator** agent updates checkbox state in `GOAL.md`.

- Coordinator marks completed items by editing `GOAL.md` and changing `- [ ]` to `- [x]`.
- Non-coordinator agents do **not** edit `GOAL.md` checkboxes directly.

### Reporting completion (non-coordinator agents)

When a non-coordinator agent finishes a task listed in `GOAL.md`, it reports completion to the coordinator by sending a message that starts with `GOAL COMPLETE:`.

Use this format:

```text
GOAL COMPLETE: [exact checkbox text from GOAL.md]
```

## Coordinator workflow for marking tasks complete

1. Check the current `GOAL.md` status using the `project-completion-verification` skill.
2. Verify there is evidence the work is complete (for example: test results, code review, agent confirmation).
3. Edit `GOAL.md` and mark the item complete by changing `- [ ]` to `- [x]`.
4. Re-run the `project-completion-verification` skill to confirm the updated status.
5. Log the checkbox change in `.sgai/PROJECT_MANAGEMENT.md`.

## Notes

- Treat `GOAL COMPLETE:` messages as a trigger to verify work and update the matching checkbox.
- Marking checkboxes without verification is worse than leaving them unchecked.
