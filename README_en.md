# ccenv

[中文文档](README.md)

A CLI tool for managing Claude Code environment variables via named profiles. Switch API providers per terminal without affecting other sessions.

## Why ccenv?

Claude Code reads its configuration from `~/.claude/settings.json`. When you need to switch between providers (official API, proxy, self-hosted relay), editing `settings.json` takes effect globally — all running Claude Code sessions are affected, potentially interrupting active tasks.

**ccenv makes switching scoped to a single terminal.** Other terminals keep their own environment untouched.

## How It Works

```
~/.claude/settings.json       # Claude Code config (ccenv reads env from here)
~/.claude/ccenv/<name>.profile  # Stored profiles (shell-exportable files)
~/.claude/ccenv.activate      # Symlink → currently active profile
```

- `ccenv use <name>` creates a symlink at `~/.claude/ccenv.activate` pointing to the chosen profile
- You `source` that symlink in your shell to load the environment variables
- `settings.json` is never modified by `use` — only `save` reads from it
- Each terminal independently `source`s whichever profile it needs

## Installation

### Option 1: Download from Releases (Recommended)

Go to [GitHub Releases](https://github.com/ekeyme/claude-code-env/releases) and download the archive for your platform.

```bash
# Example for Linux amd64
tar xzf ccenv_*_linux_amd64.tar.gz
mkdir -p ~/.local/bin
cp ccenv ~/.local/bin/
```

### Option 2: Go Install

```bash
go install github.com/ekeyme/claude-code-env/cmd/ccenv@latest
```

### Option 3: Build from Source

```bash
git clone https://github.com/ekeyme/claude-code-env.git
cd claude-code-env
make install
```

`make install` does the following:
1. Builds the `ccenv` binary → `~/.local/bin/ccenv`
2. Installs helper scripts:
   - `~/.local/bin/ccenv-claude` — launch claude with a specific profile
   - `~/.local/bin/vscode-claude-wrapper.sh` — VSCode Claude Code extension integration
   - `~/.claude/ccenv.deactivate` — clear ANTHROPIC_ variables

> **Note:** If you downloaded a pre-built binary, you can find the helper scripts in the release archive. Install them manually if needed.

```bash
cp ccenv ccenv-claude vscode-claude-wrapper.sh ~/.local/bin/
chmod +x ~/.local/bin/ccenv ~/.local/bin/ccenv-claude ~/.local/bin/vscode-claude-wrapper.sh
mkdir -p ~/.claude
cp ccenv.deactivate ~/.claude/
```

### Post-Install

Ensure `~/.local/bin` is in your `PATH`:

```bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

## Quick Start

```bash
# 1. Configure your provider in Claude Code first (edit ~/.claude/settings.json)
#    Make sure the "env" field has your API credentials.

# 2. Save the current config as a named profile
ccenv save official

# 3. Switch to another provider in settings.json, then save that too
ccenv save proxy

# 4. Activate a profile in the current terminal
ccenv use proxy
source ~/.claude/ccenv.activate

# 5. Now this terminal uses "proxy", others are unaffected
```

## Commands

```
ccenv status              Show settings.json env, active profile, and shell ANTHROPIC_ variables
ccenv save <name>         Save current settings.json env as a profile
ccenv use <name>          Activate a profile (creates symlink)
ccenv list                List all profiles (marks active one)
ccenv show <name>         Display profile contents
ccenv delete <name>       Delete a profile (deactivates if active)
ccenv rename <old> <new>  Rename a profile (updates activation if active)
ccenv -v                  Show version
```

### Command Examples

```bash
$ ccenv status
settings.json env:
  ANTHROPIC_AUTH_TOKEN=sk-a****
  ANTHROPIC_BASE_URL=https://api.anthropic.com

已激活 profile: proxy
  ANTHROPIC_AUTH_TOKEN=sk-p****
  ANTHROPIC_BASE_URL=https://proxy.example.com

当前环境中的 ANTHROPIC_ 变量:
  ANTHROPIC_AUTH_TOKEN=sk-p****
  ANTHROPIC_BASE_URL=https://proxy.example.com

$ ccenv save dev
已保存 profile 'dev' (2 个变量)

$ ccenv list
已保存的 profile:
  dev
  proxy (已激活)

$ ccenv show dev
profile 'dev':
  ANTHROPIC_AUTH_TOKEN=sk-d****
  ANTHROPIC_BASE_URL=https://dev.example.com

$ ccenv use dev
已激活 profile 'dev' (2 个变量)
生效方式: source /home/user/.claude/ccenv.activate

$ ccenv delete old-profile
已删除 profile 'old-profile'
```

## Usage Scenarios

### 1. Switching Between Multiple Providers

You have official API access and a regional proxy. Save both as profiles and switch freely:

```bash
ccenv save official    # Save official API credentials
ccenv save proxy-cn    # Save proxy credentials
ccenv use proxy-cn     # Activate proxy in this terminal
source ~/.claude/ccenv.activate
```

### 2. Multiple Terminals, Different Providers Simultaneously

Terminal A is running a long task with the official API. You open Terminal B and switch to a different provider — Terminal A keeps running undisturbed:

```bash
# Terminal B
ccenv use proxy
source ~/.claude/ccenv.activate
claude
```

### 3. VSCode Integration

The VSCode Claude Code extension launches `claude` as a subprocess. Normally it inherits the system environment — which profile (if any) is active depends on what you last set in your terminal.

**How it works:** `vscode-claude-wrapper.sh` replaces the `claude` binary path. Every time VSCode starts a Claude session, the wrapper:

1. Clears existing `ANTHROPIC_` variables (via `ccenv.deactivate`)
2. Sources `~/.claude/ccenv.activate` (the symlink pointing to your active profile)
3. `exec claude "$@"` — launches the real `claude` with the profile's environment

**Switching profiles:** To change which provider VSCode uses, simply run `ccenv use <name>` in any terminal. This updates the `ccenv.activate` symlink. The next Claude session in VSCode will pick up the new profile automatically.

Setup:

```json
{
  "claude.coder.path": "/home/user/.local/bin/vscode-claude-wrapper.sh"
}
```

> **Note:** Already-running Claude sessions in VSCode keep their original environment. The new profile takes effect for sessions started *after* the switch.

### 4. Quick Launch with ccenv-claude

`ccenv-claude` sources a profile and launches Claude in one step — no manual `source` needed.

**How it works:** Same principle as the VSCode wrapper, but for terminal use. It deactivates any current `ANTHROPIC_` variables, sources the chosen profile file, then passes remaining arguments to `claude`.

```bash
# Use the currently active profile (reads ccenv.activate symlink)
ccenv-claude

# Use a specific profile directly (ignores the active symlink)
ccenv-claude proxy-cn

# Pass Claude arguments when using a specific profile
ccenv-claude proxy-cn --resume

# Interactive selection from all saved profiles
ccenv-claude list
```

This is useful when you want to quickly spin up a Claude session with a specific provider without permanently changing your active profile.

### 5. Team Profile Sharing

Profile files are plain shell scripts. Share them with your team:

```bash
# Export a profile
cp ~/.claude/ccenv/dev.profile ./shared/dev.profile

# Import on another machine
cp ./shared/dev.profile ~/.claude/ccenv/dev.profile
```

## Security

- **Auto-masking**: Values with keys containing `token`, `key`, `secret`, `password`, `credential`, or `private` are masked in `status` and `show` output (first 4 chars visible, rest replaced with `****`)
- **File permissions**: Profile files are written with `0600`, directories with `0700`
- **Atomic writes**: Files are written to a temp file first, then renamed — prevents data corruption on crash

## Deactivate

To clear all `ANTHROPIC_` variables in the current shell:

```bash
source ~/.claude/ccenv.deactivate
```

## License

[MIT](LICENSE)
