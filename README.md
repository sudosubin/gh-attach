# gh-attach

![release](https://badgen.net/github/release/sudosubin/gh-attach)

GitHub user attachment upload CLI for `gh` (GitHub CLI).

## Supported Browser

- Chromium Family (Chrome, Chromium, Edge, Brave, Vivaldi, Opera)
- Firefox
- Safari

## Usage

```bash
$ gh attach ./image.png -R owner/repo
https://github.com/user-attachments/assets/550e8400-e29b-41d4-a716-446655440000

$ gh attach ./image.png -R owner/repo --browser chrome --profile Default
https://github.com/user-attachments/assets/550e8400-e29b-41d4-a716-446655440000

$ gh attach ./image.png --json id,href,name
{
  "href": "https://github.com/user-attachments/assets/550e8400-e29b-41d4-a716-446655440000",
  "id": 123,
  "name": "image.png"
}

$ gh attach ./image.png --json href,name --template '{{.name}} -> {{.href}}'
image.png -> https://github.com/user-attachments/assets/550e8400-e29b-41d4-a716-446655440000
```

### Options

- `-R, --repo <[HOST/]OWNER/REPO>`: Target repository. Auto detection is available from current repository.
- `--browser <name>`: Browser to read cookies from (`auto|chrome|chromium|edge|firefox|safari|brave|vivaldi|opera`).
- `--profile <name-or-path>`: Browser profile name/path.
- `--cookie-store-path <path>`: Explicit cookie DB file path.
- `--json <fields>`: Output JSON with selected fields.
- `-q, --jq <expression>`: Apply jq filter to JSON output (requires `--json`).
- `-t, --template <go-template>`: Format JSON output using Go template (requires `--json`).
- `-v, --verbose`: Print cookie source resolution logs to stderr.
- `-h, --help`: Show help.

## Config File

You can use config file to register frequently used browser settings without having to pass them as command line arguments each time.

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

- `browser`: Browser to read cookies from (Required, one of `auto|chrome|chromium|edge|firefox|safari|brave|vivaldi|opera`)
- `profile`: Browser profile name/path (Optional, name or path)
- `cookie_store_path`: Explicit cookie DB file path (Optional)
