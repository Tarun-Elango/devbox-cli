import DocOutline from './doc-outline'
import DocPage from './doc-page'

const sections = [
  { id: 'snapshots', label: 'Snapshots' },
  { id: 'templates', label: 'Templates' },
]

export default function SnapshotsDoc() {
  return (
    <DocPage title="Snapshots & templates">
      <DocOutline items={sections} />

      <div className="card">
        <h2 id="snapshots">Snapshots</h2>
        <p className="note">
          A snapshot is a saved disk image of a box — restore one when creating a new box
          with <code>--from</code>.
        </p>

        <dl className="cmd-variant">
          <dt>List all snapshots</dt>
          <dd>
            <code>devbox snapshot [ls]</code>
          </dd>
          <dd className="example">
            Example: <code>devbox snapshot ls</code>
          </dd>
        </dl>
        <dl className="cmd-variant">
          <dt>Create a snapshot</dt>
          <dd>
            <code>devbox snapshot create {'<id-or-name>'} {'<name>'}</code>
          </dd>
          <dd className="example">
            Example: <code>devbox snapshot create mybox pre-upgrade</code>
          </dd>
        </dl>
        <dl className="cmd-variant">
          <dt>Show snapshot details</dt>
          <dd>
            <code>devbox snapshot ls {'<amiId-or-name>'}</code>
          </dd>
          <dd className="example">
            Example: <code>devbox snapshot ls pre-upgrade</code>
          </dd>
        </dl>
        <dl className="cmd-variant">
          <dt>Delete a snapshot</dt>
          <dd>
            <code>devbox snapshot delete {'<amiId-or-name>'}</code>
          </dd>
          <dd className="example">
            Example: <code>devbox snapshot delete pre-upgrade</code>
          </dd>
        </dl>
        <dl className="cmd-variant">
          <dt>Restore when creating a box</dt>
          <dd>
            <code>devbox create {'<name>'} --from {'<amiId|name>'}</code>
          </dd>
          <dd className="example">
            Example: <code>devbox create mybox --from pre-upgrade</code>
          </dd>
        </dl>
      </div>

      <div className="card">
        <h2 id="templates">Templates</h2>
        <p className="note">
          A template preloads a box with libraries, tools, and other setup when you create
          it.
        </p>

        <dl className="cmd-variant">
          <dt>List available templates</dt>
          <dd>
            <code>devbox template [ls]</code>
          </dd>
          <dd className="example">
            Example: <code>devbox template ls</code>
          </dd>
        </dl>
        <dl className="cmd-variant">
          <dt>Create a template</dt>
          <dd>
            <code>devbox template new {'<templateName>'} [command]</code>
          </dd>
          <dd className="example">
            Example:{' '}
            <code>devbox template new my-stack &quot;npm install -g pnpm&quot;</code>
          </dd>
        </dl>
        <dl className="cmd-variant">
          <dt>Search templates by name</dt>
          <dd>
            <code>devbox template search {'<query>'}</code>
          </dd>
          <dd className="example">
            Example: <code>devbox template search node</code>
          </dd>
        </dl>
        <dl className="cmd-variant">
          <dt>Rename a template</dt>
          <dd>
            <code>
              devbox template rename {'<templateName>'} {'<new-templateName>'}
            </code>
          </dd>
          <dd className="example">
            Example: <code>devbox template rename my-stack my-stack-v2</code>
          </dd>
        </dl>
        <dl className="cmd-variant">
          <dt>Delete a template</dt>
          <dd>
            <code>devbox template delete {'<templateName>'}</code>
          </dd>
          <dd className="example">
            Example: <code>devbox template delete my-stack</code>
          </dd>
        </dl>
        <dl className="cmd-variant">
          <dt>Use templates when creating a box</dt>
          <dd>
            <code>devbox create {'<name>'} [--template {'<templateName>'}...]</code>
          </dd>
          <dd className="example">
            Example: <code>devbox create mybox --template node go</code>
          </dd>
        </dl>
      </div>
    </DocPage>
  )
}
