---
name: release
description: Create a new release of tmux-scout-golang. Use this skill whenever the user asks to cut a release, create a new version, bump the version, tag a release, or ship a new version of this project.
---

# Release Skill for tmux-scout-golang

This skill guides you through creating a new release of tmux-scout-golang. The release process requires a PR (main is branch-protected), so it has two phases: the PR phase and the tag phase.

## Step 1: Determine the new version

Read the current version:

```bash
cat .version
```

Parse the version (format: `vMAJOR.MINOR.PATCH`) and compute the three candidates:
- **patch**: increment PATCH (e.g. v0.4.2 → v0.4.3)
- **minor**: increment MINOR, reset PATCH to 0 (e.g. v0.4.2 → v0.5.0)
- **major**: increment MAJOR, reset MINOR and PATCH to 0 (e.g. v0.4.2 → v1.0.0)

Present the options to the user using `AskUserQuestion` with three choices showing the computed version number for each type. Wait for their selection before proceeding.

## Step 2: Create the release branch and PR

Once the user picks a version type:

1. Create a branch: `git checkout -b release/<new-version>`
2. Update `.version` with the new version string (no trailing newline issues — write exactly `vMAJOR.MINOR.PATCH`)
3. Stage and commit: `git add .version && git commit -m "chore: bump version to <new-version>"`
4. Push: `git push -u origin release/<new-version>`
5. Open a PR with `gh pr create`:
   - Title: `chore: bump version to <new-version>`
   - Body should mention that after merge the user should confirm so the tag can be created

## Step 3: Wait for merge, then tag

After creating the PR, tell the user:

> "PR is open at <url>. Merge it and let me know when it's done — I'll then pull main and run `make tag` to create the `<new-version>` tag and trigger the GitHub release."

When the user confirms the PR is merged:

1. `git checkout main && git pull`
2. `make tag`

`make tag` handles everything: verifies the working tree is clean, runs tests, creates the tag, and pushes it to trigger the GitHub Actions release workflow.

## Notes

- Never push commits directly to main — branch protection is enforced on the remote.
- The `make tag` command is authoritative for release preconditions; if it fails, read its error output and fix the issue before retrying.
- Version format is always `vMAJOR.MINOR.PATCH` with a lowercase `v` prefix.
