---
name: release
description: Create a new release — bump version, generate changelog, create git tag, and publish GitHub release via goreleaser.
allowed-tools: Read, Write, Edit, Glob, Grep, Bash, AskUserQuestion
disable-model-invocation: true
argument-hint: "[optional version hint like 'patch' or 'minor']"
---

# Release Command

Create a new release: bump version, generate changelog, create git tag, and publish GitHub release.

---

## Flow

```text
1. PREPARE  → Switch to main, pull, find latest tag
2. CHANGELOG → Show changes since last tag
3. BUMP     → Ask which version part to bump
4. TAG      → Create git tag
5. RELEASE  → Push tag + create GitHub release via gh/goreleaser
```

---

## Step 1: PREPARE

Switch to main and pull latest changes. Find the latest tag.

```bash
git checkout main
git pull
```

Get the latest tag:
```bash
git describe --tags --abbrev=0
```

Store it as `LAST_TAG`. Parse into MAJOR, MINOR, PATCH components.

---

## Step 2: CHANGELOG

Show commits since the last tag:

```bash
git log ${LAST_TAG}..HEAD --oneline --no-merges
```

Also show the full log with authors for the release notes:
```bash
git log ${LAST_TAG}..HEAD --pretty=format:"- %s (%an)" --no-merges
```

Display this to the user as a summary of what's in this release.

If there are NO commits since the last tag, warn the user and ask if they want to continue.

---

## Step 3: BUMP VERSION

Ask the user which part to bump using AskUserQuestion:

Current version: `LAST_TAG` (e.g., v1.5.2)

Options:
- **patch** → vX.Y.(Z+1) — bug fixes, small changes
- **minor** → vX.(Y+1).0 — new features
- **major** → v(X+1).0.0 — breaking changes

Calculate the new version based on selection. Store as `NEW_VERSION`.

**IMPORTANT:** Check existing tag format (with or without `v` prefix) and match it.

---

## Step 4: TAG

Create an annotated tag:

```bash
git tag -a {NEW_VERSION} -m "Release {NEW_VERSION}"
```

---

## Step 5: PUSH & GITHUB RELEASE

Push the tag:

```bash
git push origin main
git push origin {NEW_VERSION}
```

Create GitHub release using gh CLI. Use the changelog content from Step 2:

```bash
gh release create {NEW_VERSION} --title "{NEW_VERSION}" --notes "$(cat <<'NOTES'
{release notes content}
NOTES
)"
```

If goreleaser is configured (`.goreleaser.yaml` exists), mention that goreleaser will handle binary builds via CI.

---

## Final Output

Show the user:
```text
Release {NEW_VERSION} created!

- Tag: {NEW_VERSION}
- GitHub release: {URL from gh output}
- Commits included: {count}
```

---

## Error Handling

- If `git checkout main` fails due to uncommitted changes → warn and stop
- If `git pull` has conflicts → warn and stop
- If no commits since last tag → ask user if they want to proceed anyway
- If `gh` is not installed or not authenticated → warn and stop
- If tag already exists → warn and stop

---

## Start

$ARGUMENTS

Run through the steps above sequentially. Ask for confirmation before creating the tag and pushing (Step 4-5).
