import { Link } from 'react-router-dom'
import DocOutline from './doc-outline'
import DocPage from './doc-page'

const sections = [
  { id: 'ssh-via-outpost', label: 'SSH via outpost' },
  { id: 'ssh-config', label: 'SSH via plain ssh' },
  { id: 'copy-sync', label: 'Copy & sync' },
  { id: 'remote-commands', label: 'Remote commands' },
  { id: 'git-sync', label: 'Git sync' },
]

export default function ConnectDoc() {
  return (
    <DocPage title="Connect & transfer">
      <DocOutline items={sections} />

      <div className="card">
        <h2 id="ssh-via-outpost">Option 1: SSH via outpost</h2>
        <pre>
          <code>outpost ssh [-i key] {'<id-or-name>'} [-- {'<ssh-option>'}...]</code>
        </pre>
        <p className="note">
          Open an interactive SSH session. On first connect, outpost waits until the
          instance is ready before handing off to ssh. When you run{' '}
          <code>outpost create</code>, your local public key (
          <code>~/.ssh/id_ed25519.pub</code>) is added to the box so the matching
          private key can log in.
        </p>
        <dl className="cmd-variant">
          <dt>Simple SSH</dt>
          <dd>
            <code>outpost ssh {'<id-or-name>'}</code>
          </dd>
          <dd className="example">
            Example: <code>outpost ssh mybox</code> (uses <code>~/.ssh/id_ed25519</code>)
          </dd>
        </dl>
        <dl className="cmd-variant">
          <dt>Custom private key</dt>
          <dd>
            <code>outpost ssh [-i key] {'<id-or-name>'}</code>
          </dd>
          <dd className="example">
            Example: <code>outpost ssh -i path/to/key mybox</code>
          </dd>
        </dl>
        <dl className="cmd-variant">
          <dt>Native ssh flags</dt>
          <dd>
            <code>outpost ssh {'<id-or-name>'} [-- {'<ssh-option>'}...]</code>
          </dd>
          <dd className="example">
            Example:{' '}
            <code>outpost ssh mybox -- -L 8080:localhost:8080</code> (port forwarding,{' '}
            <code>-A</code>, <code>-v</code>, etc.)
          </dd>
        </dl>

      </div>

      <div className="card">
        <h2 id="ssh-config">Option 2: SSH via your computer&apos;s SSH config</h2>
        <p>
          Every time you create, start, or rename a box, outpost writes a host entry to{' '}
          <code>~/.ssh/config</code> on your machine. This lets you connect with plain{' '}
          <code>ssh</code> (or any SSH-based tool) without going through the{' '}
          <code>outpost</code> CLI.
        </p>
        
  
        <dl className="cmd-variant">
          <dt>Plain SSH</dt>
          <dd>
            <code>ssh outpost-mybox</code>
          </dd>
          <dd className="example">
            Works with <code>scp</code>, <code>rsync</code>, port forwarding, and any other
            SSH option your client supports.
          </dd>
        </dl>
        <dl className="cmd-variant">
          <dt>VS Code Remote-SSH</dt>
          <dd>
            Inside VS Code, open the Command Palette → <strong>Remote-SSH: Connect to Host…</strong>
          </dd>
          <dd className="example">
            <code>outpost-mybox</code> shows up automatically in the host list since VS Code
            reads <code>~/.ssh/config</code>.
          </dd>
        </dl>
        <p className="note">
          For a full walkthrough, see{' '}
          <Link to="/how-tos/vscode-ssh">VS Code &amp; SSH without the CLI</Link>.
        </p>
      </div>

      <div className="card">
        <h2 id="copy-sync">Copy &amp; sync</h2>
        <p className="note">
          Copy or sync files between your machine and a box. Remote paths use{' '}
          <code>{'<id-or-name>'}:/path</code> syntax. 
        </p>
        <dl className="cmd-variant">
          <dt>Copy a file</dt>
          <dd>
            <code>outpost cp [-i key] {'<source>'} {'<dest>'}</code>
          </dd>
          <dd className="example">
            Example: <code>outpost cp ./main.go mybox:/home/ec2-user/app/</code> (upload),{' '}
            <code>outpost cp mybox:/home/ec2-user/app/main.go ./</code> (download)
          </dd>
        </dl>
        <dl className="cmd-variant">
          <dt>Sync a directory</dt>
          <dd>
            <code>outpost sync [-i key] [--delete] {'<source>'} {'<dest>'}</code>
          </dd>
          <dd className="example">
            Uses <code>rsync</code> over SSH. Path syntax matches <code>cp</code>: one side is
            local, the other is <code>{'<id-or-name>'}:/path</code>. Only the{' '}
            <strong>destination</strong> is modified — new or changed files are copied from
            source; the source is never changed.
          </dd>
          <dd className="example">
            Upload: <code>outpost sync ./src mybox:/home/ec2-user/app/</code> (updates the box).
            Download: <code>outpost sync mybox:/home/ec2-user/app/ ./src</code> (updates your
            machine).
          </dd>
          <dd className="example">
            <code>--delete</code> also removes files on the destination that are not in the
            source, so dest matches source exactly.
          </dd>
        </dl>
      </div>

      <div className="card">
        <h2 id="remote-commands">Remote commands</h2>
        <p className="note">
          Run one-off commands or forward ports without opening an interactive SSH
          session.
        </p>
        <dl className="cmd-variant">
          <dt>Run a command</dt>
          <dd>
            <code>outpost exec [-i key] {'<id-or-name>'} -- {'<command>'}</code>
          </dd>
          <dd className="example">
            Example: <code>outpost exec mybox -- uname -a</code>
          </dd>
        </dl>
        <dl className="cmd-variant">
          <dt>Shell snippet</dt>
          <dd>
            <code>outpost exec -s {'<id-or-name>'} -- &quot;{'<snippet>'}&quot;</code>
          </dd>
          <dd className="example">
            Example:{' '}
            <code>outpost exec -s mybox -- &quot;cd app &amp;&amp; make&quot;</code> (pipes,{' '}
            <code>&amp;&amp;</code>, <code>cd</code>, etc.)
          </dd>
        </dl>
        <dl className="cmd-variant">
          <dt>Interactive / sudo</dt>
          <dd>
            <code>outpost exec -t {'<id-or-name>'} -- {'<command>'}</code>
          </dd>
          <dd className="example">
            Example: <code>outpost exec -t mybox -- sudo apt update</code> (allocates a
            TTY)
          </dd>
        </dl>
        <dl className="cmd-variant">
          <dt>Port forward</dt>
          <dd>
            <code>outpost forward {'<id-or-name>'} {'<port>'}</code>
          </dd>
          <dd className="example">
            <code>{'<port>'}</code> is the port on the <strong>box</strong> (e.g. a server
            listening on 8080). outpost picks a free port on your machine and tunnels{' '}
            <code>localhost:your-local-port</code> on your laptop to that box port. It prints
            the URL (e.g. <code>http://localhost:54321</code>); press Ctrl+C to stop.
          </dd>
          <dd className="example">
            Example: <code>outpost forward mybox 8080</code> — open the printed{' '}
            <code>localhost:your-local-port</code> URL in your browser to reach the
            box&apos;s port 8080.
          </dd>
        </dl>
      </div>

      <div className="card">
        <h2 id="git-sync">Git sync</h2>
        <p className="note">
          Use your local GitHub SSH key on a box (for <code>git push</code>,{' '}
          <code>git clone</code>, etc.) without copying it there: adds the key to{' '}
          <code>ssh-agent</code> and enables agent forwarding (<code>-A</code>) in the
          box&apos;s SSH config. Run again to undo both.
        </p>
        <ul>
          <li>
            <code>outpost git-sync {'<id-or-name>'}</code> — toggle GitHub SSH agent
            forwarding for a box
          </li>
        </ul>
        <p className="note">
          Git identity is separate from SSH auth: on the box you may still need to set{' '}
          <code>git config --global user.name "Your Name"</code> and{' '}
          <code>git config --global user.email "you@example.com"</code> before commits show the right author.
        </p>
      </div>
    </DocPage>
  )
}
