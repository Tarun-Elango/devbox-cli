import { Link } from 'react-router-dom'
import DocOutline from '../docs/doc-outline'
import HowToPage from './how-to-page'

const sections = [
  { id: 'how-to-use', label: 'How to use git-sync' },
  { id: 'how-it-works', label: 'How it works' },
]

export default function GithubSyncHowTo() {
  return (
    <HowToPage title="Sync GitHub account">
      <DocOutline items={sections} />

      <div className="card">
        <p>
          Use your local GitHub SSH key on a outpost for <code>git clone</code>,{' '}
          <code>git push</code>, and other Git operations — without copying the key
          onto the box. <code>outpost git-sync</code> sets up SSH agent forwarding so
          the box uses the key already on your laptop.
        </p>
      </div>

      <div className="card">
        <h2 id="how-to-use">How to use git-sync</h2>

        <p>
          First, set up GitHub SSH login on your laptop: generate an SSH key (default path is <code>~/.ssh/id_ed25519</code>), add the
          public key under{' '}
          <a href="https://github.com/settings/keys" rel="noreferrer" target="_blank">
            GitHub → Settings → SSH and GPG keys
          </a>
          , and confirm <code>ssh -T git@github.com</code> succeeds locally — then run:
        </p>
        <pre>
          <code>outpost git-sync mybox</code>
        </pre>
        <p>
          Replace <code>mybox</code> with the box name or ID. The command is a
          toggle — run it again on the same box to turn GitHub SSH access off.
        </p>

        <p>When enabling, outpost:</p>
        <ol>
          <li>
            Adds your key to <code>ssh-agent</code> (you may be prompted for the
            key passphrase).
          </li>
          <li>
            Sets <code>ForwardAgent yes</code> on the box&apos;s SSH config block.
          </li>
        </ol>

        <p>
          Open a new SSH session so the box can use your local key for git — an
          existing session does not pick up agent forwarding automatically:
        </p>
        <pre>
          <code>{`ssh outpost-mybox
# or
outpost ssh mybox -- -A`}</code>
        </pre>

        <p>On the box, verify GitHub sees your account:</p>
        <pre>
          <code>ssh -T git@github.com</code>
        </pre>
        <p className="note">
          You should see a greeting like &quot;Hi username! You&apos;ve successfully
          authenticated…&quot; Then <code>git clone git@github.com:org/repo.git</code>{' '}
          and <code>git push</code> use your local key as usual.
        </p>

        <p>
          Git identity is separate from SSH auth. On the box, set your commit author
          once if you have not already:
        </p>
        <pre>
          <code>{`git config --global user.name "Your Name"
git config --global user.email "you@example.com"`}</code>
        </pre>

        <p>When disabling, outpost removes the key from <code>ssh-agent</code> and
          drops <code>ForwardAgent</code> from the SSH config block for that box.</p>
      </div>

      <div className="card">
        <h2 id="how-it-works">How it works</h2>

        <p>
          GitHub SSH auth normally requires a private key on the machine running{' '}
          <code>git</code>. Copying that key onto a remote box is risky. Agent
          forwarding is the safer pattern: your laptop keeps the key, and SSH
          forwards signing requests to your local <code>ssh-agent</code> when the
          box talks to <code>git@github.com</code>.
        </p>

        <p>
          <code>outpost git-sync</code> wires up both sides of that flow on your
          machine:
        </p>
        <ul>
          <li>
            <strong>ssh-agent</strong> — loads <code>~/.ssh/id_ed25519</code> so
            SSH can offer it for forwarded connections.
          </li>
          <li>
            <strong>ForwardAgent yes</strong> — added to the{' '}
            <code>Host outpost-mybox</code> block in <code>~/.ssh/config</code> so
            connections to the box allow agent forwarding (same as passing{' '}
            <code>-A</code> to <code>ssh</code>).
          </li>
        </ul>

        <p>
          The command checks whether <em>both</em> are already in place for that
          box. If yes, it turns them off; if either is missing, it turns them on.
          That keeps &quot;synced&quot; state easy to reason about — one command to
          enable, the same command to disable.
        </p>

        <p className="note">
          Agent forwarding only applies to new SSH sessions after you reconnect.
          For day-to-day editing over VS Code, see{' '}
          <Link to="/how-tos/vscode-ssh">VS Code &amp; SSH without the outpost CLI</Link>.
          Command reference: <Link to="/docs/connect#git-sync">Git sync</Link> on
          the Connect page.
        </p>
      </div>
    </HowToPage>
  )
}
