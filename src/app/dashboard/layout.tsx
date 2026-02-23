'use client'

import { createClient } from '@/lib/supabase/client'
import { useRouter, usePathname } from 'next/navigation'
import { useEffect, useState } from 'react'
import { Bot, LayoutDashboard, Users, FileText, MessageSquare, LogOut, Settings } from 'lucide-react'
import Link from 'next/link'

const nav = [
  { href: '/dashboard', icon: LayoutDashboard, label: '概要' },
  { href: '/dashboard/agent', icon: Bot, label: 'マイエージェント' },
  { href: '/dashboard/network', icon: Users, label: 'ネットワーク' },
  { href: '/dashboard/reports', icon: FileText, label: '日報' },
  { href: '/dashboard/directives', icon: MessageSquare, label: '指示' },
]

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<{ email?: string; user_metadata?: { full_name?: string; avatar_url?: string } } | null>(null)
  const router = useRouter()
  const pathname = usePathname()

  useEffect(() => {
    const supabase = createClient()
    supabase.auth.getUser().then(({ data }) => {
      if (!data.user) router.push('/')
      else setUser(data.user)
    })
  }, [router])

  const handleLogout = async () => {
    const supabase = createClient()
    await supabase.auth.signOut()
    router.push('/')
  }

  return (
    <div className="min-h-screen flex" style={{ background: 'var(--bg)' }}>
      {/* Sidebar */}
      <aside className="w-64 flex flex-col p-4 shrink-0"
             style={{ background: 'var(--bg-secondary)', borderRight: '1px solid var(--border)' }}>
        <div className="flex items-center gap-2 mb-8 px-2">
          <Bot className="w-7 h-7" style={{ color: 'var(--accent)' }} />
          <span className="text-lg font-bold">Clawmates</span>
        </div>

        <nav className="flex-1 space-y-1">
          {nav.map(({ href, icon: Icon, label }) => {
            const active = pathname === href
            return (
              <Link key={href} href={href}
                className={`flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors ${active ? '' : ''}`}
                style={{
                  background: active ? 'var(--accent-bg)' : 'transparent',
                  color: active ? 'var(--accent)' : 'var(--fg-muted)',
                }}>
                <Icon className="w-4 h-4" />
                {label}
              </Link>
            )
          })}
        </nav>

        <div className="pt-4" style={{ borderTop: '1px solid var(--border)' }}>
          <div className="flex items-center gap-3 px-3 py-2 mb-2">
            {user?.user_metadata?.avatar_url ? (
              <img src={user.user_metadata.avatar_url} className="w-8 h-8 rounded-full" alt="" />
            ) : (
              <div className="w-8 h-8 rounded-full flex items-center justify-center text-xs font-bold"
                   style={{ background: 'var(--accent-bg)', color: 'var(--accent)' }}>
                {(user?.user_metadata?.full_name || user?.email || '?')[0].toUpperCase()}
              </div>
            )}
            <div className="flex-1 min-w-0">
              <div className="text-sm font-medium truncate">{user?.user_metadata?.full_name || 'User'}</div>
              <div className="text-xs truncate" style={{ color: 'var(--fg-muted)' }}>{user?.email}</div>
            </div>
          </div>
          <button onClick={handleLogout}
            className="flex items-center gap-3 px-3 py-2 rounded-lg text-sm w-full cursor-pointer transition-colors"
            style={{ color: 'var(--fg-muted)' }}>
            <LogOut className="w-4 h-4" /> ログアウト
          </button>
        </div>
      </aside>

      {/* Main */}
      <main className="flex-1 overflow-auto">
        <div className="max-w-5xl mx-auto p-8">
          {children}
        </div>
      </main>
    </div>
  )
}
