#!/usr/bin/env bash
CURRENT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Determine installed version and expected version
_expected=$(cat "$CURRENT_DIR/.version" 2>/dev/null)
_installed=""
if [ -x "$CURRENT_DIR/bin/tmux-scout" ]; then
  _installed=$("$CURRENT_DIR/bin/tmux-scout" --version 2>/dev/null)
fi

if [ "$_installed" != "$_expected" ]; then
  # Detect platform
  _uname_s=$(uname -s)
  _uname_m=$(uname -m)
  case "$_uname_s" in
    Linux)  _os="linux"  ;;
    Darwin) _os="darwin" ;;
    *)      _os=""       ;;
  esac
  case "$_uname_m" in
    x86_64|amd64) _arch="amd64" ;;
    arm64|aarch64) _arch="arm64" ;;
    *)             _arch=""      ;;
  esac

  _downloaded=false
  if [ -n "$_os" ] && [ -n "$_arch" ] && [ -n "$_expected" ]; then
    _binary_name="tmux-scout-${_os}-${_arch}"
    _tmp_binary="/tmp/${_binary_name}"
    _tmp_sums="/tmp/tmux-scout-SHA256SUMS"
    _base_url="https://github.com/ianchesal/tmux-scout-golang/releases/download/${_expected}"

    # Try download
    _dl_ok=false
    if command -v curl >/dev/null 2>&1; then
      if curl -fsSL -o "$_tmp_binary" "${_base_url}/${_binary_name}" 2>/dev/null && \
         curl -fsSL -o "$_tmp_sums"   "${_base_url}/SHA256SUMS"       2>/dev/null; then
        _dl_ok=true
      fi
    elif command -v wget >/dev/null 2>&1; then
      if wget -q -O "$_tmp_binary" "${_base_url}/${_binary_name}" 2>/dev/null && \
         wget -q -O "$_tmp_sums"   "${_base_url}/SHA256SUMS"       2>/dev/null; then
        _dl_ok=true
      fi
    fi

    if [ "$_dl_ok" = "true" ]; then
      # Verify checksum
      _ck_ok=false
      case "$_uname_s" in
        Linux)
          if (cd /tmp && sha256sum -c tmux-scout-SHA256SUMS --ignore-missing 2>/dev/null); then
            _ck_ok=true
          fi
          ;;
        Darwin)
          if (cd /tmp && shasum -a 256 -c tmux-scout-SHA256SUMS --ignore-missing 2>/dev/null); then
            _ck_ok=true
          fi
          ;;
      esac
      rm -f "$_tmp_sums"

      if [ "$_ck_ok" = "true" ]; then
        mkdir -p "$CURRENT_DIR/bin"
        mv "$_tmp_binary" "$CURRENT_DIR/bin/tmux-scout"
        chmod +x "$CURRENT_DIR/bin/tmux-scout"
        _downloaded=true
      else
        rm -f "$_tmp_binary"
        tmux display-message "tmux-scout: checksum verification failed. Binary may be tampered. Remove bin/tmux-scout and retry."
      fi
    else
      rm -f "$_tmp_binary" "$_tmp_sums"
    fi
  fi

  # Fall back to go build if download didn't succeed
  if [ "$_downloaded" = "false" ]; then
    if command -v go >/dev/null 2>&1; then
      _ver_flag=$(cat "$CURRENT_DIR/.version" 2>/dev/null || echo "dev")
      tmux display-message "tmux-scout: building binary (requires Go)..."
      if ! (cd "$CURRENT_DIR" && go build -ldflags "-X main.version=${_ver_flag}" -o bin/tmux-scout . 2>/tmp/tmux-scout-build.log); then
        tmux display-message "tmux-scout: download failed and build failed — see /tmp/tmux-scout-build.log"
        return 1 2>/dev/null || exit 1
      fi
    else
      tmux display-message "tmux-scout: download failed and Go not found. Install Go or manually place the binary at bin/tmux-scout"
      return 1 2>/dev/null || exit 1
    fi
  fi
fi

key=$(tmux show-option -gqv "@scout-key")
[ -z "$key" ] && key="O"
tmux set-environment -g SCOUT_DIR "$CURRENT_DIR"
tmux bind-key "$key" run-shell -b "$CURRENT_DIR/scripts/picker/picker.sh"
tmux run-shell -b "\"$CURRENT_DIR/bin/tmux-scout\" setup status --quiet 2>/dev/null || tmux display-message 'tmux-scout: hooks not installed. Run: $CURRENT_DIR/scripts/setup.sh install'"

# Status bar widget — users add #($SCOUT_DIR/scripts/status-widget.sh) to their status-right config
