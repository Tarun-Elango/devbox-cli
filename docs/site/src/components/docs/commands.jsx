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
        Run <code>outpost help</code> for usage, or use the reference below.
      </p>

      <div className="card">
        <h2>Config and health</h2>
        <p className="note">CLI version, setup, credentials, and health checks.</p>
        <CmdTable
          rows={[
            ['outpost version', 'Show the outpost CLI version'],
            ['outpost update', 'Check for and install a newer release'],
            ['outpost setup', 'Configure AWS credentials and region'],
            ['outpost clear-creds', 'Clear saved AWS credentials'],
            ['outpost health', 'Check config, credentials, region, and database'],
          ]}
        />
      </div>

      <div className="card">
        <h2>Boxes</h2>
        <p className="note">Create, list, resize, and manage box instances.</p>
        <CmdTable
          rows={[
            [
              "outpost create <name> [--template <name>...] [--from <amiId|name>]",
              'Create a box, optionally from templates or a snapshot',
            ],
            ['outpost ls', 'List all boxes'],
            ['outpost status <id-or-name>', 'Show details for a box'],
            ['outpost rename <id-or-name> <new-name>', 'Rename a box'],
            [
              'outpost resize <id-or-name> · outpost upgrade <id-or-name>',
              'Resize instance type or root disk (box must be stopped)',
            ],
            ['outpost stop <id-or-name>', 'Stop a running box'],
            ['outpost start <id-or-name>', 'Start a stopped box'],
            [
              'outpost restart <id-or-name> · outpost reboot <id-or-name>',
              'Reboot a running box',
            ],
            ['outpost delete <id-or-name>', 'Delete a box'],
          ]}
        />
      </div>

      <div className="card">
        <h2>Connect and transfer</h2>
        <p className="note">SSH, file copy, sync, remote exec, and port forwarding.</p>
        <CmdTable
          rows={[
            [
              'outpost ssh [-i key] <id-or-name> [-- <ssh-option>...]',
              'Open an SSH session (use -- before native ssh flags)',
            ],
            [
              'outpost cp [-i key] <source> <dest>',
              'Copy a file to or from a box',
            ],
            [
              'outpost sync [-i key] [--delete] <source> <dest>',
              'Incremental directory sync via rsync; only dest is modified',
            ],
            [
              'outpost exec [-i key] [-s] [-t] <id-or-name> -- <command>',
              'Run a one-off command on a running box',
            ],
            ['outpost forward <id-or-name> <port>', 'Forward a port from a box'],
          ]}
        />
      </div>

      <div className="card">
        <h2>Snapshots</h2>
        <p className="note">Save and restore box disk images.</p>
        <CmdTable
          rows={[
            ['outpost snapshot [ls]', 'List all snapshots'],
            [
              'outpost snapshot ls <amiId-or-name>',
              'Show details for a snapshot',
            ],
            [
              'outpost snapshot create <id-or-name> <name>',
              'Create a snapshot of a box',
            ],
            [
              'outpost snapshot delete <amiId-or-name>',
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
            ['outpost template [ls]', 'List available templates'],
            [
              'outpost template new <name> [command]',
              'Create a template with optional startup command',
            ],
            ['outpost template delete <name>', 'Delete a template'],
            [
              'outpost template rename <name> <new-name>',
              'Rename a template',
            ],
            [
              'outpost template search <query>',
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
              'outpost idle-stop set <id-or-name> <minutes>',
              'Stop the box after inactivity',
            ],
            [
              'outpost idle-stop show <id-or-name>',
              'Show idle-stop settings',
            ],
            [
              'outpost idle-stop update <id-or-name> <minutes>',
              'Update idle-stop timeout',
            ],
            [
              'outpost idle-stop delete <id-or-name>',
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
              'outpost git-sync <id-or-name>',
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
            ['outpost budget [ls]', 'List all budgets'],
            ['outpost budget [ls] --refresh', 'Refresh budget list from AWS'],
            [
              'outpost budget create <name> <limit> <email>',
              'Create a monthly budget with spend alerts',
            ],
            ['outpost budget update <name>', 'Update a budget'],
            ['outpost budget delete <name>', 'Delete a budget'],
          ]}
        />
      </div>

      <div className="card">
        <h2>Uninstall</h2>
        <p className="note">Remove outpost from your machine.</p>
        <CmdTable
          rows={[
            [
              'outpost uninstall',
              'Remove the binary, ~/.outpost, ~/.outpost-backup, and shell PATH entries',
            ],
          ]}
        />
      </div>
    </DocPage>
  )
}
