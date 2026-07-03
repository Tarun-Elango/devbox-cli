package localDb

import (
	"database/sql"
	"fmt"
)

type defaultTemplate struct {
	ID          string
	Name        string
	Description string
	Script      string
	CreatedAt   string
}

// Built-in startup templates for Amazon Linux 2023 (dnf). Each ID is seeded at most
// once; default_template_seeds records offerings so user deletes are not restored on Open().
// When a seeded row still exists, description and startup_script are synced from this file.
var defaultTemplates = []defaultTemplate{
	// Languages
	{
		ID:          "00000000-0000-0000-0001-000000000001",
		Name:        "python3",
		Description: "Python 3 and pip (Amazon Linux 2023)",
		CreatedAt:   "1970-01-01 00:00:01",
		Script: `command -v python3 >/dev/null 2>&1 || dnf install -y python3 python3-pip
`,
	},
	{
		ID:          "00000000-0000-0000-0001-000000000002",
		Name:        "java21",
		Description: "OpenJDK 21 (Amazon Corretto) on Amazon Linux 2023",
		CreatedAt:   "1970-01-01 00:00:02",
		Script: `command -v javac >/dev/null 2>&1 || dnf install -y java-21-amazon-corretto-devel
`,
	},
	{
		ID:          "00000000-0000-0000-0001-000000000003",
		Name:        "cpp",
		Description: "GCC C/C++ toolchain on Amazon Linux 2023",
		CreatedAt:   "1970-01-01 00:00:03",
		Script: `command -v g++ >/dev/null 2>&1 || dnf install -y gcc gcc-c++ make
`,
	},
	{
		ID:          "00000000-0000-0000-0001-000000000004",
		Name:        "go",
		Description: "Go toolchain from Amazon Linux 2023 repos",
		CreatedAt:   "1970-01-01 00:00:04",
		Script: `command -v go >/dev/null 2>&1 || dnf install -y golang
`,
	},
	{
		ID:          "00000000-0000-0000-0001-000000000005",
		Name:        "rust",
		Description: "Rust via rustup for ec2-user",
		CreatedAt:   "1970-01-01 00:00:05",
		Script: `runuser -u ec2-user -- bash -lc 'command -v rustc >/dev/null 2>&1 || curl --proto "=https" --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y'
`,
	},
	{
		ID:          "00000000-0000-0000-0001-000000000006",
		Name:        "ruby",
		Description: "Ruby from Amazon Linux 2023 repos",
		CreatedAt:   "1970-01-01 00:00:06",
		Script: `command -v ruby >/dev/null 2>&1 || dnf install -y ruby
`,
	},
	{
		ID:          "00000000-0000-0000-0001-000000000007",
		Name:        "node22",
		Description: "Node.js 22 and npm from Amazon Linux 2023 repos",
		CreatedAt:   "1970-01-01 00:00:07",
		Script: `command -v node >/dev/null 2>&1 || dnf install -y nodejs22 nodejs22-npm
`,
	},
	{
		ID:          "00000000-0000-0000-0001-000000000008",
		Name:        "dotnet8",
		Description: ".NET 8 SDK on Amazon Linux 2023",
		CreatedAt:   "1970-01-01 00:00:08",
		Script: `command -v dotnet >/dev/null 2>&1 || dnf install -y dotnet-sdk-8.0
`,
	},

	// Package managers
	{
		ID:          "00000000-0000-0000-0001-000000000010",
		Name:        "pip",
		Description: "Python pip (Amazon Linux 2023)",
		CreatedAt:   "1970-01-01 00:00:10",
		Script: `command -v pip3 >/dev/null 2>&1 || dnf install -y python3 python3-pip
`,
	},
	{
		ID:          "00000000-0000-0000-0001-000000000011",
		Name:        "npm22",
		Description: "npm via Node.js 22 on Amazon Linux 2023",
		CreatedAt:   "1970-01-01 00:00:11",
		Script: `command -v npm >/dev/null 2>&1 || dnf install -y nodejs22 nodejs22-npm
`,
	},
	{
		ID:          "00000000-0000-0000-0001-000000000012",
		Name:        "bun",
		Description: "Bun JavaScript runtime (system-wide install)",
		CreatedAt:   "1970-01-01 00:00:12",
		Script: `if ! command -v bun >/dev/null 2>&1; then
  export BUN_INSTALL=/usr/local/bun
  curl -fsSL https://bun.sh/install | bash
  ln -sf /usr/local/bun/bin/bun /usr/local/bin/bun
fi
`,
	},
	{
		ID:          "00000000-0000-0000-0001-000000000013",
		Name:        "pnpm22",
		Description: "pnpm via npm for ec2-user (installs Node.js 22 if needed)",
		CreatedAt:   "1970-01-01 00:00:13",
		Script: `command -v node >/dev/null 2>&1 || dnf install -y nodejs22 nodejs22-npm
if ! runuser -u ec2-user -- bash -lc 'command -v pnpm >/dev/null 2>&1'; then
  runuser -u ec2-user -- bash -lc 'npm install -g pnpm'
  grep -q '\.local/bin' /home/ec2-user/.bashrc 2>/dev/null || echo 'export PATH="$HOME/.local/bin:$PATH"' >> /home/ec2-user/.bashrc
fi
`,
	},
	{
		ID:          "00000000-0000-0000-0001-000000000014",
		Name:        "yarn22",
		Description: "Yarn via npm for ec2-user (installs Node.js 22 if needed)",
		CreatedAt:   "1970-01-01 00:00:14",
		Script: `command -v node >/dev/null 2>&1 || dnf install -y nodejs22 nodejs22-npm
if ! runuser -u ec2-user -- bash -lc 'command -v yarn >/dev/null 2>&1'; then
  runuser -u ec2-user -- bash -lc 'npm install -g yarn'
  grep -q '\.local/bin' /home/ec2-user/.bashrc 2>/dev/null || echo 'export PATH="$HOME/.local/bin:$PATH"' >> /home/ec2-user/.bashrc
fi
`,
	},

	// Tools
	{
		ID:          "00000000-0000-0000-0001-000000000015",
		Name:        "git",
		Description: "Git version control on Amazon Linux 2023",
		CreatedAt:   "1970-01-01 00:00:15",
		Script: `command -v git >/dev/null 2>&1 || dnf install -y git
`,
	},
	{
		ID:          "00000000-0000-0000-0001-000000000016",
		Name:        "docker",
		Description: "Docker engine on Amazon Linux 2023 (ec2-user in docker group)",
		CreatedAt:   "1970-01-01 00:00:16",
		Script: `if ! command -v docker >/dev/null 2>&1; then
  dnf install -y docker
  systemctl enable --now docker
fi
getent group docker >/dev/null || groupadd docker
usermod -aG docker ec2-user 2>/dev/null || true
`,
	},
	{
		ID:          "00000000-0000-0000-0001-000000000017",
		Name:        "uv",
		Description: "uv Python package manager for ec2-user",
		CreatedAt:   "1970-01-01 00:00:17",
		Script: `if ! runuser -u ec2-user -- bash -lc 'command -v uv >/dev/null 2>&1'; then
  runuser -u ec2-user -- bash -lc 'curl -LsSf https://astral.sh/uv/install.sh | sh'
  grep -q '\.local/bin' /home/ec2-user/.bashrc 2>/dev/null || echo 'export PATH="$HOME/.local/bin:$PATH"' >> /home/ec2-user/.bashrc
fi
`,
	},
	{
		ID:          "00000000-0000-0000-0001-000000000018",
		Name:        "maven",
		Description: "Apache Maven on Amazon Linux 2023",
		CreatedAt:   "1970-01-01 00:00:18",
		Script: `command -v mvn >/dev/null 2>&1 || dnf install -y maven
`,
	},
	{
		ID:          "00000000-0000-0000-0001-000000000019",
		Name:        "gradle",
		Description: "Gradle build tool on Amazon Linux 2023",
		CreatedAt:   "1970-01-01 00:00:19",
		Script: `command -v gradle >/dev/null 2>&1 || dnf install -y gradle
`,
	},

	// AI coding agents
	{
		ID:          "00000000-0000-0000-0001-000000000020",
		Name:        "claude-code",
		Description: "Claude Code CLI (Anthropic dnf repo for Amazon Linux 2023)",
		CreatedAt:   "1970-01-01 00:00:20",
		Script: `if ! command -v claude >/dev/null 2>&1; then
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
`,
	},
	{
		ID:          "00000000-0000-0000-0001-000000000021",
		Name:        "cursor",
		Description: "Cursor Agent CLI for ec2-user",
		CreatedAt:   "1970-01-01 00:00:21",
		Script: `if ! runuser -u ec2-user -- bash -lc 'command -v agent >/dev/null 2>&1'; then
  runuser -u ec2-user -- bash -lc 'curl https://cursor.com/install -fsS | bash'
  grep -q '\.local/bin' /home/ec2-user/.bashrc 2>/dev/null || echo 'export PATH="$HOME/.local/bin:$PATH"' >> /home/ec2-user/.bashrc
fi
`,
	},
	{
		ID:          "00000000-0000-0000-0001-000000000022",
		Name:        "codex22",
		Description: "OpenAI Codex CLI via npm for ec2-user (installs Node.js 22 if needed)",
		CreatedAt:   "1970-01-01 00:00:22",
		Script: `command -v node >/dev/null 2>&1 || dnf install -y nodejs22 nodejs22-npm
if ! runuser -u ec2-user -- bash -lc 'command -v codex >/dev/null 2>&1'; then
  runuser -u ec2-user -- bash -lc 'npm install -g @openai/codex'
  grep -q '\.local/bin' /home/ec2-user/.bashrc 2>/dev/null || echo 'export PATH="$HOME/.local/bin:$PATH"' >> /home/ec2-user/.bashrc
fi
`,
	},
	{
		ID:          "00000000-0000-0000-0001-000000000023",
		Name:        "pi",
		Description: "Pi coding agent for ec2-user (pi.dev)",
		CreatedAt:   "1970-01-01 00:00:23",
		Script: `if ! runuser -u ec2-user -- bash -lc 'command -v pi >/dev/null 2>&1'; then
  runuser -u ec2-user -- bash -lc 'curl -fsSL https://pi.dev/install.sh | sh'
  grep -q '\.local/bin' /home/ec2-user/.bashrc 2>/dev/null || echo 'export PATH="$HOME/.local/bin:$PATH"' >> /home/ec2-user/.bashrc
fi
`,
	},
	{
		ID:          "00000000-0000-0000-0001-000000000024",
		Name:        "opencode",
		Description: "OpenCode AI agent CLI (system-wide install)",
		CreatedAt:   "1970-01-01 00:00:24",
		Script: `if ! command -v opencode >/dev/null 2>&1; then
  OPENCODE_INSTALL_DIR=/usr/local/bin curl -fsSL https://opencode.ai/install | bash
fi
`,
	},
}

func (db *DB) seedDefaultTemplates() error {
	if err := db.backfillDefaultTemplateSeeds(); err != nil { // backfill the default template seeds if they don't exist
		return err
	}

	for _, tmpl := range defaultTemplates {
		seeded, err := db.defaultTemplateAlreadySeeded(tmpl.ID) // check if the template is already seeded
		if err != nil {
			return fmt.Errorf("check seed state for %s: %w", tmpl.Name, err)
		}
		if seeded {
			// sync the template with the built-in template, incase content has changed
			if err := db.syncDefaultTemplate(tmpl); err != nil {
				return fmt.Errorf("sync template %s: %w", tmpl.Name, err)
			}
			continue
		}

		existing, err := db.GetTemplateByNameAndUserID(tmpl.Name, LocalUserID)
		if err != nil && err != sql.ErrNoRows {
			return fmt.Errorf("check existing template %s: %w", tmpl.Name, err)
		}
		if err == nil {
			if existing.ID != tmpl.ID {
				// User-owned template already uses this name; skip offering the built-in.
				if err := db.recordDefaultTemplateSeed(tmpl.ID); err != nil {
					return fmt.Errorf("record seed for %s: %w", tmpl.Name, err)
				}
				continue
			}
			// Built-in row exists but seed metadata is missing.
			if err := db.recordDefaultTemplateSeed(tmpl.ID); err != nil {
				return fmt.Errorf("record seed for %s: %w", tmpl.Name, err)
			}
			if err := db.syncDefaultTemplate(tmpl); err != nil {
				return fmt.Errorf("sync template %s: %w", tmpl.Name, err)
			}
			continue
		}

		tx, err := db.conn.Begin()
		if err != nil {
			return fmt.Errorf("seed template %s: %w", tmpl.Name, err)
		}

		_, err = tx.Exec(`
			INSERT INTO templates (id, user_id, name, description, startup_script, created_at)
			VALUES (?, ?, ?, ?, ?, ?)`,
			tmpl.ID,
			LocalUserID,
			tmpl.Name,
			nullIfEmpty(tmpl.Description),
			nullIfEmpty(tmpl.Script),
			tmpl.CreatedAt,
		)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("seed template %s: %w", tmpl.Name, err)
		}

		_, err = tx.Exec(
			`INSERT INTO default_template_seeds (template_id) VALUES (?)`,
			tmpl.ID,
		)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record seed for %s: %w", tmpl.Name, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("seed template %s: %w", tmpl.Name, err)
		}
	}
	return nil
}

// syncDefaultTemplate updates description and startup_script when the built-in row
// still exists. User renames and user-deleted templates are left unchanged.
func (db *DB) syncDefaultTemplate(tmpl defaultTemplate) error {
	record, err := db.GetTemplateByID(tmpl.ID)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	if record.UserID != LocalUserID {
		return nil
	}

	wantDescription := tmpl.Description
	wantScript := tmpl.Script
	if nullStringValue(record.Description) == wantDescription &&
		nullStringValue(record.StartupScript) == wantScript {
		return nil
	}

	_, err = db.conn.Exec(`
		UPDATE templates
		SET description = ?, startup_script = ?
		WHERE id = ? AND user_id = ?`,
		nullIfEmpty(wantDescription),
		nullIfEmpty(wantScript),
		tmpl.ID,
		LocalUserID,
	)
	if err != nil {
		return fmt.Errorf("update template content: %w", err)
	}
	return nil
}

func (db *DB) recordDefaultTemplateSeed(templateID string) error {
	_, err := db.conn.Exec(
		`INSERT OR IGNORE INTO default_template_seeds (template_id) VALUES (?)`,
		templateID,
	)
	if err != nil {
		return err
	}
	return nil
}

func (db *DB) defaultTemplateAlreadySeeded(templateID string) (bool, error) {
	var exists int
	err := db.conn.QueryRow(
		`SELECT 1 FROM default_template_seeds WHERE template_id = ? LIMIT 1`,
		templateID,
	).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// backfillDefaultTemplateSeeds upgrades DBs seeded before default_template_seeds existed.
// If any built-in template row remains, treat the whole initial batch as already offered
// (including ones the user deleted before upgrading).
func (db *DB) backfillDefaultTemplateSeeds() error {
	var seedCount int
	if err := db.conn.QueryRow(`SELECT COUNT(*) FROM default_template_seeds`).Scan(&seedCount); err != nil {
		return fmt.Errorf("count default template seeds: %w", err)
	}
	if seedCount > 0 {
		return nil
	}

	var existing int
	if err := db.conn.QueryRow(`
		SELECT COUNT(*) FROM templates WHERE id LIKE '00000000-0000-0000-0001-%'`,
	).Scan(&existing); err != nil {
		return fmt.Errorf("count existing default templates: %w", err)
	}
	if existing == 0 {
		return nil
	}

	for _, tmpl := range defaultTemplates {
		_, err := db.conn.Exec(
			`INSERT OR IGNORE INTO default_template_seeds (template_id) VALUES (?)`,
			tmpl.ID,
		)
		if err != nil {
			return fmt.Errorf("backfill seed for %s: %w", tmpl.Name, err)
		}
	}
	return nil
}
