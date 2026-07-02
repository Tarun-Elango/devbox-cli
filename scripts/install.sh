#!/usr/bin/env bash
set -euo pipefail

# --- Config ---
repo="Tarun-Elango/devbox-cli"

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
# Fetch the latest release binary for this OS and CPU
url="https://github.com/${repo}/releases/download/latest/devbox-${os}-${arch}"
tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT

echo "Downloading devbox-${os}-${arch}..."
curl -fsSL "$url" -o "$tmp"
chmod +x "$tmp"

# --- Install ---
# Copy the binary into place
mkdir -p "$install_dir"  # create the directory if it doesn't exist
install -m 755 "$tmp" "${install_dir}/devbox"  # copy the binary into the directory
echo "Installed devbox to ${install_dir}/devbox" # print the path of the binary

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
  echo "Restart your shell if needed, then run: devbox ls"
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
  echo "Restart your shell if needed, then run: devbox ls"
  exit 0
fi

# ask the user if they want to add it to their shell config
if [ -t 0 ]; then
  printf 'Add %s to PATH in %s? [y/N] ' "${install_dir}" "${rc_file}"
  read -r reply
  case "${reply}" in
    [yY] | [yY][eE][sS]) ;;
    *)
      echo "Skipped PATH setup. Add this line to ${rc_file} manually:"
      echo "  ${path_line}"
      exit 0
      ;;
  esac
else
  echo "Add this line to ${rc_file} to use devbox:"
  echo "  ${path_line}"
  exit 0
fi

# add the install directory to the shell config
{
  echo ""
  echo "# devbox"
  echo "${path_line}"
} >> "${rc_file}"
echo "Added ${install_dir} to ${rc_file}"
echo "Restart your shell, then run: devbox ls"
