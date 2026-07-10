import { Link } from 'react-router-dom'
import DocOutline from '../docs/doc-outline'
import HowToPage from './how-to-page'

const sections = [
  { id: 'plain-ssh', label: 'Connect with plain SSH' },
  { id: 'vscode', label: 'Connect with VS Code' },
  { id: 'outpost-ssh', label: 'outpost ssh vs regular ssh' },
]

export default function VscodeSshHowTo() {
  return (
    <HowToPage title="VS Code & SSH without the outpost CLI">
      <DocOutline items={sections} />

      <div className="card">
        <p>
          You can manage boxes with outpost cli and still connect with plain <code>ssh</code> or
          VS Code Remote-SSH. Every time you create, start, or rename a box, outpost cli writes
          a host entry to <code>~/.ssh/config</code> so standard SSH tools work without
          calling <code>outpost ssh</code>. After <code>outpost create mybox</code>, look for a
          block named <code>outpost-mybox</code>:
        </p>

        <p>outpost cli adds a block to your <code>~/.ssh/config</code> file:</p>
        <pre>
          <code>{`Host outpost-mybox
    HostName 203.0.113.42
    User ec2-user
    IdentityFile ~/.ssh/id_ed25519
    StrictHostKeyChecking accept-new`}</code>
        </pre>
        <p className="note">
          The <code>HostName</code> is updated automatically when a box gets a new public
          IP (for example after <code>outpost start</code>). Renaming a box rewrites the
          host alias (<code>outpost-old</code> → <code>outpost-new</code>).
        </p>
      </div>

      <div className="card">
        <h2 id="plain-ssh">Connect with plain SSH</h2>

        <p>Make sure the box is running and ready to connect to. Then run:</p>
        <pre>
          <code>ssh outpost-mybox</code>
        </pre>
        <p>
          Use any SSH option your client supports — port forwarding, remote commands,{' '}
          <code>scp</code>, <code>rsync</code>, and so on:
        </p>
        <pre>
          <code>{`scp ./app.go outpost-mybox:/home/ec2-user/
ssh outpost-mybox -L 8080:localhost:8080`}</code>
        </pre>
        <p className="note">
          For copy and sync helpers built into outpost, see{' '}
          <Link to="/how-tos/transfer">Transfer data and files</Link>.
        </p>
      </div>

      <div className="card">
        <h2 id="vscode">Connect with VS Code</h2>
        <ol>
          <li>
            Install the{' '}
            <a
              href="https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-ssh"
              rel="noreferrer"
              target="_blank"
            >
              Remote - SSH
            </a>{' '}
            extension.
          </li>
          <li>
            Open the Command Palette (<kbd>CMD+Shift+P</kbd> or <kbd>Ctrl+Shift+P</kbd>) →{' '}
            <strong>Remote-SSH: Connect to Host…</strong>
          </li>
          <li>
            Pick <code>outpost-mybox</code> from the list (VS Code reads{' '}
            <code>~/.ssh/config</code>).
          </li>
          <li>
            Open a folder on the remote machine, e.g.{' '}
            <code>/home/ec2-user</code>.
          </li>
        </ol>
        <p className="note">
          You still use outpost for lifecycle tasks — create, start, stop, delete, resize —
          but day-to-day editing and terminal work can stay in VS Code over SSH.
        </p>
      </div>

      <div className="card">
        <h2 id="outpost-ssh">
          <code>outpost ssh</code> vs regular <code>ssh</code>
        </h2>
        <p>
          Both open the same interactive session on the box. Port forwarding, remote
          commands, and the rest of your SSH client&apos;s options work either way —{' '}
          <code>outpost ssh mybox -- -L 8080:localhost:8080</code> and{' '}
          <code>ssh outpost-mybox -L 8080:localhost:8080</code> do the same thing.
        </p>
        <p>
          Use <code>ssh outpost-mybox</code> (or VS Code Remote-SSH) for day-to-day work
          once the box is running. Reach for <code>outpost ssh</code> when you want
          outpost to handle a bit more:
        </p>
        <ul>
          <li>
            First connection while the box is still provisioning —{' '}
            <code>outpost ssh</code> polls until the instance is ready.
          </li>
          <li>
            Passing a non-default key: <code>outpost ssh mybox -i path/to/key</code>
          </li>
          <li>
            GitHub agent forwarding via <code>outpost git-sync</code> — see{' '}
            <Link to="/how-tos/github-sync">Sync GitHub account</Link>
          </li>
        </ul>
      </div>
    </HowToPage>
  )
}
