#!/usr/bin/env bash
set -euo pipefail

# Steps:
# 1. Choose the installation directory.
# 2. Detect the operating system and CPU architecture.
# 3. Download and verify the matching release binary.
# 4. Install the binary.
# 5. Add the installation directory to PATH when needed.

# --- Config ---
repo="Tarun-Elango/outpost"

# When run via sudo, the binary and shell config should target the invoking
# user, not root — otherwise `sudo bash install.sh` installs to /root while
# the PATH line goes into the real user's rc file (or vice versa).
config_home="${HOME}"
config_shell="${SHELL:-}"
if [ "${EUID}" -eq 0 ] && [ -n "${SUDO_USER:-}" ] && [ "${SUDO_USER}" != "root" ]; then
  config_home="$(eval echo "~${SUDO_USER}")"
  if command -v getent >/dev/null 2>&1; then
    config_shell="$(getent passwd "${SUDO_USER}" | cut -d: -f7)"
  elif [ "$(uname -s)" = "Darwin" ]; then
    config_shell="$(dscl . -read "/Users/${SUDO_USER}" UserShell 2>/dev/null | awk '{print $2}')"
  fi
fi

# Where to install the binary (override with INSTALL_DIR=...)
install_dir="${INSTALL_DIR:-${config_home}/.local/bin}"

# Reject install dirs containing shell metacharacters: this value gets
# embedded in an `export PATH=...` line appended to the user's shell rc
# file, so command substitution, quotes, or newlines here would let
# INSTALL_DIR inject arbitrary commands that run on every new shell.
case "${install_dir}" in
  *'$'* | *'`'* | *'"'* | *"'"* | *$'\n'*)
    echo "INSTALL_DIR contains disallowed characters (\$, \`, quotes, or newline): ${install_dir}" >&2
    exit 1
    ;;
esac

# --- Detect platform ---
# Map uname output to release asset names
case "$(uname -s)" in
  Darwin) os="darwin" ;;
  Linux) os="linux" ;;
  *)
    echo "Unsupported OS: $(uname -s)" >&2
    exit 1
    ;;
esac

case "$(uname -m)" in
  x86_64 | amd64) arch="amd64" ;;
  arm64 | aarch64) arch="arm64" ;;
  *)
    echo "Unsupported architecture: $(uname -m)" >&2
    exit 1
    ;;
esac

# --- Download ---
# Fetch the release binary for this OS and CPU.
# Pin a version: curl -fsSL .../install.sh | RELEASE_TAG=v0.7.0 bash
# (RELEASE_TAG must be on bash, not curl — a pipe does not pass env vars across.)
release_tag="${RELEASE_TAG:-latest}"
asset_name="outpost-${os}-${arch}"
url="https://github.com/${repo}/releases/download/${release_tag}/${asset_name}"
checksums_url="https://github.com/${repo}/releases/download/${release_tag}/checksums.txt"
tmp="$(mktemp)"
tmp_checksums="$(mktemp)"
trap 'rm -f "$tmp" "$tmp_checksums"' EXIT

echo "Installing release ${release_tag} (${asset_name})..."
curl -fsSL "$url" -o "$tmp"
chmod +x "$tmp"

# --- Verify checksum ---
# Confirm the downloaded binary matches the checksum published alongside
# it in the same release, so a compromised CDN/mirror or partial download
# doesn't get installed and executed.
echo "Verifying checksum..."
curl -fsSL "$checksums_url" -o "$tmp_checksums"

expected_line="$(grep -F "  ${asset_name}" "$tmp_checksums" || true)"
if [ -z "${expected_line}" ]; then
  echo "Could not find a checksum entry for ${asset_name} in checksums.txt" >&2
  exit 1
fi
expected_sha="$(printf '%s' "${expected_line}" | awk '{print $1}')"

if command -v sha256sum >/dev/null 2>&1; then
  actual_sha="$(sha256sum "$tmp" | awk '{print $1}')"
elif command -v shasum >/dev/null 2>&1; then
  actual_sha="$(shasum -a 256 "$tmp" | awk '{print $1}')"
else
  echo "Neither sha256sum nor shasum is available; cannot verify checksum." >&2
  exit 1
fi

if [ "${expected_sha}" != "${actual_sha}" ]; then
  echo "Checksum mismatch for ${asset_name}!" >&2
  echo "  expected: ${expected_sha}" >&2
  echo "  actual:   ${actual_sha}" >&2
  exit 1
fi
echo "Checksum OK."

# --- Install ---
# Copy the binary into place
mkdir -p "$install_dir"  # create the directory if it doesn't exist
install -m 755 "$tmp" "${install_dir}/outpost"  # copy the binary into the directory
echo "Installed outpost to ${install_dir}/outpost" # print the path of the binary

# --- PATH setup ---
# Purpose: check if the install directory is usable, if not asks the 
# user to add it their shell config, or tells the user to add it manually


# check if the install directory is a system directory, if so, skip the path setup
is_system_bin_dir() { 
  case "$1" in
    /usr/local/bin | /usr/bin | /usr/local/sbin | /usr/sbin) return 0 ;;
  esac
  return 1
}

# System install dirs are usually already on PATH — don't touch anyone's shell config
if is_system_bin_dir "${install_dir}"; then
  echo "${install_dir} is a standard system directory (usually already on PATH)."
  echo "Restart your shell if needed, then run: outpost ls"
  exit 0
fi

# if the user is running as root without sudo, skip the path setup
# tells the user to add it manually
if [ "${EUID}" -eq 0 ] && [ -z "${SUDO_USER:-}" ]; then
  echo "Running as root without sudo; skipping shell config updates."
  echo "Add this to your shell config manually:"
  echo "  export PATH=\"${install_dir}:\$PATH\""
  exit 0
fi

# Skip if install dir is already on PATH
if echo ":${PATH}:" | grep -q ":${install_dir}:"; then
  exit 0
fi

# checks if the install directory is already configured in the shell config
# if so, skips the path setup
path_configured_in_rc() {
  local rc="$1"
  [ -f "$rc" ] || return 1

  if grep -Fq "${install_dir}" "$rc"; then
    return 0
  fi

  # Default install dir is often configured with $HOME instead of an absolute path
  if [ "${install_dir}" = "${config_home}/.local/bin" ] \
    && grep -Eq '\$HOME/\.local/bin|\$\{HOME\}/\.local/bin' "$rc"; then
    return 0
  fi

  return 1
}

# Pick the shell config file to update
case "${config_shell}" in
  */zsh) rc_file="${config_home}/.zshrc" ;;
  */bash)
    # macOS Terminal.app launches bash as a login shell, which reads
    # .bash_profile (not .bashrc). Prefer an existing .bash_profile there;
    # otherwise create it so the PATH change actually takes effect.
    if [ "$(uname -s)" = "Darwin" ]; then
      if [ -f "${config_home}/.bash_profile" ] || [ ! -f "${config_home}/.bashrc" ]; then
        rc_file="${config_home}/.bash_profile"
      else
        rc_file="${config_home}/.bashrc"
      fi
    else
      rc_file="${config_home}/.bashrc"
    fi
    ;;
  *)
    if [ -f "${config_home}/.zshrc" ]; then
      rc_file="${config_home}/.zshrc"
    elif [ "$(uname -s)" = "Darwin" ] && [ -f "${config_home}/.bash_profile" ]; then
      rc_file="${config_home}/.bash_profile"
    elif [ -f "${config_home}/.bashrc" ]; then
      rc_file="${config_home}/.bashrc"
    else
      rc_file="${config_home}/.profile"
    fi
    ;;
esac

path_line="export PATH=\"${install_dir}:\$PATH\""
if path_configured_in_rc "${rc_file}"; then
  echo "${install_dir} is already configured in ${rc_file}"
  echo "Restart your shell if needed, then run: outpost ls"
  exit 0
fi

# Add the install directory to the shell config
{
  echo ""
  echo "# outpost"
  echo "${path_line}"
} >> "${rc_file}"
echo "Added ${install_dir} to ${rc_file}"
echo "Restart your shell, then run: outpost ls"
