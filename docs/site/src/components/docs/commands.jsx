import DocPage from './doc-page'

function CmdTable({ rows }) {
  return (
    <table className="cmd-table">
      <thead>
        <tr>
          <th>Command</th>
          <th>Description</th>
        </tr>
      </thead>
      <tbody>
        {rows.map(([cmd, desc]) => (
          <tr key={cmd}>
            <td>
              <code>{cmd}</code>
            </td>
            <td>{desc}</td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}

export default function CommandsDoc() {
  return (
    <DocPage title="All commands">
      <p className="note">
        Run <code>devbox help</code> for usage, or use the reference below.
      </p>

      <div className="card">
        <h2>Config and health</h2>
        <p className="note">CLI version, setup, credentials, and health checks.</p>
        <CmdTable
          rows={[
            ['devbox version', 'Show the devbox CLI version'],
            ['devbox update', 'Check for and install a newer release'],
            ['devbox setup', 'Configure AWS credentials and region'],
            ['devbox clear-creds', 'Clear saved AWS credentials'],
            ['devbox health', 'Check config, credentials, region, and database'],
          ]}
        />
      </div>

      <div className="card">
        <h2>Boxes</h2>
        <p className="note">Create, list, resize, and manage box instances.</p>
        <CmdTable
          rows={[
            [
              "devbox create <name> [--template <name>...] [--from <amiId|name>]",
              'Create a box, optionally from templates or a snapshot',
            ],
            ['devbox ls', 'List all boxes'],
            ['devbox status <id-or-name>', 'Show details for a box'],
            ['devbox rename <id-or-name> <new-name>', 'Rename a box'],
            [
              'devbox resize <id-or-name> · devbox upgrade <id-or-name>',
              'Resize instance type or root disk (box must be stopped)',
            ],
            ['devbox stop <id-or-name>', 'Stop a running box'],
            ['devbox start <id-or-name>', 'Start a stopped box'],
            [
              'devbox restart <id-or-name> · devbox reboot <id-or-name>',
              'Reboot a running box',
            ],
            ['devbox delete <id-or-name>', 'Delete a box'],
          ]}
        />
      </div>

      <div className="card">
        <h2>Connect and transfer</h2>
        <p className="note">SSH, file copy, sync, remote exec, and port forwarding.</p>
        <CmdTable
          rows={[
            [
              'devbox ssh [-i key] <id-or-name> [-- <ssh-option>...]',
              'Open an SSH session (use -- before native ssh flags)',
            ],
            [
              'devbox cp [-i key] <source> <dest>',
              'Copy a file to or from a box',
            ],
            [
              'devbox sync [-i key] [--delete] <source> <dest>',
              'Incremental directory sync via rsync; only dest is modified',
            ],
            [
              'devbox exec [-i key] [-s] [-t] <id-or-name> -- <command>',
              'Run a one-off command on a running box',
            ],
            ['devbox forward <id-or-name> <port>', 'Forward a port from a box'],
          ]}
        />
      </div>

      <div className="card">
        <h2>Snapshots</h2>
        <p className="note">Save and restore box disk images.</p>
        <CmdTable
          rows={[
            ['devbox snapshot [ls]', 'List all snapshots'],
            [
              'devbox snapshot ls <amiId-or-name>',
              'Show details for a snapshot',
            ],
            [
              'devbox snapshot create <id-or-name> <name>',
              'Create a snapshot of a box',
            ],
            [
              'devbox snapshot delete <amiId-or-name>',
              'Delete a snapshot',
            ],
          ]}
        />
      </div>

      <div className="card">
        <h2>Templates</h2>
        <p className="note">Reusable box setups with preinstalled tools.</p>
        <CmdTable
          rows={[
            ['devbox template [ls]', 'List available templates'],
            [
              'devbox template new <name> [command]',
              'Create a template with optional startup command',
            ],
            ['devbox template delete <name>', 'Delete a template'],
            [
              'devbox template rename <name> <new-name>',
              'Rename a template',
            ],
            [
              'devbox template search <query>',
              'Search templates by name',
            ],
          ]}
        />
      </div>

      <div className="card">
        <h2>Idle stop</h2>
        <p className="note">Automatically stop boxes after inactivity.</p>
        <CmdTable
          rows={[
            [
              'devbox idle-stop set <id-or-name> <minutes>',
              'Stop the box after inactivity',
            ],
            [
              'devbox idle-stop show <id-or-name>',
              'Show idle-stop settings',
            ],
            [
              'devbox idle-stop update <id-or-name> <minutes>',
              'Update idle-stop timeout',
            ],
            [
              'devbox idle-stop delete <id-or-name>',
              'Remove idle-stop from a box',
            ],
          ]}
        />
      </div>

      <div className="card">
        <h2>Git sync</h2>
        <p className="note">
          Use your local GitHub SSH key on a box without copying it there.
        </p>
        <CmdTable
          rows={[
            [
              'devbox git-sync <id-or-name>',
              'Toggle GitHub SSH agent forwarding (run again to undo)',
            ],
          ]}
        />
      </div>

      <div className="card">
        <h2>Budgets</h2>
        <p className="note">List and manage AWS account cost budgets.</p>
        <CmdTable
          rows={[
            ['devbox budget [ls]', 'List all budgets'],
            ['devbox budget [ls] --refresh', 'Refresh budget list from AWS'],
            [
              'devbox budget create <name> <limit> <email>',
              'Create a monthly budget with spend alerts',
            ],
            ['devbox budget update <name>', 'Update a budget'],
            ['devbox budget delete <name>', 'Delete a budget'],
          ]}
        />
      </div>

      <div className="card">
        <h2>Uninstall</h2>
        <p className="note">Remove devbox from your machine.</p>
        <CmdTable
          rows={[
            [
              'devbox uninstall',
              'Remove the binary, ~/.devbox, ~/.devbox-backup, and shell PATH entries',
            ],
          ]}
        />
      </div>
    </DocPage>
  )
}
