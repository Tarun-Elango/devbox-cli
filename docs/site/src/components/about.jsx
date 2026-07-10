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
          <div className="purpose-diagram">
            <svg
              width="672"
              height="272"
              viewBox="4 4 672 272"
              xmlns="http://www.w3.org/2000/svg"
              role="img"
            >
              <title>outpost architecture</title>
              <desc>
                Your laptop runs the outpost CLI; config and credentials stay local.
                Commands like setup, create, start, ssh, sync, forward, stop, and
                delete manage computers running in the cloud.
              </desc>
              <defs>
                <marker
                  id="arrow"
                  viewBox="0 0 10 10"
                  refX="8"
                  refY="5"
                  markerWidth="7"
                  markerHeight="7"
                  orient="auto-start-reverse"
                >
                  <path
                    d="M2 1L8 5L2 9"
                    fill="none"
                    stroke="#000080"
                    strokeWidth="1.5"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                  />
                </marker>
              </defs>

              <rect
                x="4"
                y="4"
                width="672"
                height="272"
                fill="none"
                stroke="#000080"
                strokeWidth="3"
              />
              <rect
                x="9"
                y="9"
                width="662"
                height="262"
                fill="none"
                stroke="#000080"
                strokeWidth="1"
              />

              <rect
                x="40"
                y="60"
                width="220"
                height="140"
                fill="#FFFFCC"
                stroke="#000080"
                strokeWidth="2"
              />
              <rect
                x="44"
                y="64"
                width="212"
                height="132"
                fill="none"
                stroke="#000080"
                strokeWidth="0.75"
              />
              <text
                x="150"
                y="86"
                textAnchor="middle"
                fontFamily="Times New Roman, Times, serif"
                fontSize="16"
                fontWeight="bold"
                fill="#000080"
              >
                Your laptop
              </text>

              <rect
                x="64"
                y="106"
                width="172"
                height="70"
                fill="#000000"
                stroke="#003300"
                strokeWidth="1"
              />
              <text
                x="150"
                y="132"
                textAnchor="middle"
                fontFamily="Courier New, monospace"
                fontSize="14"
                fontWeight="bold"
                fill="#33FF33"
              >
                outpost
              </text>
              <text
                x="150"
                y="150"
                textAnchor="middle"
                fontFamily="Courier New, monospace"
                fontSize="14"
                fontWeight="bold"
                fill="#33FF33"
              >
                CLI
              </text>
              <text
                x="150"
                y="168"
                textAnchor="middle"
                fontFamily="Courier New, monospace"
                fontSize="11"
                fill="#33FF33"
              >
                $_
              </text>

              <line
                x1="260"
                y1="130"
                x2="418"
                y2="130"
                stroke="#000080"
                strokeWidth="2"
                markerStart="url(#arrow)"
                markerEnd="url(#arrow)"
              />
              <text
                x="339"
                y="112"
                textAnchor="middle"
                fontFamily="Times New Roman, Times, serif"
                fontSize="12"
                fill="#000080"
              >
                create · start · ssh · sync
              </text>
              <text
                x="339"
                y="152"
                textAnchor="middle"
                fontFamily="Times New Roman, Times, serif"
                fontSize="12"
                fill="#000080"
              >
                forward · stop · delete
              </text>

              <rect
                x="420"
                y="30"
                width="220"
                height="218"
                fill="#FFFFCC"
                stroke="#000080"
                strokeWidth="2"
              />
              <rect
                x="424"
                y="34"
                width="212"
                height="210"
                fill="none"
                stroke="#000080"
                strokeWidth="0.75"
              />
              <text
                x="530"
                y="56"
                textAnchor="middle"
                fontFamily="Times New Roman, Times, serif"
                fontSize="16"
                fontWeight="bold"
                fill="#000080"
              >
                Cloud
              </text>

              <rect
                x="444"
                y="76"
                width="172"
                height="60"
                fill="#FFFFFF"
                stroke="#000080"
                strokeWidth="1.5"
              />
              <text
                x="530"
                y="100"
                textAnchor="middle"
                fontFamily="Times New Roman, Times, serif"
                fontSize="14"
                fontWeight="bold"
                fill="#000080"
              >
                Computer
              </text>
              <text
                x="530"
                y="120"
                textAnchor="middle"
                fontFamily="Times New Roman, Times, serif"
                fontSize="12"
                fill="#000080"
              >
                dev box 1
              </text>

              <rect
                x="444"
                y="150"
                width="172"
                height="60"
                fill="#FFFFFF"
                stroke="#000080"
                strokeWidth="1.5"
              />
              <text
                x="530"
                y="174"
                textAnchor="middle"
                fontFamily="Times New Roman, Times, serif"
                fontSize="14"
                fontWeight="bold"
                fill="#000080"
              >
                Computer 
              </text>
              <text
                x="530"
                y="194"
                textAnchor="middle"
                fontFamily="Times New Roman, Times, serif"
                fontSize="12"
                fill="#000080"
              >
                dev box 2
              </text>
              <text
                x="530"
                y="224"
                textAnchor="middle"
                fontFamily="Times New Roman, Times, serif"
                fontSize="11"
                fontStyle="italic"
                fill="#000080"
              >
                spin up n boxes as needed
              </text>

              <text
                x="150"
                y="190"
                textAnchor="middle"
                fontFamily="Times New Roman, Times, serif"
                fontSize="11"
                fill="#000080"
              >
                config &amp; credentials stay local
              </text>
            </svg>
          </div>
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
              <code>git-sync</code>, and more
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
            <a href="https://github.com/Tarun-Elango/outpost/releases/tag/latest">
              latest release
            </a>, run the following command:

          </p>
          <pre>
            <code>{`curl -fsSL https://raw.githubusercontent.com/Tarun-Elango/outpost/latest/scripts/install.sh | bash`}</code>
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
            <code>{`curl -fsSL https://raw.githubusercontent.com/Tarun-Elango/outpost/latest/scripts/install.sh | RELEASE_TAG=v0.7.0 bash`}</code>
          </pre>

          <h3>
            Install system-wide — To install to <code>/usr/local/bin</code> (requires{' '}
            <code>sudo</code>, no shell config changes):
          </h3>
          <pre>
            <code>{`INSTALL_DIR=/usr/local/bin curl -fsSL https://raw.githubusercontent.com/Tarun-Elango/outpost/latest/scripts/install.sh | sudo bash`}</code>
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
          </ul>
          <p className="note">
            <Link to="/docs/commands">See all commands</Link>
          </p>
        </div>

        <div className="card">
          <h2>Links</h2>
          <ul>
            <li>
              <a href="https://github.com/Tarun-Elango/outpost">GitHub repository</a>
            </li>
            <li>
              <a href="https://github.com/Tarun-Elango/outpost/releases">Releases</a>
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
