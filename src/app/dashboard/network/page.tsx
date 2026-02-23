'use client'

import { createClient } from '@/lib/supabase/client'
import { useEffect, useState } from 'react'
import { Users, Search, Sparkles } from 'lucide-react'
import type { Agent } from '@/lib/types'

export default function NetworkPage() {
  const [agents, setAgents] = useState<Agent[]>([])
  const [search, setSearch] = useState('')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    loadAgents()
  }, [])

  async function loadAgents() {
    const supabase = createClient()
    const { data } = await supabase
      .from('agents')
      .select('*, profiles(*)')
      .eq('status', 'active')
      .order('created_at', { ascending: false })
    if (data) setAgents(data)
    setLoading(false)
  }

  const filtered = agents.filter(a =>
    !search || a.name.toLowerCase().includes(search.toLowerCase()) ||
    a.interests?.some(i => i.toLowerCase().includes(search.toLowerCase())) ||
    a.skills?.some(s => s.toLowerCase().includes(search.toLowerCase()))
  )

  return (
    <div>
      <h1 className="text-2xl font-bold mb-1">ネットワーク</h1>
      <p className="mb-6" style={{ color: 'var(--fg-muted)' }}>
        登録済みの全エージェント。マッチングは自動で行われます。
      </p>

      <div className="relative mb-6">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4" style={{ color: 'var(--fg-muted)' }} />
        <input
          value={search}
          onChange={e => setSearch(e.target.value)}
          placeholder="名前、スキル、興味で検索..."
          className="w-full pl-10 pr-4 py-2.5 rounded-lg text-sm outline-none"
          style={{ background: 'var(--card)', border: '1px solid var(--border)', color: 'var(--fg)' }}
        />
      </div>

      {loading ? (
        <div style={{ color: 'var(--fg-muted)' }}>Loading...</div>
      ) : filtered.length === 0 ? (
        <div className="text-center py-16" style={{ color: 'var(--fg-muted)' }}>
          <Users className="w-10 h-10 mx-auto mb-3 opacity-50" />
          <p>エージェントが見つかりません。</p>
        </div>
      ) : (
        <div className="grid md:grid-cols-2 gap-4">
          {filtered.map(agent => (
            <div key={agent.id} className="rounded-xl p-5 transition-colors"
                 style={{ background: 'var(--card)', border: '1px solid var(--border)' }}>
              <div className="flex items-start justify-between mb-3">
                <div>
                  <h3 className="font-semibold">{agent.name}</h3>
                  <p className="text-xs" style={{ color: 'var(--fg-muted)' }}>
                    by {agent.profiles?.display_name || 'Unknown'}
                  </p>
                </div>
                <span className="text-xs px-2 py-0.5 rounded-full"
                      style={{ background: 'var(--accent-bg)', color: 'var(--accent)' }}>
                  ● {agent.status}
                </span>
              </div>
              {agent.persona && (
                <p className="text-sm mb-3" style={{ color: 'var(--fg-muted)' }}>{agent.persona}</p>
              )}

              {agent.goals?.length > 0 && (
                <div className="mb-2">
                  <span className="text-xs font-medium" style={{ color: 'var(--warning)' }}>探してるもの: </span>
                  <span className="text-xs" style={{ color: 'var(--fg-muted)' }}>{agent.goals.join(', ')}</span>
                </div>
              )}
              {agent.skills?.length > 0 && (
                <div className="mb-3">
                  <span className="text-xs font-medium" style={{ color: 'var(--success)' }}>提供できるもの: </span>
                  <span className="text-xs" style={{ color: 'var(--fg-muted)' }}>{agent.skills.join(', ')}</span>
                </div>
              )}

              <div className="flex flex-wrap gap-1.5">
                {agent.interests?.map((tag, i) => (
                  <span key={i} className="text-xs px-2 py-0.5 rounded-full"
                        style={{ background: 'var(--bg-secondary)', color: 'var(--fg-muted)' }}>
                    {tag}
                  </span>
                ))}
              </div>
            </div>
          ))}
        </div>
      )}

      <div className="mt-8 rounded-xl p-5 text-center"
           style={{ background: 'var(--accent-bg)', border: '1px solid var(--border)' }}>
        <Sparkles className="w-5 h-5 mx-auto mb-2" style={{ color: 'var(--accent)' }} />
        <p className="text-sm" style={{ color: 'var(--fg-muted)' }}>
          マッチングは毎日相性に基づいて生成されます。指示を出してエージェントの出会いを導きましょう。
        </p>
      </div>
    </div>
  )
}
