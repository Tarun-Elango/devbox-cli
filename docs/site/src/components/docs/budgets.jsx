import DocOutline from './doc-outline'
import DocPage from './doc-page'

const sections = [{ id: 'budgets', label: 'Budgets' }]

export default function BudgetsDoc() {
  return (
    <DocPage title="Budgets">
      <DocOutline items={sections} />

      <div className="card">
        <h2 id="budgets">Budgets</h2>
        <p className="note">
          List and manage AWS account cost budgets from the CLI. Results are cached under{' '}
          <code>~/.outpost/</code> for 12 hours. Requires the{' '}
          <code>AWSBudgetsActionsWithAWSResourceControlAccess</code> IAM policy.
        </p>

        <dl className="cmd-variant">
          <dt>List all budgets</dt>
          <dd>
            <code>outpost budget [ls]</code>
          </dd>
          <dd className="example">
            Example: <code>outpost budget ls</code>
          </dd>
        </dl>
        <dl className="cmd-variant">
          <dt>Refresh from AWS</dt>
          <dd>
            <code>outpost budget [ls] --refresh</code>
          </dd>
          <dd className="example">
            Example: <code>outpost budget ls --refresh</code>
          </dd>
        </dl>
        <dl className="cmd-variant">
          <dt>Create a monthly budget</dt>
          <dd>
            <code>outpost budget create {'<name>'} {'<limit>'} {'<email>'}</code>
          </dd>
          <dd className="example">
            Example: <code>outpost budget create outpost-monthly 50 you@example.com</code>
          </dd>
          <dd className="note">
            Alerts at 85% actual, 100% actual, and 100% forecasted spend.
          </dd>
        </dl>
        <dl className="cmd-variant">
          <dt>Update a budget</dt>
          <dd>
            <code>outpost budget update {'<name>'}</code>
          </dd>
          <dd className="example">
            Example: <code>outpost budget update outpost-monthly</code>
          </dd>
          <dd className="note">Interactively update name, limit, or alert email (Enter keeps each current value).</dd>
        </dl>
        <dl className="cmd-variant">
          <dt>Delete a budget</dt>
          <dd>
            <code>outpost budget delete {'<name>'}</code>
          </dd>
          <dd className="example">
            Example: <code>outpost budget delete outpost-monthly</code>
          </dd>
          <dd className="note">Quote names with spaces.</dd>
        </dl>
      </div>
    </DocPage>
  )
}
