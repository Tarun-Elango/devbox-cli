import { Link } from 'react-router-dom'

const topics = [
  {
    to: '/docs/install',
    title: 'Installation',
    description: 'Quick install, install with specific version or shared machine, github repo build.',
  },
  {
    to: '/docs/setup',
    title: 'Setup',
    description:
      'configure AWS credentials with outpost setup, IAM setup on AWS console',
  },
  {
    to: '/docs/boxes',
    title: 'Managing boxes',
    description: 'create, list, start, stop, delete, resize, and add an idle stop timer',
  },
  {
    to: '/docs/connect',
    title: 'Connect & transfer',
    description: 'ssh, cp, sync, exec, port forwarding, and sync local git ssh to remote box',
  },
  {
    to: '/docs/snapshots',
    title: 'Snapshots & templates',
    description: 'save images, restore boxes, custom templates',
  },
  {
    to: '/docs/budgets',
    title: 'Budgets',
    description: 'list, create, update, and delete AWS cost budgets',
  },
  {
    to: '/docs/commands',
    title: 'See all commands',
    description: 'full CLI command reference',
  },
]

export default function DocsIndexPage() {
  return (
    <>
      <h1>Documentation</h1>
      <p className="tagline">
        Guides for installing, configuring, and using outpost.
      </p>

      <div className="card">
        <h2>Topics</h2>
        <ul>
          {topics.map(({ to, title, description }) => (
            <li key={to}>
              <Link to={to}>{title}</Link> — {description}
            </li>
          ))}
        </ul>
      </div>
    </>
  )
}
