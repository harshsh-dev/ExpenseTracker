import type {
  ReactNode,
  ButtonHTMLAttributes,
  InputHTMLAttributes,
  SelectHTMLAttributes,
} from 'react'

export function Card({ children, className = '' }: { children: ReactNode; className?: string }) {
  return <div className={`card ${className}`}>{children}</div>
}

export function PageHeader({
  title,
  subtitle,
  action,
}: {
  title: string
  subtitle?: ReactNode
  action?: ReactNode
}) {
  return (
    <div className="topbar">
      <div>
        <h1>{title}</h1>
        {subtitle && <div className="subtitle">{subtitle}</div>}
      </div>
      {action}
    </div>
  )
}

export function Button({
  variant = 'primary',
  className = '',
  ...props
}: ButtonHTMLAttributes<HTMLButtonElement> & { variant?: 'primary' | 'ghost' | 'danger' }) {
  return <button className={`btn btn-${variant} ${className}`} {...props} />
}

export function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <label className="field">
      <span className="field-label">{label}</span>
      {children}
    </label>
  )
}

export function Input({ className = '', ...props }: InputHTMLAttributes<HTMLInputElement>) {
  return <input className={`input ${className}`} {...props} />
}

export function Select({ className = '', ...props }: SelectHTMLAttributes<HTMLSelectElement>) {
  return <select className={`select ${className}`} {...props} />
}

export function Modal({
  title,
  open,
  onClose,
  children,
}: {
  title: string
  open: boolean
  onClose: () => void
  children: ReactNode
}) {
  if (!open) return null
  return (
    <div className="modal-bg" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <h2>{title}</h2>
        {children}
      </div>
    </div>
  )
}

export function Empty({ children }: { children: ReactNode }) {
  return <div className="empty">{children}</div>
}

export function Pill({ children, color }: { children: ReactNode; color?: string }) {
  const c = color ?? '#888780'
  return (
    <span className="pill" style={{ background: c + '1f', color: shade(c) }}>
      <span className="pill-dot" style={{ background: c }} />
      {children}
    </span>
  )
}

// Slightly darken a hex color for readable pill text on its tint.
function shade(hex: string): string {
  const m = hex.replace('#', '')
  if (m.length !== 6) return hex
  const n = parseInt(m, 16)
  const r = Math.max(0, ((n >> 16) & 255) - 40)
  const g = Math.max(0, ((n >> 8) & 255) - 40)
  const b = Math.max(0, (n & 255) - 40)
  return `rgb(${r},${g},${b})`
}
