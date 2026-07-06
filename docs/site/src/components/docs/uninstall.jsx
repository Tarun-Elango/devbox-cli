import DocPage from './doc-page'

export default function UninstallDoc() {
  return (
    <DocPage title="Uninstall">
      <div className="card">
        <h2>Uninstall</h2>
        <pre>
          <code>devbox uninstall</code>
        </pre>
        <p className="note">
          Removes devbox from your machine. You will be asked to confirm before
          anything is deleted.
        </p>
        <p className="note">This command:</p>
        <ul>
          <li>Deletes the <code>devbox</code> binary</li>
          <li>Removes <code>~/.devbox</code> (config, database, caches)</li>
          <li>Removes <code>~/.devbox-backup</code></li>
          <li>Clears devbox PATH entries from your shell config</li>
        </ul>
        <p className="note">
          Restart your shell after uninstalling. If you installed to{' '}
          <code>/usr/local/bin</code>, run with <code>sudo</code>.
        </p>
      </div>
    </DocPage>
  )
}
