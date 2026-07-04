import { Link } from 'react-router-dom'
import HowToPage from './how-to-page'

export default function AiSandboxHowTo() {
  return (
    <HowToPage title="AI sandbox box">
      <div className="card">
        <p>
          Spin up a remote box with Codex, Cursor Agent, Claude Code, or another AI coding
          tool pre-installed. Experiment on AWS instead of your laptop — isolated from
          local projects, easy to stop when you are done, and simple to snapshot if you
          want to keep the setup.
        </p>
      </div>

      <div className="card">
        <h2>1. Pick a template</h2>
        <p>List built-in templates (includes several AI agents):</p>
        <pre>
          <code>devbox template</code>
        </pre>
        <p>Common choices:</p>
        <ul>
          <li>
            <code>codex22</code> — OpenAI Codex CLI (installs Node.js 22 if needed)
          </li>
          <li>
            <code>cursor</code> — Cursor Agent CLI
          </li>
          <li>
            <code>claude-code</code> — Claude Code CLI
          </li>
          <li>
            <code>opencode</code> — OpenCode agent CLI
          </li>
        </ul>
        <p className="note">
          Search by name: <code>devbox template search codex</code>. Combine templates
          when creating a box, e.g. <code>go codex22</code> for Go plus Codex.
        </p>
      </div>

      <div className="card">
        <h2>2. Create the box</h2>
        <pre>
          <code>devbox create sandbox --template codex22</code>
        </pre>
        <p>
          devbox launches an EC2 instance, adds <code>devbox-sandbox</code> to{' '}
          <code>~/.ssh/config</code>, and runs the template install scripts on first boot.
        </p>
        <p className="note">
          New to devbox? See <Link to="/docs/install">Installation</Link> and{' '}
          <Link to="/docs/setup">Setup</Link> first.
        </p>
      </div>

      <div className="card">
        <h2>3. Wait for setup, then connect</h2>
        <pre>
          <code>{`devbox status sandbox
devbox ssh sandbox`}</code>
        </pre>
        <p>
          <code>devbox ssh</code> waits until the instance is ready and templates have
          finished. After the first successful login, you can also use plain SSH or VS
          Code — see{' '}
          <Link to="/how-tos/vscode-ssh">VS Code &amp; SSH without the CLI</Link>.
        </p>
        <p className="note">
          Template scripts run at boot; verify your tools are installed (
          <code>codex --version</code>, <code>agent --version</code>, etc.). Older
          startup scripts may not fully install on every AMI — reinstall manually if
          needed.
        </p>
      </div>

      <div className="card">
        <h2>4. Work remotely</h2>
        <p>Typical workflow on the box:</p>
        <pre>
          <code>{`mkdir -p ~/experiments && cd ~/experiments
git clone git@github.com:you/some-repo.git
codex`}</code>
        </pre>
        <ul>
          <li>
            Use <code>devbox git-sync sandbox</code> to forward your local GitHub SSH key
            for <code>git clone</code> / <code>git push</code> without copying keys to
            the box.
          </li>
          <li>
            Copy local files with <code>devbox cp</code> or <code>devbox sync</code> — see{' '}
            <Link to="/how-tos/transfer">Transfer data and files</Link>.
          </li>
          <li>
            Open the project in VS Code via Remote-SSH to <code>devbox-sandbox</code> for
            a full IDE on the remote machine.
          </li>
        </ul>
      </div>

      <div className="card">
        <h2>5. Save money when idle</h2>
        <p>Stop the box when you are not using it:</p>
        <pre>
          <code>devbox stop sandbox</code>
        </pre>
        <p>Or auto-stop after inactivity:</p>
        <pre>
          <code>devbox idle-stop set sandbox 60</code>
        </pre>
        <p>
          Snapshot a box you want to reuse later:{' '}
          <code>devbox snapshot create sandbox my-snapshot</code>, then restore with{' '}
          <code>devbox create newbox --template codex22 --from my-snapshot</code>.
        </p>
      </div>
    </HowToPage>
  )
}
