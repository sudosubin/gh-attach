---
name: gh-attach
description: Uploads a local file (screenshot, image, PDF, zip, video) to GitHub user-attachments and embeds it in a PR, issue, or comment. Use when asked to "attach a screenshot to the PR", "add an image to the issue", "embed before/after screenshots", or "attach this file". Powered by `gh-attach`.
license: MIT
---

# gh-attach

`gh attach` uploads a file to GitHub's internal user-attachments endpoint (no public API exists) and prints the URL, which GitHub auto-renders (image/video/file) wherever it's pasted. The URL inherits the repo's visibility, so private-repo uploads stay private.

## Prerequisites

```sh
gh extension list | grep -q 'gh attach' || gh extension install sudosubin/gh-attach
```

`gh` must be authenticated (interactive `gh auth login` if it reports a 404/auth error). Uploads use your **browser** session cookie, not the `gh` token; wrong account → add `--browser <name> --profile <name>`. No headless/CI support.

## Steps

**1. Upload**: absolute quoted path; `-R` optional inside a repo (GHES: `-R host/owner/repo`). Prints one line, the URL; GitHub auto-renders it (image/video/file) so use it as-is:

```sh
URL=$(gh attach "$FILE" -R <owner>/<repo>)
```

**2. Embed** (always `--body-file -`, e.g. `gh pr comment/edit`, `gh issue comment/edit`):

```sh
printf '## Screenshots\n\n%s\n' "$URL" | gh pr comment <pr> -R <owner>/<repo> --body-file -
```

## Notes

- Private repo: URL renders only for authorized viewers; anonymous fetch 404/403 is expected.
- Sizing: embed `<img width="800" src="$URL">` instead of the bare URL.
- Works for images and clean-mime binaries (PDF, zip). Text files (`.txt/.log/.md/.csv/.html`) currently 422 pending a content-type fix.
