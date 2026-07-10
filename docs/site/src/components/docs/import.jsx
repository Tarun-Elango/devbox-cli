import DocOutline from './doc-outline'
import DocPage from './doc-page'

const sections = [
  { id: 'use-import', label: 'Import resources' },
  { id: 'how-it-works', label: 'How it works' },
]

export default function ImportDoc() {
  return (
    <DocPage title="Import existing AWS resources">
      <DocOutline items={sections} />

      <div className="card">
        <h2 id="use-import">Import resources</h2>
        <p className="note">
          Import existing EC2 instances and self-owned AMIs from the AWS region
          configured with <code>outpost setup</code>.
        </p>
        <pre>
          <code>outpost import</code>
        </pre>
        <p>
          Outpost shows each instance and self-owned AMI that is not already tracked
          locally. Answer <code>y</code> to import a resource or press Enter to skip it.
        </p>
        <p>
          For a running instance with a reachable IP address, Outpost asks for the
          existing private SSH key that AWS originally configured for the instance.
          Enter its path, for example:
        </p>
        <pre>
          <code>~/Downloads/my-key.pem</code>
        </pre>
        <p>
          You can also enter the full path, such as{' '}
          <code>/Users/you/Downloads/my-key.pem</code>, or leave it blank to skip SSH
          authorization. The key must be the matching <code>.pem</code> file, and its
          permissions should be restricted, for example with{' '}
          <code>chmod 400 ~/Downloads/my-key.pem</code>.
        </p>
        <p className="note">
          Imported boxes must run Amazon Linux for <code>outpost ssh</code> and{' '}
          <code>outpost idle-stop</code> to work. Other Linux distributions may be
          imported and tracked, but Outpost-specific SSH setup and idle detection are
          not guaranteed to work.
        </p>
      </div>

      <div className="card">
        <h2 id="how-it-works">How it works</h2>
        <p>
          In simple terms, import connects to AWS, looks at the EC2 instances and
          self-owned AMIs in your configured region, and compares them with the
          resources already saved in Outpost&apos;s local database.
        </p>
        <ol>
          <li>
            Outpost finds untracked instances and self-owned AMIs in the configured
            region.
          </li>
          <li>
            It asks you about each resource one at a time, so you choose what to add.
          </li>
          <li>
            Accepted instances and self-owned AMIs are saved locally with an Outpost
            name. Existing AWS resources are not recreated.
          </li>
          <li>
            For a running box, the optional <code>.pem</code> key lets Outpost log in
            once and add your local Outpost public key to the box&apos;s{' '}
            <code>authorized_keys</code>.
          </li>
        </ol>
        <p className="note">
          Import does not copy or move EC2 instances, their disks, or self-owned AMIs.
          It only records the AWS resource locally so you can manage it through Outpost.
        </p>
      </div>
    </DocPage>
  )
}
