import { useEffect } from 'react'
import { Route, Routes, useLocation } from 'react-router-dom'
import './index.css'
import SiteHeader from './components/header'
import AboutPage from './components/about'
import HowTosLayout from './components/how-tos'
import HowTosIndexPage from './components/how-tos/index-page'
import SshHowTo from './components/how-tos/ssh'
import TransferHowTo from './components/how-tos/transfer'
import GithubSyncHowTo from './components/how-tos/github-sync'
import RemoteDesktopHowTo from './components/how-tos/remote-desktop'
import VscodeSshHowTo from './components/how-tos/vscode-ssh'
import AiSandboxHowTo from './components/how-tos/ai-sandbox'
import DocsLayout from './components/docs'
import DocsIndexPage from './components/docs/index-page'
import InstallDoc from './components/docs/install'
import SetupDoc from './components/docs/setup'
import BoxesDoc from './components/docs/boxes'
import ConnectDoc from './components/docs/connect'
import SnapshotsDoc from './components/docs/snapshots'
import BudgetsDoc from './components/docs/budgets'
import ImportDoc from './components/docs/import'
import CommandsDoc from './components/docs/commands'
import PlanetDecoration from './components/planet-decoration'

function ScrollToTop() {
  const { pathname } = useLocation()

  useEffect(() => {
    window.scrollTo(0, 0)
  }, [pathname])

  return null
}

function App() {
  return (
    <>
      <PlanetDecoration />
      <div className="shell">
        <ScrollToTop />
        <SiteHeader />
        <div className="wrap">
          <Routes>
            <Route path="/" element={<AboutPage />} />
            <Route path="/how-tos" element={<HowTosLayout />}>
              <Route index element={<HowTosIndexPage />} />
              <Route path="ssh" element={<SshHowTo />} />
              <Route path="transfer" element={<TransferHowTo />} />
              <Route path="github-sync" element={<GithubSyncHowTo />} />
              <Route path="remote-desktop" element={<RemoteDesktopHowTo />} />
              <Route path="vscode-ssh" element={<VscodeSshHowTo />} />
              <Route path="ai-sandbox" element={<AiSandboxHowTo />} />
            </Route>
            <Route path="/docs" element={<DocsLayout />}>
              {/* below are the routes for the docs page, so docs/children */}
              <Route index element={<DocsIndexPage />} />
              <Route path="install" element={<InstallDoc />} />
              <Route path="setup" element={<SetupDoc />} />
              <Route path="boxes" element={<BoxesDoc />} />
              <Route path="connect" element={<ConnectDoc />} />
              <Route path="snapshots" element={<SnapshotsDoc />} />
              <Route path="budgets" element={<BudgetsDoc />} />
              <Route path="import" element={<ImportDoc />} />
              <Route path="commands" element={<CommandsDoc />} />
            </Route>
          </Routes>
        </div>
      </div>
    </>
  )
}

export default App
