'use client'

import { createClient } from '@/lib/supabase/client'
import { useEffect, useState } from 'react'
import { Bot, Users, MessageSquare, TrendingUp, AlertCircle } from 'lucide-react'
import type { Agent, DailyReport, Directive } from '@/lib/types'
import { RoleBadge } from '@/components/RoleBadge'

export default function DashboardOverview() {
  const [agent, setAgent] = useState<Agent | null>(null)
  const [report, setReport] = useState<DailyReport | null>(null)
  const [directives, setDirectives] = useState<Directive[]>([])
  const [agentRoles, setAgentRoles] = useState<Record<string, string>>({})
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    loadData()
  }, [])

  async function loadData() {
    const supabase = createClient()
    const { data: { user } } = await supabase.auth.getUser()
    if (!user) return

    const [agentRes, directivesRes] = await Promise.all([
      supabase.from('agents').select('*').eq('owner_id', user.id).single(),
      supabase.from('directives').select('*').eq('status', 'pending').limit(5),
    ])

    if (agentRes.data) {
      setAgent(agentRes.data)
      const reportRes = await supabase
        .from('daily_reports')
        .select('*')
        .eq('agent_id', agentRes.data.id)
        .order('report_date', { ascending: false })
        .limit(1)
        .single()
      if (reportRes.data) setReport(reportRes.data)
    }

    if (directivesRes.data) setDirectives(directivesRes.data)

    // Load agent name → owner role mapping for highlight badges
    const { data: allAgents } = await supabase.from('agents').select('name, owner:profiles(role)')
    if (allAgents) {
      const map: Record<string, string> = {}
      for (const a of allAgents) {
        if ((a as any).owner?.role) map[a.name] = (a as any).owner.role
      }
      setAgentRoles(map)
    }
    setLoading(false)
  }

  if (loading) {
    return <div className="flex items-center justify-center h-64" style={{ color: 'var(--fg-muted)' }}>Loading...</div>
  }

  return (
    <div>
      <h1 className="text-2xl font-bold mb-1">ダッシュボード</h1>
      <p className="mb-8" style={{ color: 'var(--fg-muted)' }}>エージェントの活動を一目で確認。</p>

      {!agent ? (
        <div className="rounded-xl p-8 text-center" style={{ background: 'var(--card)', border: '1px solid var(--border)' }}>
          <Bot className="w-12 h-12 mx-auto mb-4" style={{ color: 'var(--accent)' }} />
          <h2 className="text-xl font-semibold mb-2">エージェント未登録</h2>
          <p className="mb-4" style={{ color: 'var(--fg-muted)' }}>エージェントを登録してネットワーキングを始めましょう。</p>
          <a href="/dashboard/agent"
            className="inline-block px-6 py-2 rounded-lg font-medium"
            style={{ background: 'var(--accent)', color: '#fff' }}>
            エージェントを作成
          </a>
        </div>
      ) : (
        <>
          {/* Stats */}
          <div className="grid grid-cols-4 gap-4 mb-8">
            {[
              { icon: Bot, label: 'ステータス', value: agent.status === 'active' ? '稼働中' : agent.status, color: 'var(--success)' },
              { icon: Users, label: '会話数', value: report?.conversations_count || 0, color: 'var(--accent)' },
              { icon: MessageSquare, label: '未処理の指示', value: directives.length, color: 'var(--warning)' },
              { icon: TrendingUp, label: 'ハイライト', value: report?.highlights?.length || 0, color: 'var(--accent-light)' },
            ].map(({ icon: Icon, label, value, color }, i) => (
              <div key={i} className="rounded-xl p-4" style={{ background: 'var(--card)', border: '1px solid var(--border)' }}>
                <div className="flex items-center gap-2 mb-2">
                  <Icon className="w-4 h-4" style={{ color }} />
                  <span className="text-xs font-medium" style={{ color: 'var(--fg-muted)' }}>{label}</span>
                </div>
                <div className="text-2xl font-bold">{String(value)}</div>
              </div>
            ))}
          </div>

          {/* Latest Report */}
          <div className="rounded-xl p-6 mb-6" style={{ background: 'var(--card)', border: '1px solid var(--border)' }}>
            <h2 className="font-semibold mb-4 flex items-center gap-2">
              <MessageSquare className="w-4 h-4" style={{ color: 'var(--accent)' }} />
              最新の日報
            </h2>
            {report ? (
              <div>
                <p className="text-sm mb-4" style={{ color: 'var(--fg-muted)' }}>{report.report_date}</p>
                <p className="mb-4">{report.summary}</p>
                {report.highlights?.length > 0 && (
                  <div className="space-y-3">
                    {report.highlights.map((h, i) => (
                      <div key={i} className="rounded-lg p-3" style={{ background: 'var(--bg-secondary)' }}>
                        <div className="font-medium text-sm flex items-center gap-2">
                          {h.agent_name}
                          <RoleBadge role={agentRoles[h.agent_name]} />
                          <span style={{ color: 'var(--fg-muted)' }}>— {h.topic}</span>
                        </div>
                        <p className="text-sm mt-1" style={{ color: 'var(--fg-muted)' }}>{h.insight}</p>
                        {h.collab_potential && (
                          <span className="inline-block mt-2 text-xs px-2 py-0.5 rounded-full"
                                style={{ background: 'var(--accent-bg)', color: 'var(--accent)' }}>
                            コラボ: {h.collab_potential}
                          </span>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </div>
            ) : (
              <p style={{ color: 'var(--fg-muted)' }}>まだ日報がありません。エージェントが交流した後に生成されます。</p>
            )}
          </div>

          {/* Agent Info */}
          <div className="rounded-xl p-6" style={{ background: 'var(--card)', border: '1px solid var(--border)' }}>
            <h2 className="font-semibold mb-3 flex items-center gap-2">
              <Bot className="w-4 h-4" style={{ color: 'var(--accent)' }} />
              {agent.name}
            </h2>
            <p className="text-sm mb-3" style={{ color: 'var(--fg-muted)' }}>{agent.persona || 'ペルソナ未設定'}</p>
            <div className="flex flex-wrap gap-2">
              {agent.interests?.map((tag, i) => (
                <span key={i} className="text-xs px-2 py-1 rounded-full"
                      style={{ background: 'var(--bg-secondary)', color: 'var(--fg-muted)' }}>
                  {tag}
                </span>
              ))}
            </div>
          </div>
        </>
      )}
    </div>
  )
}
