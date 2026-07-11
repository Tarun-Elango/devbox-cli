package localDb

import "fmt"

type defaultTemplate struct {
	ID          string
	Name        string
	Description string
	Script      string
	OSFamily    string
	CreatedAt   string
}

type defaultTemplateDefinition struct {
	name        string
	description string
}

// These are the only built-in startup templates. Each definition is available
// for Amazon Linux 2023, Ubuntu, and Debian.
var defaultTemplateDefinitions = []defaultTemplateDefinition{
	{"claude", "Claude Code CLI"},
	{"pi", "Pi coding agent"},
	{"opencode", "OpenCode AI agent CLI"},
	{"go", "Go toolchain"},
	{"python3", "Python 3"},
	{"pip", "Python pip"},
	{"git", "Git version control"},
	{"node22", "Node.js 22"},
	{"npm", "npm via Node.js 22"},
	{"pnpm", "pnpm via Node.js 22"},
	{"bun", "Bun JavaScript runtime"},
	{"docker", "Docker engine"},
}

var defaultTemplates = append(
	append(
		defaultTemplatesForOS("amazon-linux", "ec2-user", "0001", 1),
		defaultTemplatesForOS("ubuntu", "ubuntu", "0002", 2)...,
	),
	defaultTemplatesForOS("debian", "admin", "0003", 3)...,
)

func defaultTemplatesForOS(osFamily, user, idPrefix string, createdMinute int) []defaultTemplate {
	templates := make([]defaultTemplate, 0, len(defaultTemplateDefinitions))
	for i, definition := range defaultTemplateDefinitions {
		templates = append(templates, defaultTemplate{
			ID:          fmt.Sprintf("00000000-0000-0000-%s-%012d", idPrefix, i+1),
			Name:        definition.name,
			Description: fmt.Sprintf("%s (%s)", definition.description, osFamily),
			Script:      defaultTemplateScript(definition.name, osFamily, user),
			OSFamily:    osFamily,
			CreatedAt:   fmt.Sprintf("1970-01-01 00:%02d:%02d", createdMinute, i+1),
		})
	}
	return templates
}

func defaultTemplateScript(name, osFamily, user string) string {
	if osFamily == "amazon-linux" {
		return amazonLinuxTemplateScript(name, user)
	}
	return debianTemplateScript(name, user)
}

func amazonLinuxTemplateScript(name, user string) string {
	dnf := func(command, packages string) string {
		return fmt.Sprintf("command -v %s >/dev/null 2>&1 || dnf install -y %s\n", command, packages)
	}
	node := dnf("node", "nodejs22 nodejs22-npm")
	userNPMInstall := func(command, packageName string) string {
		return node + fmt.Sprintf(
			"if ! runuser -u %s -- bash -lc 'command -v %s >/dev/null 2>&1'; then\n  runuser -u %s -- bash -lc 'npm install -g %s'\nfi\n",
			user, command, user, packageName,
		)
	}

	switch name {
	case "claude":
		return `if ! command -v claude >/dev/null 2>&1; then
  tee /etc/yum.repos.d/claude-code.repo >/dev/null <<'EOF'
[claude-code]
name=Claude Code
baseurl=https://downloads.claude.ai/claude-code/rpm/stable
enabled=1
gpgcheck=1
gpgkey=https://downloads.claude.ai/keys/claude-code.asc
EOF
  dnf install -y claude-code
fi
`
	case "pi":
		return userNPMInstall("pi", "@earendil-works/pi-coding-agent")
	case "opencode":
		return fmt.Sprintf("if ! runuser -u %s -- bash -lc 'command -v opencode >/dev/null 2>&1'; then\n  runuser -u %s -- bash -lc 'curl -fsSL https://opencode.ai/install | bash'\nfi\n", user, user)
	case "go":
		return dnf("go", "golang")
	case "python3":
		return dnf("python3", "python3")
	case "pip":
		return dnf("pip3", "python3-pip")
	case "git":
		return dnf("git", "git")
	case "node22", "npm":
		return node
	case "pnpm":
		return userNPMInstall("pnpm", "pnpm")
	case "bun":
		return `if ! command -v bun >/dev/null 2>&1; then
  export BUN_INSTALL=/usr/local/bun
  curl -fsSL https://bun.sh/install | bash
  ln -sf /usr/local/bun/bin/bun /usr/local/bin/bun
fi
`
	case "docker":
		return fmt.Sprintf(`if ! command -v docker >/dev/null 2>&1; then
  dnf install -y docker
  systemctl enable --now docker
fi
getent group docker >/dev/null || groupadd docker
usermod -aG docker %s 2>/dev/null || true
`, user)
	}
	return ""
}

func debianTemplateScript(name, user string) string {
	apt := func(command, packages string) string {
		return fmt.Sprintf("export DEBIAN_FRONTEND=noninteractive\ncommand -v %s >/dev/null 2>&1 || (apt-get update -qq && apt-get install -y %s)\n", command, packages)
	}
	node := ensureNode22()
	userNPMInstall := func(command, packageName string) string {
		return node + fmt.Sprintf(
			"if ! runuser -u %s -- bash -lc 'command -v %s >/dev/null 2>&1'; then\n  runuser -u %s -- bash -lc 'npm install -g %s'\nfi\n",
			user, command, user, packageName,
		)
	}

	switch name {
	case "claude":
		return fmt.Sprintf("if ! runuser -u %s -- bash -lc 'command -v claude >/dev/null 2>&1'; then\n  runuser -u %s -- bash -lc 'curl -fsSL https://claude.ai/install.sh | bash'\nfi\n", user, user)
	case "pi":
		return userNPMInstall("pi", "@earendil-works/pi-coding-agent")
	case "opencode":
		return fmt.Sprintf("if ! runuser -u %s -- bash -lc 'command -v opencode >/dev/null 2>&1'; then\n  runuser -u %s -- bash -lc 'curl -fsSL https://opencode.ai/install | bash'\nfi\n", user, user)
	case "go":
		return apt("go", "golang-go")
	case "python3":
		return apt("python3", "python3")
	case "pip":
		return apt("pip3", "python3-pip")
	case "git":
		return apt("git", "git")
	case "node22", "npm":
		return node
	case "pnpm":
		return userNPMInstall("pnpm", "pnpm")
	case "bun":
		return `if ! command -v bun >/dev/null 2>&1; then
  export BUN_INSTALL=/usr/local/bun
  curl -fsSL https://bun.sh/install | bash
  ln -sf /usr/local/bun/bin/bun /usr/local/bin/bun
fi
`
	case "docker":
		return fmt.Sprintf(`export DEBIAN_FRONTEND=noninteractive
if ! command -v docker >/dev/null 2>&1; then
  apt-get update -qq && apt-get install -y docker.io
  systemctl enable --now docker
fi
getent group docker >/dev/null || groupadd docker
usermod -aG docker %s 2>/dev/null || true
`, user)
	}
	return ""
}

// ensureNode22 installs Node.js 22 and npm from NodeSource because distro
// repositories frequently provide an older Node version.
func ensureNode22() string {
	return `export DEBIAN_FRONTEND=noninteractive
if ! command -v node >/dev/null 2>&1 || ! node -e 'process.exit(Number(process.versions.node.split(".")[0])>=22?0:1)' 2>/dev/null; then
  apt-get update -qq
  apt-get install -y ca-certificates curl gnupg
  mkdir -p /etc/apt/keyrings
  curl -fsSL https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key | gpg --dearmor -o /etc/apt/keyrings/nodesource.gpg
  echo "deb [signed-by=/etc/apt/keyrings/nodesource.gpg] https://deb.nodesource.com/node_22.x nodistro main" > /etc/apt/sources.list.d/nodesource.list
  apt-get update -qq
  apt-get install -y nodejs
fi
`
}
