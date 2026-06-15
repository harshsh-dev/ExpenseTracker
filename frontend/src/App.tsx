import { NavLink, Navigate, Route, Routes } from 'react-router-dom'
import {
  IconLayoutDashboard,
  IconCash,
  IconReceipt,
  IconTrendingUp,
  IconTag,
  IconDeviceFloppy,
  IconWallet,
  IconMoon,
  IconSun,
} from '@tabler/icons-react'
import type { Icon } from '@tabler/icons-react'
import { useTheme } from './theme'
import { useFeatures, type Feature } from './features'
import Dashboard from './modules/Dashboard'
import IncomePage from './modules/Income'
import ExpensesPage from './modules/Expenses'
import InvestmentsPage from './modules/Investments'
import CategoriesPage from './modules/Categories'
import SettingsPage from './modules/Settings'

type NavEntry = { to: string; label: string; icon: Icon; feature: Feature; element: React.ReactElement }

const nav: NavEntry[] = [
  { to: '/dashboard', label: 'Dashboard', icon: IconLayoutDashboard, feature: 'dashboard', element: <Dashboard /> },
  { to: '/income', label: 'Income', icon: IconCash, feature: 'income', element: <IncomePage /> },
  { to: '/expenses', label: 'Expenses', icon: IconReceipt, feature: 'expenses', element: <ExpensesPage /> },
  { to: '/investments', label: 'Investments', icon: IconTrendingUp, feature: 'investments', element: <InvestmentsPage /> },
  { to: '/categories', label: 'Categories', icon: IconTag, feature: 'categories', element: <CategoriesPage /> },
  { to: '/settings', label: 'Backup', icon: IconDeviceFloppy, feature: 'backup', element: <SettingsPage /> },
]

function NavItems({ items }: { items: NavEntry[] }) {
  return (
    <>
      {items.map(({ to, label, icon: Icon }) => (
        <NavLink
          key={to}
          to={to}
          className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`}
        >
          <Icon size={17} stroke={1.75} />
          {label}
        </NavLink>
      ))}
    </>
  )
}

function ThemeToggle() {
  const { theme, toggle } = useTheme()
  const dark = theme === 'dark'
  return (
    <button className="nav-item" onClick={toggle} title="Toggle theme">
      {dark ? <IconSun size={17} stroke={1.75} /> : <IconMoon size={17} stroke={1.75} />}
      {dark ? 'Light mode' : 'Dark mode'}
    </button>
  )
}

export default function App() {
  const features = useFeatures()
  const items = nav.filter((n) => features.has(n.feature))
  const home = items[0]?.to ?? '/dashboard'

  return (
    <div className="app">
      <aside className="sidebar">
        <div className="logo">
          <IconWallet size={20} stroke={1.75} /> Money Tracker
        </div>
        <NavItems items={items} />
        <div style={{ marginTop: 'auto' }}>
          <ThemeToggle />
        </div>
      </aside>

      <div style={{ flex: 1, minWidth: 0 }}>
        <nav className="mobile-nav">
          <NavItems items={items} />
          <ThemeToggle />
        </nav>
        <main className="main">
          <Routes>
            <Route path="/" element={<Navigate to={home} replace />} />
            {items.map((n) => (
              <Route key={n.to} path={n.to} element={n.element} />
            ))}
            <Route path="*" element={<Navigate to={home} replace />} />
          </Routes>
        </main>
      </div>
    </div>
  )
}
