import DocOutline from './doc-outline'
import DocPage from './doc-page'

const leastPrivilegePolicy = `{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "OutpostEC2",
      "Effect": "Allow",
      "Action": [
        "ec2:RunInstances",
        "ec2:TerminateInstances",
        "ec2:StopInstances",
        "ec2:StartInstances",
        "ec2:RebootInstances",
        "ec2:DescribeInstances",
        "ec2:CreateTags",
        "ec2:ModifyInstanceAttribute",
        "ec2:ModifyVolume",
        "ec2:DescribeVolumes",
        "ec2:CreateVolume",
        "ec2:DescribeImages",
        "ec2:CreateImage",
        "ec2:CreateSnapshot",
        "ec2:DeregisterImage",
        "ec2:DeleteSnapshot",
        "ec2:DescribeSubnets",
        "ec2:DescribeVpcs",
        "ec2:DescribeSecurityGroups",
        "ec2:CreateSecurityGroup",
        "ec2:AuthorizeSecurityGroupIngress"
      ],
      "Resource": "*"
    },
    {
      "Sid": "OutpostSTS",
      "Effect": "Allow",
      "Action": ["sts:GetCallerIdentity"],
      "Resource": "*"
    }
  ]
}`

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
        <h2 id="aws-credentials">Create AWS credentials</h2>
        <p className="note">
          Create a dedicated IAM user for Outpost, then create an access key for that user.
          You will enter the access key and secret access key when you run <code>outpost setup</code>.
        </p>
        <ol>
          <li>
            In the AWS IAM console, go to <strong>Users</strong> → <strong>Create user</strong>.
            Name it something recognizable, such as <code>outpost</code>.
          </li>
          <li>
            Give the user permissions by attaching <code>AmazonEC2FullAccess</code>,{' '}
            <code>AWSBudgetsActionsWithAWSResourceControlAccess</code>, and{' '}
            <code>AmazonSSMReadOnlyAccess</code>, then create the user.
            For more limited EC2 permissions, use the custom policy below in place of{' '}
            <code>AmazonEC2FullAccess</code>.
          </li>
          <li>
            Open the new user, select <strong>Security credentials</strong>, then create an
            access key. Choose <strong>Local code</strong> when AWS asks how the key will be used.
          </li>
          <li>
            Save both values: the access key ID and secret access key. AWS shows the secret
            access key only once.
          </li>
        </ol>

        <details className="drawer" id="least-privilege">
          <summary>
            Use a custom EC2 policy instead of <code>AmazonEC2FullAccess</code>
          </summary>
          <p className="note">
            This policy grants only the EC2 and STS permissions Outpost needs to manage boxes,
            snapshots, imports, security groups, and <code>outpost health</code>. You must still
            attach <code>AWSBudgetsActionsWithAWSResourceControlAccess</code> for budget controls
            and <code>AmazonSSMReadOnlyAccess</code> to resolve latest AMI IDs.
          </p>
          <ol>
            <li>
              In IAM, go to <strong>Policies</strong> → <strong>Create policy</strong> →{' '}
              <strong>JSON</strong>.
            </li>
            <li>
              Paste the policy below, name it (for example, <code>OutpostEC2</code>), and create it.
            </li>
            <li>
              Attach <code>OutpostEC2</code>,{' '}
              <code>AWSBudgetsActionsWithAWSResourceControlAccess</code>, and{' '}
              <code>AmazonSSMReadOnlyAccess</code> to the IAM user you created.
            </li>
          </ol>
          <pre>
            <code>{leastPrivilegePolicy}</code>
          </pre>
        </details>
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
