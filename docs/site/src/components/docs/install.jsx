import DocOutline from './doc-outline'
import DocPage from './doc-page'

const sections = [
  { id: 'quick-install', label: 'Quick install' },
  { id: 'pin-version', label: 'Pin a version' },
  { id: 'shared-install', label: 'Shared machine' },
  { id: 'build-from-source', label: 'Build from source' },
]

export default function InstallDoc() {
  return (
    <DocPage title="Installation">
      <DocOutline items={sections} />

      <div className="card">
        <h2 id="quick-install">Quick install</h2>
        <pre>
          <code>{`curl -fsSL https://raw.githubusercontent.com/Tarun-Elango/outpost/latest/scripts/install.sh | bash`}</code>
        </pre>
        <p className="note">
          Detects your OS and CPU, downloads the matching binary, installs to{' '}
          <code>~/.local/bin</code>, and adds that directory to your shell config if
          needed. Restart your shell, then verify:
        </p>
        <pre>
          <code>outpost version</code>
        </pre>
      </div>

      <div className="card">
        <h2 id="pin-version">In case you want to pin a specific version</h2>
        <p className="note">
          To install a particular release instead of <code>latest</code>, set{' '}
          <code>RELEASE_TAG</code> on <code>bash</code> (not on <code>curl</code> — a
          pipe does not pass env vars to the right-hand command):
        </p>
        <pre>
          <code>{`curl -fsSL https://raw.githubusercontent.com/Tarun-Elango/outpost/latest/scripts/install.sh | RELEASE_TAG=v0.7.0 bash`}</code>
        </pre>
      </div>

      <div className="card">
        <h2 id="shared-install">Install once for every user on this machine</h2>
        <p className="note">
          Use this on a shared Mac or Linux desktop, or if you prefer{' '}
          <code>/usr/local/bin</code> over <code>~/.local/bin</code>. Installs the
          binary for all accounts (each user still has their own{' '}
          <code>~/.outpost</code> config). Requires <code>sudo</code> and skips shell
          config changes because <code>/usr/local/bin</code> is already on PATH.
        </p>
        <pre>
          <code>{`INSTALL_DIR=/usr/local/bin curl -fsSL https://raw.githubusercontent.com/Tarun-Elango/outpost/main/scripts/install.sh | sudo bash`}</code>
        </pre>
      </div>

      <div className="card">
        <h2 id="build-from-source">Build from source</h2>
        <p className="note">
          For contributors or inspecting the code. Requires{' '}
          <a href="https://go.dev/doc/install" target="_blank" rel="noreferrer">
            Go
          </a>
          .
        </p>

        <h3>Install Go</h3>
        <p className="note">If you don't have Go yet:</p>
        <pre>
          <code>{`# macOS
brew install go

# Linux (Debian/Ubuntu)
sudo apt install golang-go`}</code>
        </pre>
        <p className="note">
          Other ways: see{' '}
          <a href="https://go.dev/doc/install" target="_blank" rel="noreferrer">
            go.dev/doc/install
          </a>
          .
        </p>

        <p className="note">Clone the repo — all build steps below run from inside it:</p>
        <pre>
          <code>{`git clone https://github.com/Tarun-Elango/outpost.git
cd outpost`}</code>
        </pre>

        <h3>Build in the repo folder</h3>
        <p className="note">From the <code>outpost</code> directory:</p>
        <pre>
          <code>go build -o outpost .</code>
        </pre>
        <p className="note">Test the binary in place:</p>
        <pre>
          <code>./outpost version</code>
        </pre>

        <h3>Install system-wide</h3>
        <p className="note">
          Still from the <code>outpost</code> directory — puts <code>outpost</code> on
          your PATH so you can run it from anywhere:
        </p>
        <pre>
          <code>{`go build -o "$(go env GOPATH)/bin/outpost" .`}</code>
        </pre>
        <p className="note">
          If <code>$GOPATH/bin</code> isn't on your PATH yet, run:
        </p>
        <pre>
          <code>{`export GOPATH="\${GOPATH:-$HOME/go}"
export PATH="$GOPATH/bin:$PATH"`}</code>
        </pre>
        <p className="note">Verify:</p>
        <pre>
          <code>outpost version</code>
        </pre>
      </div>
    </DocPage>
  )
}
