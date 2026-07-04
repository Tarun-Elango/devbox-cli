import DocPage from './doc-page'

export default function CommandsDoc() {
  return (
    <DocPage title="All commands">
      <p className="note">
        Run <code>devbox help</code> to print usage, or use the reference
        below.
      </p>

      <div className="card">
        <h2>Config and health</h2>
        <ul>
          <li>
            <code>devbox version</code> — show the devbox CLI version
          </li>
          <li>
            <code>devbox update</code> — check GitHub releases for a newer version and
            install it after confirmation
          </li>
          <li>
            <code>devbox setup</code> — configure or change AWS credentials and region
            (stored in <code>~/.devbox/config.json</code>)
          </li>
          <li>
            <code>devbox clear-creds</code> — clear saved AWS credentials from{' '}
            <code>~/.devbox/config.json</code>
          </li>
          <li>
            <code>devbox health</code> — check config, AWS credentials, region, and
            database
          </li>
        </ul>
      </div>

      <div className="card">
        <h2>Boxes</h2>
        <ul>
          <li>
            <code>devbox create {'<name>'} [--from {'<amiId|name>'}]</code> — create a
            new box (optionally restore from a snapshot)
          </li>
          <li>
            <code>
              devbox create --template {'<template>'} [{'<template>'}...]{' '}
              {'<name>'} [--from {'<amiId|name>'}]
            </code>{' '}
            — create a box from one or more templates (optionally from snapshot)
          </li>
          <li>
            <code>devbox ls</code> — list all boxes
          </li>
          <li>
            <code>devbox status {'<id-or-name>'}</code> — show details for a box
          </li>
          <li>
            <code>devbox rename {'<id-or-name>'} {'<new-name>'}</code> — rename a box
          </li>
          <li>
            <code>devbox resize {'<id-or-name>'}</code> or{' '}
            <code>devbox upgrade {'<id-or-name>'}</code> — resize a stopped box instance
            type or root disk
          </li>
          <li>
            <code>devbox stop {'<id-or-name>'}</code> — stop a running box
          </li>
          <li>
            <code>devbox start {'<id-or-name>'}</code> — start a stopped box
          </li>
          <li>
            <code>devbox restart {'<id-or-name>'}</code> or{' '}
            <code>devbox reboot {'<id-or-name>'}</code> — reboot a running box
          </li>
          <li>
            <code>devbox delete {'<id-or-name>'}</code> — delete a box
          </li>
        </ul>
      </div>

      <div className="card">
        <h2>Connect and transfer</h2>

        <h3>SSH</h3>
        <pre>
          <code>devbox ssh [-i key] {'<id-or-name>'} [-- {'<ssh-option>'}...]</code>
        </pre>
        <p className="note">
          Open an SSH session. Default key: <code>~/.ssh/id_ed25519</code>. Use{' '}
          <code>--</code> before native ssh options (e.g. <code>-L 8080:localhost:8080</code>
          ). For one-off remote commands, use <code>exec</code>.
        </p>

        <h3>Copy &amp; sync</h3>
        <ul>
          <li>
            <code>devbox cp [-i key] {'<source>'} {'<dest>'}</code> — copy a file to or
            from a box
          </li>
          <li>
            <code>devbox sync [-i key] [--delete] {'<source>'} {'<dest>'}</code> — sync
            files or directories
          </li>
        </ul>
        <p className="note">
          Example: <code>devbox cp ./main.go mybox:/home/ec2-user/app/</code>.{' '}
          <code>--delete</code> removes destination files missing from source.
        </p>

        <h3>Exec &amp; forward</h3>
        <ul>
          <li>
            <code>devbox exec [-i key] [-s] [-t] {'<id-or-name>'} -- {'<command>'}</code>{' '}
            — run a one-off command on a running box
          </li>
          <li>
            <code>devbox forward {'<id-or-name>'} {'<port>'}</code> — forward a port from
            a box
          </li>
        </ul>
        <p className="note">
          <code>-s</code> runs as a shell snippet via <code>sh -lc</code>. <code>-t</code>{' '}
          allocates a pseudo-TTY for sudo or interactive commands.
        </p>
      </div>

      <div className="card">
        <h2>Snapshots</h2>
        <p className="note">
          A snapshot is a saved disk image of a box; restore one with{' '}
          <code>create --from</code>.
        </p>
        <ul>
          <li>
            <code>devbox snapshot</code> — list all snapshots
          </li>
          <li>
            <code>devbox snapshot create {'<id-or-name>'} {'<name>'}</code> — create a
            snapshot of a box
          </li>
          <li>
            <code>devbox snapshot ls {'<amiId-or-name>'}</code> — show details for a
            snapshot
          </li>
          <li>
            <code>devbox snapshot delete {'<amiId-or-name>'}</code> — delete a snapshot
          </li>
        </ul>
      </div>

      <div className="card">
        <h2>Templates</h2>
        <p className="note">
          Templates let you create boxes preloaded with libs, tools, and other setup.
        </p>
        <ul>
          <li>
            <code>devbox template</code> — list available templates
          </li>
          <li>
            <code>devbox template new {'<name>'} [command string]</code> — create a
            template with optional startup command
          </li>
          <li>
            <code>devbox template delete {'<name>'}</code> — delete a template
          </li>
          <li>
            <code>devbox template rename {'<name>'} {'<new-name>'}</code> — rename a
            template
          </li>
          <li>
            <code>devbox template search {'<query>'}</code> — search templates by name
            (returns partial matches)
          </li>
        </ul>
      </div>

      <div className="card">
        <h2>Idle stop</h2>
        <ul>
          <li>
            <code>devbox idle-stop set {'<id-or-name>'} {'<minutes>'}</code> — stop the
            box after inactivity
          </li>
          <li>
            <code>devbox idle-stop show {'<id-or-name>'}</code> — show idle-stop
            settings for a box
          </li>
          <li>
            <code>devbox idle-stop update {'<id-or-name>'} {'<minutes>'}</code> — update
            idle-stop timeout
          </li>
          <li>
            <code>devbox idle-stop delete {'<id-or-name>'}</code> — remove idle-stop from
            a box
          </li>
        </ul>
      </div>

      <div className="card">
        <h2>Git sync</h2>
        <p className="note">
          Use your local GitHub SSH key on a box (for <code>git push</code>,{' '}
          <code>git clone</code>, etc.) without copying it there: adds the key to{' '}
          <code>ssh-agent</code> and enables agent forwarding (<code>-A</code>) in the
          box&apos;s SSH config. Run again to undo both.
        </p>
        <ul>
          <li>
            <code>devbox git-sync {'<id-or-name>'}</code> — toggle GitHub SSH agent
            forwarding for a box
          </li>
        </ul>
      </div>
    </DocPage>
  )
}
