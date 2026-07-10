import DocOutline from './doc-outline'
import DocPage from './doc-page'

const sections = [
  { id: 'aws-credentials', label: 'AWS credentials' },
  { id: 'wizard', label: 'Setup on outpost CLI' },
  { id: 'local-config', label: 'Local config' },
  { id: 'related-commands', label: 'Related commands' },
]

export default function SetupDoc() {
  return (
    <DocPage title="Setup">
      <DocOutline items={sections} />

      <div className="card">
        <h2 id="aws-credentials">AWS setup — get access key and secret access key</h2>
        <p className="note">
          Create a dedicated IAM user before running the setup wizard.
        </p>
        <ol>
          <li>
            IAM console → <strong>Users</strong> → <strong>Create user</strong> (e.g.{' '}
            <code>outpost</code>)
          </li>
          <li>
            Attach <code>AmazonEC2FullAccess</code> and <code>AWSBudgetsActionsWithAWSResourceControlAccess</code> directly and create the user
          </li>
          <li>
            Open the user → <strong>Security credentials</strong> → create an access key
            (choose <strong>Local code</strong>)
          </li>
          <li>
            Copy the access key and secret access key (secret access key shown only once)
          </li>
        </ol>
      </div>

      <div className="card">
        <h2 id="wizard">Setup on outpost CLI</h2>
        <p className="note">
          Run once after install to save AWS credentials and region locally.
        </p>
        <pre>
          <code>outpost setup</code>
        </pre>
        <p className="note">
          Enter the access key, secret, and preferred AWS region when prompted. 
        </p>
      </div>

      <div className="card">
        <h2 id="local-config">Local config details</h2>
        <p className="note">
          Credentials and tokens live in <code>~/.outpost/config.json</code> (mode 0600).
        </p>
        <ul>
          <li>Do not sync <code>~/.outpost</code> via dotfiles, iCloud, Dropbox, or Git</li>
          <li>Use a dedicated IAM user for AWS keys</li>
          <li>
            Run <code>outpost health</code> to verify config, credentials, region, and
            database
          </li>
          <li>You can run <code>outpost setup</code> again to update the config and credentials.</li>
        </ul>
      </div>

      <div className="card">
        <h2 id="related-commands">Related commands</h2>
        <ul>
          <li>
            <code>outpost health</code> — check config, credentials, region, and database
          </li>
          <li>
            <code>outpost clear-creds</code> — remove saved AWS credentials
          </li>
          <li>
            <code>outpost update</code> — check for and install a newer CLI release
          </li>
          <li>
            <code>outpost version</code> — show the installed CLI version
          </li>
        </ul>
      </div>
    </DocPage>
  )
}
