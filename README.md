<div align="center">

# gh-attach

[![version](https://badgen.net/github/release/sudosubin/gh-attach?label=version)](https://github.com/sudosubin/gh-attach/releases)
[![license](https://badgen.net/github/license/sudosubin/gh-attach?color=green)](./LICENSE)
[![downloads](https://img.shields.io/github/downloads/sudosubin/gh-attach/total?color=green)](https://github.com/sudosubin/gh-attach/releases)

A GitHub CLI extension that uploads files to GitHub attachments.

<a href="docs/assets/gh-attach-demo.webp">
  <img src="docs/assets/gh-attach-demo.webp" alt="gh-attach demo" width="800" />
</a>

</div>

## Quick Start

```sh
gh extension install sudosubin/gh-attach
gh attach ./image.png -R owner/repo
```

## Installation

Requires [GitHub CLI](https://github.com/cli/cli#installation). If you haven't authenticated yet, run `gh auth login`.

```sh
gh extension install sudosubin/gh-attach
gh attach ./image.png -R owner/repo
```

## Usage

```sh
$ gh attach ./image.png -R owner/repo
https://github.com/user-attachments/assets/550e8400-e29b-41d4-a716-446655440000

$ gh attach ./image.png ./report.pdf -R owner/repo
https://github.com/user-attachments/assets/550e8400-e29b-41d4-a716-446655440000
https://github.com/user-attachments/files/123/report.pdf

$ gh attach ./image.png -R owner/repo --browser chrome --profile Default
https://github.com/user-attachments/assets/550e8400-e29b-41d4-a716-446655440000

$ gh attach ./image.png --json id,href,name
[
  {
    "href": "https://github.com/user-attachments/assets/550e8400-e29b-41d4-a716-446655440000",
    "id": 123,
    "name": "image.png"
  }
]

$ gh attach ./image.png --json href --jq '.[].href'
https://github.com/user-attachments/assets/550e8400-e29b-41d4-a716-446655440000

$ gh attach ./image.png --json href,name --template '{{range .}}{{.name}} -> {{.href}}{{"\n"}}{{end}}'
image.png -> https://github.com/user-attachments/assets/550e8400-e29b-41d4-a716-446655440000
```

Multiple files are uploaded sequentially, and per-file failures do not stop the remaining uploads. With `--json`, results are always returned as an array.

### Options

- `-R, --repo <[HOST/]OWNER/REPO>`: Target repository. Auto detection is available from current repository.
- `--browser <name>`: Browser to read cookies from (`auto|arc|atlas|brave|chrome|chromium|comet|dia|edge|firefox|floorp|helium|librewolf|opera|safari|vivaldi|waterfox|whale|zen`).
- `--profile <name>`: Browser profile name. For Firefox-family multi-account containers, append `:<container-name>` or `:id=<container-id>` to pin a specific container (e.g. `default:Work`, `default:id=2`).
- `--cookie-store-path <path>`: Explicit cookie DB file path.
- `--json <fields>`: Output JSON with selected fields.
- `-q, --jq <expression>`: Apply jq filter to JSON output (requires `--json`).
- `-t, --template <go-template>`: Format JSON output using Go template (requires `--json`).
- `-v, --verbose`: Print cookie source resolution logs to stderr.
- `-h, --help`: Show help.

## Configuration

You can use a config file to register frequently used browser settings without having to pass them as command line arguments each time.

The config file is loaded from `${XDG_CONFIG_HOME:-~/.config}/gh/attach.yml`.

**Example**

```yaml
browsers:
  - browser: chrome
    profile: Default
  - browser: firefox
    profile: default-release
  - browser: safari
```

**Schema**

- `browser`: Browser to read cookies from (Required, one of `auto|arc|atlas|brave|chrome|chromium|comet|dia|edge|firefox|floorp|helium|librewolf|opera|safari|vivaldi|waterfox|whale|zen`)
- `profile`: Browser profile name/path (Optional, name or path)
- `cookie_store_path`: Explicit cookie DB file path (Optional)

## Supported Browsers

- Chromium family (Arc, Atlas, Brave, Chrome, Chromium, Comet, Dia, Edge, Helium, Opera, Vivaldi, Whale)
- Firefox family (Firefox, Floorp, LibreWolf, Waterfox, Zen)
- Safari

## How It Works

- It first resolves the target repository (`owner/repo`) and the current GitHub login via the `gh` API.
- Based on CLI flags or config file, it looks up browser cookie sources and selects a session whose [`dotcom_user`](https://docs.github.com/en/site-policy/privacy-policies/github-cookies#cookies) matches the current login.
- Using that session cookie, it requests GitHub upload policies (`/upload/policies/assets`) and uploads each file binary.
- It finalizes each user-attachments asset and prints the results as URLs or formatted output via `--json`.

## Development

```sh
go test ./...
go build ./...
```

## License

MIT, see [LICENSE](./LICENSE).
