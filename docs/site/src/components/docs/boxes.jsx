import DocOutline from './doc-outline'
import DocPage from './doc-page'

const sections = [
  { id: 'create-box', label: 'Create a box' },
  { id: 'power-lifecycle', label: 'Power & lifecycle' },
  { id: 'idle-stop', label: 'Idle stop' },
]

export default function BoxesDoc() {
  return (
    <DocPage title="Managing boxes">
      <DocOutline items={sections} />

      <p className="note">
        A box is an EC2 instance running a supported <strong>Linux</strong> distro —
        Amazon Linux 2023, Ubuntu 24.04, or Debian 12. You choose the OS when creating
        a box. Templates and SSH login users are scoped to that OS (
        <code>ec2-user</code>, <code>ubuntu</code>, or <code>admin</code>).
      </p>

      <div className="card">
        <h2 id="create-box">Create a box</h2>
        <pre>
          <code>outpost create mybox</code>
        </pre>
        <p className="note">
          Launches an interactive wizard. You will be asked to choose a Linux OS,
          an instance type (↑/↓, Enter to confirm), and a root disk size in GB (press
          Enter to accept the default). Press Ctrl+C at any prompt to cancel.
        </p>
        <dl className="cmd-variant">
          <dt>Restore from a snapshot</dt>
          <dd>
            <code>outpost create {'<name>'} [--from {'<amiId|name>'}]</code>
          </dd>
          <dd className="example">
            Example: <code>outpost create mybox --from my-snapshot</code>
          </dd>
        </dl>
        <dl className="cmd-variant">
          <dt>Preload templates ( 1 upto n templates)</dt>
          <dd>
            <code>
              outpost create {'<name>'} [--template {'<templateName>'}...] 
            </code>
          </dd>
          <dd className="example">
            Example:{' '}
            <code>outpost create mybox --template node go opencode</code>
          </dd>
          <dd className="note">
            You can <code>outpost ssh</code> quickly, but template startup may
            still be running — wait a few minutes before checking installed tools.
          </dd>
        </dl>
        <dl className="cmd-variant">
          <dt>Templates and snapshot together</dt>
          <dd>
            <code>
              outpost create {'<name>'} [--template {'<templateName>'}...] [--from {'<amiId|name>'}]
            </code>
          </dd>
          <dd className="example">
            Example:{' '}
            <code>
              outpost create mybox --template java claude-code --from my-snapshot
            </code>
          </dd>
        </dl>

        <h3>List &amp; inspect</h3>
        <ul>
          <li>
            <code>outpost ls</code> — list all boxes
          </li>
          <li>
            <code>outpost status mybox</code> — show details for a box
          </li>
          <li>
            <code>outpost rename mybox new-name</code> — rename a box
          </li>
        </ul>
      </div>

      <div className="card">
        <h2 id="power-lifecycle">Power &amp; lifecycle</h2>
        <ul>
          <li>
            <code>outpost stop mybox</code> — stop a running box
          </li>
          <li>
            <code>outpost start mybox</code> — start a stopped box
          </li>
          <li>
            <code>outpost restart mybox</code> — reboot a running box (
            <code>reboot</code> is an alias)
          </li>
          <li>
            <code>outpost delete mybox</code> — delete a box
          </li>
        </ul>

        <h3>Resize</h3>
        <pre>
          <code>
            outpost stop mybox{'\n'}outpost resize mybox
          </code>
        </pre>
        <p className="note">
          The box must be stopped first. <code>upgrade</code> is an alias for{' '}
          <code>resize</code>. You will be asked whether to change the instance type
          and root disk size — answer <code>n</code> to keep either one the same. Disk
          size can only be increased, not decreased.
        </p>
      </div>

      <div className="card">
        <h2 id="idle-stop">Idle stop</h2>
        <p className="note">
          Automatically stop a box after a period with no activity — useful when you
          forget to shut down and want to save cost.
        </p>
        <pre>
          <code>outpost idle-stop set mybox 30</code>
        </pre>
        <p className="note">
          Stops <code>mybox</code> after 30 minutes of inactivity. The box must be
          running and SSH-ready.
        </p>
        <ul>
          <li>
            <code>outpost idle-stop show mybox</code> — show the current timeout
          </li>
          <li>
            <code>outpost idle-stop update mybox 60</code> — change the timeout
          </li>
          <li>
            <code>outpost idle-stop delete mybox</code> — remove idle-stop from a box
          </li>
        </ul>
      </div>
    </DocPage>
  )
}
