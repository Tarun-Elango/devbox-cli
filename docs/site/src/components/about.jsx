import { Link } from 'react-router-dom'

export default function AboutPage() {
    return (
      <>
        <div className="page-title">
          <h1>Outpost</h1>
          <a href="https://github.com/Tarun-Elango" className="byline">
            by Tarun-Elango
          </a>
        </div>
        <p className="tagline">
          Manage remote dev boxes from the CLI — provision, connect, sync, and destroy
          them using your own AWS account (BYOK).
        </p>

        <div className="card card-purpose">
          <h2>Why outpost?</h2>
          {/* <figure className="purpose-diagram">
            <div className="purpose-flow">
              <div className="purpose-step">
                <span>Your terminal</span>
                <strong>You</strong>
              </div>
              <span className="purpose-arrow" aria-hidden="true">→</span>
              <div className="purpose-step purpose-step-cli">
                <span>Remote control</span>
                <strong>Outpost CLI</strong>
              </div>
              <span className="purpose-arrow" aria-hidden="true">→</span>
              <div className="purpose-step">
                <span>Your AWS account</span>
                <strong>Dev box</strong>
              </div>
            </div>
            <figcaption>
              Create, connect to, and remove cloud development machines from your
              terminal. Your AWS credentials stay local.
            </figcaption>
          </figure> */}
          <ul>
            <li>
              <strong>Dedicated dev machine on the cloud</strong> — your own EC2
              instance, separate from production and your daily driver
            </li>
            <li>
              <strong>Smaller blast radius</strong> — experiments, tooling, and
              dependencies stay off your main machine
            </li>
            <li>
              <strong>Fast lifecycle</strong> — create, use, and tear down boxes
              in minutes
            </li>
            <li>
              <strong>Reproducible setups</strong> — spin up consistent environments
              from templates
            </li>
            <li>
              <strong>Commands that simplify daily work</strong> —{' '}
              <code>ssh</code>, <code>sync</code>, <code>idle-stop</code>,{' '}
              <code>git-sync</code>, <code>import</code>, and more
            </li>
            <li>
              <strong>Secure by default</strong> — AWS credentials and config stored
              locally on your machine, and the code is open source and available on GitHub.
            </li>
          </ul>
        </div>

        <div className="card">
          <h2>Requirements</h2>
          <ul>
            <li>macOS or Linux</li>
            <li>Your own AWS account (BYOK)</li>
            <li>
              On <code>PATH</code> you might need: <code>ssh</code> for SSH commands, <code>scp</code> for
              copy, <code>rsync</code> for folder sync, and <code>ssh-agent</code> for
              GitHub sync between your machine and a box
            </li>
          </ul>
          <p className="note">
            outpost cli runs on your machine and uses your AWS account — no shared cloud,
            no hosted credentials. Run <code>outpost setup</code> to save keys locally in{' '}
            <code>~/.outpost/</code>.
          </p>
        </div>

        <div className="card">
          <h2>Quick install</h2>
          <p className="note">
            Every push to <code>main</code> publishes binaries to the{' '}
            <a href="https://github.com/Tarun-Elango/Outpost/releases/tag/latest">
              latest release
            </a>, run the following command:

          </p>
          <pre>
            <code>{`curl -fsSL https://raw.githubusercontent.com/Tarun-Elango/Outpost/latest/scripts/install.sh | bash`}</code>
          </pre>
          <p className="note">
            Verify with the command <code>outpost ls</code>.
          </p>
          <p className="note">
            If that worked, you&apos;re done — skip the sections below. They&apos;re
            optional alternatives for pinning a version or installing system-wide.
          </p>

          <h3>
            Pin a specific version — To install a particular release instead of{' '}
            <code>latest</code>, set <code>RELEASE_TAG</code> on <code>bash</code> (not
            on <code>curl</code>):
          </h3>
          <pre>
            <code>{`curl -fsSL https://raw.githubusercontent.com/Tarun-Elango/Outpost/latest/scripts/install.sh | RELEASE_TAG=v0.7.0 bash`}</code>
          </pre>

          <h3>
            Install system-wide — To install to <code>/usr/local/bin</code> (requires{' '}
            <code>sudo</code>, no shell config changes):
          </h3>
          <pre>
            <code>{`INSTALL_DIR=/usr/local/bin curl -fsSL https://raw.githubusercontent.com/Tarun-Elango/Outpost/latest/scripts/install.sh | sudo bash`}</code>
          </pre>

         
        </div>

        <div className="card">
          <h2>Common commands</h2>
          <ul>
            <li>
              <code>outpost setup</code> — configure AWS credentials{' '}
              <Link to="/docs/setup">(see how to get AWS credentials)</Link>
            </li>
            <li>
              <code>outpost create {'<name>'}</code> — create a box
            </li>
            <li>
              <code>outpost ls</code> — list boxes
            </li>
            <li>
              <code>outpost ssh {'<name>'}</code> — connect via SSH
            </li>
            <li>
              <code>outpost import </code> — import existing instance from aws.
            </li>
          </ul>
          <p className="note">
            <Link to="/docs/commands">See all commands</Link>
          </p>
        </div>

        <div className="card">
          <h2>Links</h2>
          <ul>
            <li>
              <a href="https://github.com/Tarun-Elango/Outpost">GitHub repository</a>
            </li>
            <li>
              <a href="https://github.com/Tarun-Elango/Outpost/releases">Releases</a>
            </li>
            <li>
              <a href="https://github.com/Tarun-Elango/outpost/blob/main/readme.md">
                Full README
              </a>
            </li>
          </ul>
        </div>
      </>
    )
  }
