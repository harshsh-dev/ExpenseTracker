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
import Dashboard from './modules/Dashboard'
import IncomePage from './modules/Income'
import ExpensesPage from './modules/Expenses'
import InvestmentsPage from './modules/Investments'
import CategoriesPage from './modules/Categories'
import SettingsPage from './modules/Settings'

const nav: { to: string; label: string; icon: Icon }[] = [
  { to: '/dashboard', label: 'Dashboard', icon: IconLayoutDashboard },
  { to: '/income', label: 'Income', icon: IconCash },
  { to: '/expenses', label: 'Expenses', icon: IconReceipt },
  { to: '/investments', label: 'Investments', icon: IconTrendingUp },
  { to: '/categories', label: 'Categories', icon: IconTag },
  { to: '/settings', label: 'Backup', icon: IconDeviceFloppy },
]

function NavItems() {
  return (
    <>
      {nav.map(({ to, label, icon: Icon }) => (
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
  return (
    <div className="app">
      <aside className="sidebar">
        <div className="logo">
          <IconWallet size={20} stroke={1.75} /> Money Tracker
        </div>
        <NavItems />
        <div style={{ marginTop: 'auto' }}>
          <ThemeToggle />
        </div>
      </aside>

      <div style={{ flex: 1, minWidth: 0 }}>
        <nav className="mobile-nav">
          <NavItems />
          <ThemeToggle />
        </nav>
        <main className="main">
          <Routes>
            <Route path="/" element={<Navigate to="/dashboard" replace />} />
            <Route path="/dashboard" element={<Dashboard />} />
            <Route path="/income" element={<IncomePage />} />
            <Route path="/expenses" element={<ExpensesPage />} />
            <Route path="/investments" element={<InvestmentsPage />} />
            <Route path="/categories" element={<CategoriesPage />} />
            <Route path="/settings" element={<SettingsPage />} />
          </Routes>
        </main>
      </div>
    </div>
  )
}
