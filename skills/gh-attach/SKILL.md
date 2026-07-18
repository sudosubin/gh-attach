---
name: gh-attach
description: Uploads a local file (screenshot, image, PDF, zip, video) to GitHub user-attachments and embeds it in a PR, issue, or comment. Use when asked to "attach a screenshot to the PR", "add an image to the issue", "embed before/after screenshots", or "attach this file". Powered by `gh-attach`.
license: MIT
---

# gh-attach

`gh attach` uploads a file to GitHub's internal user-attachments endpoint (no public API exists) and prints the **URL only**: you build the markdown. The URL inherits the repo's visibility, so private-repo uploads stay private.

## Prerequisites

```sh
gh extension list | grep -q 'gh attach' || gh extension install sudosubin/gh-attach
```

`gh` must be authenticated for the target host (`gh auth login` if `gh attach` reports a 404/auth error; never run it unattended, it is interactive). The upload itself uses your **browser** `user_session` cookie (not the `gh` token), auto-detected across Chrome/Firefox/Safari/Arc/… Wrong account → add `--browser <name> --profile <name>`. Not for headless/CI (needs a local browser login).

## Steps

**1. Upload**: one file per call, absolute quoted path; `-R` optional inside a repo (GHES: `-R host/owner/repo`):

```sh
IFS=$'\t' read -r NAME HREF CT < <(gh attach "$FILE" -R <owner>/<repo> --json name,href,content_type --jq '[.name,.href,.content_type]|@tsv')
```

**2. Build the reference** from the content type:

```sh
case "$CT" in image/*) MD="![$NAME]($HREF)";; video/*) MD="$HREF";; *) MD="[$NAME]($HREF)";; esac
```

**3. Embed**: always `--body-file -` (safe multi-line):

```sh
printf '## Screenshots\n\n%s\n' "$MD" | gh pr comment <pr> -R <owner>/<repo> --body-file -
# variants: gh pr edit <pr> --body-file - | gh issue comment|edit <n> --body-file -
```

## Notes

- Private repo: URL renders only for authorized viewers; anonymous fetch 404/403 is expected.
- Sizing: embed `<img width="800" src="$HREF">` instead of the bare markdown.
- Works for images and clean-mime binaries (PDF, zip). Text files (`.txt/.log/.md/.csv/.html`) currently 422 pending a content-type fix.
