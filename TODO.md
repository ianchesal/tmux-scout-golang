# TODO

Things that are on my list to knock off.

## Test with Codex

I don't actually use Codex so none of the Codex paths are actually tested at this point. Only the Claude paths are known to work. I should rectify that at some point and actually test with Codex.

## Add a screen recording to the README

I ended up dropping the one that came from the original repo because it was for the original plugin and not this version.

## Provide pre-built binaries for easier installation

Remove the Go requirement for end users by shipping pre-built binaries as GitHub Release assets and having `tmux-scout.tmux` download the right one on first load.

### CI changes

Add a release workflow (`.github/workflows/release.yml`) that triggers on `v*` tags and:

1. Runs `make release` to cross-compile all four targets:
   - `tmux-scout-linux-amd64`
   - `tmux-scout-linux-arm64`
   - `tmux-scout-darwin-amd64`
   - `tmux-scout-darwin-arm64`
2. Uploads each binary as a GitHub Release asset alongside a `SHA256SUMS` file.

### Plugin changes

Update `tmux-scout.tmux` to replace the current auto-build block with a download-or-build block:

1. Detect the platform (`uname -s` / `uname -m`) and map it to the binary name.
2. Determine the installed version tag (e.g. from a `.version` file committed to the repo or from `git describe`).
3. Download the matching binary from:
   ```
   https://github.com/ianchesal/tmux-scout-golang/releases/download/<tag>/<binary-name>
   ```
4. Verify the checksum against `SHA256SUMS`.
5. Make the binary executable and place it at `bin/tmux-scout`.
6. Fall back to `go build` if `curl`/`wget` are unavailable or the download fails, with a clear error message if Go is also absent.

### README changes

- Update the TPM section: remove the Go requirement note, explain the binary is auto-downloaded.
- Keep a "Building from Source" note for contributors / unsupported platforms.

### Considerations

- Checksums should be verified before execution to prevent tampered downloads.
- The fallback chain (download → go build → error) keeps the plugin usable in air-gapped or Go-only environments.
- Version pinning: the download URL must match the checked-out tag, not always `latest`, to avoid mismatches when a user has an older TPM-installed version.
