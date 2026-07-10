import { Link } from 'react-router-dom'

const guides = [
  {
    to: '/how-tos/ssh',
    title: 'SSH',
    description: 'connect to your outpost from your local machine',
  },
  {
    to: '/how-tos/transfer',
    title: 'Transfer data and files',
    description: 'copy and sync files between local and your box',
  },
  {
    to: '/how-tos/github-sync',
    title: 'Sync GitHub account',
    description: 'use the GitHub account from your local machine on your box',
  },
  // {
  //   to: '/how-tos/remote-desktop',
  //   title: 'Remote desktop–like UI',
  //   description: 'work on your box with a desktop-style experience',
  // },
  {
    to: '/how-tos/vscode-ssh',
    title: 'VS Code & SSH without the outpost CLI',
    description: 'using regular ssh config, connect using VS Code, and differences between outpost ssh and regular ssh',
  },
  {
    to: '/how-tos/ai-sandbox',
    title: 'AI sandbox box',
    description: 'create a box with Codex or another AI tool for remote experiments',
  },
]

export default function HowTosIndexPage() {
  return (
    <>
      <h1>How tos</h1>
      <p className="tagline">Step-by-step guides for common outpost workflows.</p>

      <div className="card">
        <h2>Guides</h2>
        <ul>
          {guides.map(({ to, title, description }) => (
            <li key={to}>
              <Link to={to}>{title}</Link> — {description}
            </li>
          ))}
        </ul>
      </div>
    </>
  )
}
