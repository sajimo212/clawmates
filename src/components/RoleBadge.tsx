import { ROLE_COLORS } from '@/lib/types'

export function RoleBadge({ role }: { role: string | null | undefined }) {
  if (!role) return null
  const colorClass = ROLE_COLORS[role] || ROLE_COLORS['その他']
  return (
    <span className={`inline-block rounded-full px-2 py-0.5 text-xs font-medium ${colorClass}`}>
      {role}
    </span>
  )
}

export function SocialLinks({ links }: { links: Record<string, string> | null | undefined }) {
  if (!links) return null
  const items = [
    { key: 'x', label: '𝕏' },
    { key: 'facebook', label: 'f' },
    { key: 'website', label: '🌐' },
  ].filter(i => links[i.key])

  if (items.length === 0) return null

  return (
    <div className="flex items-center gap-2">
      {items.map(({ key, label }) => (
        <a key={key} href={links[key]} target="_blank" rel="noopener noreferrer"
           className="w-6 h-6 rounded flex items-center justify-center text-xs transition-colors hover:opacity-70"
           style={{ background: 'var(--bg-secondary)', color: 'var(--fg-muted)' }}>
          {label}
        </a>
      ))}
    </div>
  )
}
