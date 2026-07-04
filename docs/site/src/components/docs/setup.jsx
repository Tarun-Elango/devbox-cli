import DocPage from './doc-page'

export default function SetupDoc() {
  return (
    <DocPage title="Setup">
      <div className="card">
        <h2>Interactive wizard</h2>
        <p className="note">
          Run once after install to save AWS credentials and region locally.
        </p>
        <pre>
          <code>devbox setup</code>
        </pre>
        <p className="note">
          Credentials are stored in <code>~/.devbox/config.json</code> (mode 0600).
          Use a dedicated IAM user — see the steps below.
        </p>
      </div>

      <div className="card">
        <h2>AWS setup</h2>
        <ol>
          <li>
            IAM console → <strong>Users</strong> → <strong>Create user</strong> (e.g.{' '}
            <code>devbox-cli</code>)
          </li>
          <li>
            Attach <code>AmazonEC2FullAccess</code> directly and create the user
          </li>
          <li>
            Open the user → <strong>Security credentials</strong> → create an access key
            (choose <strong>Local code</strong>)
          </li>
          <li>Copy the access key ID and secret (secret shown only once)</li>
          <li>
            Save in devbox: <code>devbox setup</code>
          </li>
        </ol>
      </div>

      <div className="card">
        <h2>Local config</h2>
        <p className="note">
          Credentials and tokens live in <code>~/.devbox/config.json</code> (mode 0600).
        </p>
        <ul>
          <li>Do not sync <code>~/.devbox</code> via dotfiles, iCloud, Dropbox, or Git</li>
          <li>Use a dedicated IAM user for AWS keys</li>
          <li>
            Run <code>devbox health</code> to verify config, credentials, region, and
            database
          </li>
        </ul>
      </div>

      <div className="card">
        <h2>Related commands</h2>
        <ul>
          <li>
            <code>devbox health</code> — check config, credentials, region, and database
          </li>
          <li>
            <code>devbox clear-creds</code> — remove saved AWS credentials
          </li>
          <li>
            <code>devbox update</code> — check for and install a newer CLI release
          </li>
          <li>
            <code>devbox version</code> — show the installed CLI version
          </li>
        </ul>
      </div>
    </DocPage>
  )
}
