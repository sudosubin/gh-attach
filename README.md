# gh-attach

GitHub user attachments upload CLI (WIP).

## Supported Browser

- Chromium Family (Chrome, Chromium, Edge, Brave, Vivaldi, Opera)
- Firefox
- Safari

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

- `browser`: Browser (Required, one of `auto|chrome|chromium|edge|firefox|safari|brave|vivaldi|opera`)
- `profile`: Browser Profile (Optional, name or path)
- `cookie_store_path`: Explicit cookie DB path (Optional)
