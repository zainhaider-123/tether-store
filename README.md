# Tether

Agent skill management CLI. Install skills into a shared cache, declare what is **active** in hierarchical JSON config, and symlink into AI agent skill directories.

## Install

### Download (Windows / macOS / Linux)

Pick your platform — these always point at the [latest release](https://github.com/zainhaider-123/tether-store/releases/latest):

| OS | Arch | Download |
| --- | --- | --- |
| Linux | x86_64 | [tether_linux_amd64.tar.gz](https://github.com/zainhaider-123/tether-store/releases/latest/download/tether_linux_amd64.tar.gz) |
| Linux | ARM64 | [tether_linux_arm64.tar.gz](https://github.com/zainhaider-123/tether-store/releases/latest/download/tether_linux_arm64.tar.gz) |
| macOS | Apple Silicon | [tether_darwin_arm64.tar.gz](https://github.com/zainhaider-123/tether-store/releases/latest/download/tether_darwin_arm64.tar.gz) |
| macOS | Intel | [tether_darwin_amd64.tar.gz](https://github.com/zainhaider-123/tether-store/releases/latest/download/tether_darwin_amd64.tar.gz) |
| Windows | x86_64 | [tether_windows_amd64.zip](https://github.com/zainhaider-123/tether-store/releases/latest/download/tether_windows_amd64.zip) |
| Windows | ARM64 | [tether_windows_arm64.zip](https://github.com/zainhaider-123/tether-store/releases/latest/download/tether_windows_arm64.zip) |

All releases: https://github.com/zainhaider-123/tether-store/releases

**Linux / macOS**

```bash
# Example: Linux x86_64
curl -sL https://github.com/zainhaider-123/tether-store/releases/latest/download/tether_linux_amd64.tar.gz | tar xz
chmod +x tether
sudo mv tether /usr/local/bin/tether
# or: mv tether ~/.local/bin/tether
```

**Windows (PowerShell)**

```powershell
Invoke-WebRequest -Uri https://github.com/zainhaider-123/tether-store/releases/latest/download/tether_windows_amd64.zip -OutFile tether.zip
Expand-Archive tether.zip -DestinationPath .
Move-Item tether.exe $env:LOCALAPPDATA\Microsoft\WindowsApps\
```

Then verify:

```bash
tether --help
```

### From source (Go required)

```bash
go install github.com/zainhaider-123/tether-store/cmd/tether@latest
# or from this repo:
go build -o bin/tether ./cmd/tether
# optional: install onto PATH
cp bin/tether ~/.local/bin/tether   # Linux/macOS
```

## Development setup

Prerequisites: **Go 1.26+**, **git**, and Make (optional).

```bash
git clone https://github.com/zainhaider-123/tether-store.git
cd tether-store
make build          # → bin/tether
make test
make run ARGS="ls"  # or: make install  (go install into GOBIN)
```

Useful targets: `make help` lists `build`, `run`, `dev`, `test`, `vet`, `tidy`, `install`, `clean`.

CI runs on every push/PR to `main` (`.github/workflows/ci.yml`).  
Publishing a release: push a tag like `v0.1.0` — GoReleaser builds all platform archives (`.github/workflows/release.yml`).

## Quick start

```bash
# Search the skills.sh catalog
tether search frontend

# Install into cache, write ./tether.json, symlink into selected agents
tether use vercel-labs/agent-skills@web-design-guidelines

# Non-interactive (default agents: cursor + agents)
tether use vercel-labs/agent-skills@web-design-guidelines -y

# Global activation
tether use vercel-labs/agent-skills@web-design-guidelines -g -y

# Ensure cache + relink for everything declared in the current tree
tether install

# List active skills / full cache
tether ls
tether ls-cache

# Remove from project config and unlink
tether unuse web-design-guidelines

# Refresh cached skill bodies
tether upgrade
```

## How it works

| Concern | Location |
| --- | --- |
| Install cache | `~/.local/share/tether/skills/<name>/` |
| Global active set | `~/.config/tether/tether.json` |
| Project active set | `./tether.json` (commit this) |
| Personal overrides | `./tether.local.json` (gitignore) |

`use` = install into cache + write config + activate.  
`install` = ensure cache for skills already in config + (re)link. Does not add new skills.

Config is discovered by walking **up from cwd**, merging nearer over farther, with the global file as the baseline.

### Source formats

```
owner/repo
owner/repo@skill-name
https://github.com/owner/repo
https://github.com/owner/repo/tree/<ref>/path/to/skill
```

### Agents

On interactive `use`, tether prompts with checkboxes (defaults: **cursor**, **agents**). Non-interactive:

```bash
tether use owner/repo@skill -a cursor,claude-code -y
tether use owner/repo@skill -a '*' -y   # all registered agents
```

Symlink target is always the share cache. Project `use` writes project agent paths; `-g` writes global paths.

## Project config example

```json
{
  "version": 1,
  "settings": {
    "agents": ["cursor", "agents"]
  },
  "skills": {
    "web-design-guidelines": {
      "source": "vercel-labs/agent-skills@web-design-guidelines",
      "agents": ["cursor", "agents"]
    }
  }
}
```

## Commands

| Command | Alias | Description |
| --- | --- | --- |
| `tether use <source>` | `u` | Cache + config + activate |
| `tether install [names…]` | `i` | Cache + relink active skills |
| `tether unuse <name>` | | Remove from config + unlink |
| `tether ls` | | Active skills in current context |
| `tether ls-cache` | | Everything in the share cache |
| `tether search [query]` | | Query skills.sh |
| `tether upgrade [names…]` | | Re-fetch cache + refresh links |

Flags: `-g` / `--global`, `-s` / `--skill`, `-a` / `--agent`, `-y` / `--yes`, `--all`.
