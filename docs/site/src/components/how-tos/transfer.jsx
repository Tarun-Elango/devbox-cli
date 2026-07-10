import { Link } from 'react-router-dom'
import DocOutline from '../docs/doc-outline'
import HowToPage from './how-to-page'

const sections = [
  { id: 'copy-file', label: 'Copy a file with cp' },
  { id: 'sync-directory', label: 'Sync a directory with sync' },
  { id: 'how-it-works', label: 'What happens under the hood' },
]

export default function TransferHowTo() {
  return (
    <HowToPage title="Transfer data and files">
      <DocOutline items={sections} />

      <div className="card">
        <p>
          Move files between your laptop and a remote box with <code>outpost cp</code> and{' '}
          <code>outpost sync</code>. Both commands use SSH under the hood — you do not need
          to look up the box IP or edit <code>~/.ssh/config</code> yourself. Remote paths
          use <code>mybox:/path/on/box</code> syntax (box name or ID, then a colon and the
          path).
        </p>
        <p className="note">
          For plain <code>scp</code> or <code>rsync</code> with the <code>outpost-mybox</code>{' '}
          host alias, see{' '}
          <Link to="/how-tos/vscode-ssh">VS Code &amp; SSH without the outpost CLI</Link>.
          Command reference: <Link to="/docs/connect#copy-sync">Copy &amp; sync</Link> on
          the Connect page.
        </p>
      </div>

      <div className="card">
        <h2 id="copy-file">Copy a file with <code>cp</code></h2>

        <p>
          Use <code>outpost cp</code> when you want to copy <strong>one file</strong> in either
          direction — upload from your machine to the box, or download from the box to your
          machine.
        </p>

        <p>Upload a local file to the box:</p>
        <pre>
          <code>outpost cp ./main.go mybox:/home/ec2-user/app/</code>
        </pre>

        <p>Download a file from the box:</p>
        <pre>
          <code>outpost cp mybox:/home/ec2-user/app/main.go ./</code>
        </pre>

        <p>
          Replace <code>mybox</code> with your box name or ID. Use <code>-i path/to/key</code>{' '}
          if your SSH key is not the default <code>~/.ssh/id_ed25519</code>.
        </p>
      </div>

      <div className="card">
        <h2 id="sync-directory">Sync a directory with <code>sync</code></h2>

        <p>
          Use <code>outpost sync</code> when you want to keep a <strong>whole folder</strong>{' '}
          in sync — for example your project directory. It uses <code>rsync</code> over SSH,
          so only changed files are transferred after the first run.
        </p>

        <p>Upload your local project to the box (updates files on the box):</p>
        <pre>
          <code>outpost sync ./project mybox:/home/ec2-user/project</code>
        </pre>

        <p>Download the project from the box to your laptop (updates your local copy):</p>
        <pre>
          <code>outpost sync mybox:/home/ec2-user/project ./project</code>
        </pre>

        <p>
          Add <code>--delete</code> when you want the destination to match the source exactly
          — files on the destination that are not in the source are removed:
        </p>
        <pre>
          <code>outpost sync --delete ./project mybox:/home/ec2-user/project</code>
        </pre>
        <p className="note">
          <code>--delete</code> only affects the <strong>destination</strong>. Your source
          folder is never modified or deleted.
        </p>
      </div>

      <div className="card">
        <h2 id="how-it-works">What happens under the hood</h2>

        <p>In simple terms:</p>
        <ul>
          <li>
            <strong><code>cp</code></strong> — copies a single file from source to
            destination. One side is always local; the other is{' '}
            <code>boxname:/remote/path</code>.
          </li>
          <li>
            <strong><code>sync</code></strong> — copies a directory the same way, but with{' '}
            <code>rsync</code> so repeated runs are fast. Again, one side is local and one
            is <code>boxname:/remote/path</code>.
          </li>
          <li>
            <strong>Only the destination changes.</strong> Whether you upload or download,
            outpost reads from the source and writes to the destination. The source is
            read-only — nothing on the source side is deleted or overwritten by mistake.
          </li>
          <li>
            <strong><code>--delete</code> (sync only)</strong> — after copying new and
            changed files, also removes files on the destination that no longer exist in
            the source, so both trees match.
          </li>
        </ul>

        <p>
          The box must be running and reachable over SSH. If you use a custom key, pass{' '}
          <code>-i</code> to either command the same way you would with <code>outpost ssh</code>.
        </p>
      </div>
    </HowToPage>
  )
}
