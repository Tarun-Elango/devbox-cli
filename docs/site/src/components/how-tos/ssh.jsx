import { Link } from 'react-router-dom'
import DocOutline from '../docs/doc-outline'
import HowToPage from './how-to-page'

const sections = [
  { id: 'how-to-use', label: 'How to use SSH' },
  { id: 'how-it-works', label: 'How it works' },
]

export default function SshHowTo() {
  return (
    <HowToPage title="SSH">
      <DocOutline items={sections} />

      <div className="card">
        <p>
          <code>outpost ssh</code> opens an interactive shell on a running box. It
          waits until the instance is reachable and finished provisioning, then
          hands off to your system&apos;s <code>ssh</code> client — same session
          you would get with plain SSH, with a bit of setup handled for you.
        </p>
      </div>

      <div className="card">
        <h2 id="how-to-use">How to use SSH</h2>

        <p>
          Make sure the box is running (<code>outpost start mybox</code> if needed),
          then connect with the box name or ID:
        </p>
        <pre>
          <code>outpost ssh mybox</code>
        </pre>
        <p>
          By default outpost uses <code>~/.ssh/id_ed25519</code> as the private
          key. Pass <code>-i</code> to use a different one:
        </p>
        <pre>
          <code>outpost ssh -i path/to/key mybox</code>
        </pre>

        <p>
          Anything after <code>--</code> is passed straight to <code>ssh</code> as
          native options — verbose mode, agent forwarding, port forwarding, and so
          on. Use <code>outpost exec</code> instead when you want to run a single
          remote command without opening a shell.
        </p>
        <pre>
          <code>{`# Port forward local 8080 to the box
outpost ssh mybox -- -L 8080:localhost:8080

# Forward your SSH agent (useful with git-sync)
outpost ssh mybox -- -A

# Verbose connection debug
outpost ssh mybox -- -v

# Set remote TERM (e.g. Ghostty)
outpost ssh mybox -- -o SetEnv=TERM=xterm-256color`}</code>
        </pre>

        <p>
          After the first successful <code>outpost ssh</code>, you can also connect
          with plain SSH using the host alias outpost wrote to your config:
        </p>
        <pre>
          <code>ssh outpost-mybox</code>
        </pre>
        <p className="note">
          Plain <code>ssh outpost-mybox</code> skips outpost&apos;s readiness
          polling — use <code>outpost ssh</code> right after{' '}
          <code>outpost create</code> or <code>outpost start</code> while the box
          is still coming up. For VS Code, GitHub keys, and file transfer, see{' '}
          <Link to="/how-tos/vscode-ssh">VS Code &amp; SSH</Link>,{' '}
          <Link to="/how-tos/github-sync">Sync GitHub account</Link>, and{' '}
          <Link to="/how-tos/transfer">Transfer data and files</Link>.
        </p>
      </div>

      <div className="card">
        <h2 id="how-it-works">How it works</h2>

        <p>
          SSH login needs two pieces: your laptop holds the <strong>private</strong>{' '}
          key, and the box holds the matching <strong>public</strong> key in{' '}
          <code>~/.ssh/authorized_keys</code>. outpost wires both sides up when you
          create a box.
        </p>

        <p>When you run <code>outpost create mybox</code>:</p>
        <ol>
          <li>
            outpost reads your local public key from{' '}
            <code>~/.ssh/id_ed25519.pub</code> (or prompts you to generate one).
          </li>
          <li>
            That key is embedded in the box&apos;s startup script and added to{' '}
            <code>authorized_keys</code> for the <code>ec2-user</code> account.
          </li>
          <li>
            A <code>Host outpost-mybox</code> block is written to{' '}
            <code>~/.ssh/config</code> on your machine with the box IP, user, and
            key path — so <code>ssh outpost-mybox</code> works without extra flags.
          </li>
        </ol>

        <p>When you run <code>outpost ssh mybox</code>:</p>
        <ol>
          <li>
            outpost checks that the box is running and has a public IP.
          </li>
          <li>
            It polls until SSH accepts connections and startup scripts finish
            (templates, packages, and so on).
          </li>
          <li>
            It execs your system <code>ssh</code> with{' '}
            <code>ec2-user@&lt;ip&gt;</code>, your private key, and any options
            you passed after <code>--</code>.
          </li>
        </ol>

        <p>
          The private key never leaves your laptop. The box only stores the public
          half — enough to verify that you hold the matching private key, not
          enough to impersonate you elsewhere.
        </p>

        <p className="note">
          Command reference: <Link to="/docs/connect#ssh-via-outpost">SSH via outpost</Link>{' '}
          on the Connect page.
        </p>
      </div>
    </HowToPage>
  )
}
