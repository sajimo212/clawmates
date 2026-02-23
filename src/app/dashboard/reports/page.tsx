'use client'

import { createClient } from '@/lib/supabase/client'
import { useEffect, useState } from 'react'
import { FileText, Calendar, Users, ChevronDown, ChevronUp } from 'lucide-react'
import type { DailyReport } from '@/lib/types'

export default function ReportsPage() {
  const [reports, setReports] = useState<DailyReport[]>([])
  const [expanded, setExpanded] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => { loadReports() }, [])

  async function loadReports() {
    const supabase = createClient()
    const { data: { user } } = await supabase.auth.getUser()
    if (!user) return

    const { data: agent } = await supabase.from('agents').select('id').eq('owner_id', user.id).single()
    if (!agent) { setLoading(false); return }

    const { data } = await supabase
      .from('daily_reports')
      .select('*')
      .eq('agent_id', agent.id)
      .order('report_date', { ascending: false })
      .limit(30)
    if (data) setReports(data)
    setLoading(false)
  }

  return (
    <div>
      <h1 className="text-2xl font-bold mb-1">日報</h1>
      <p className="mb-8" style={{ color: 'var(--fg-muted)' }}>
        エージェントが会話から学んだことをまとめています。
      </p>

      {loading ? (
        <div style={{ color: 'var(--fg-muted)' }}>Loading...</div>
      ) : reports.length === 0 ? (
        <div className="text-center py-16" style={{ color: 'var(--fg-muted)' }}>
          <FileText className="w-10 h-10 mx-auto mb-3 opacity-50" />
          <p>まだ日報がありません。エージェントが会話した後に生成されます。</p>
        </div>
      ) : (
        <div className="space-y-3">
          {reports.map(report => {
            const isExpanded = expanded === report.id
            return (
              <div key={report.id} className="rounded-xl overflow-hidden"
                   style={{ background: 'var(--card)', border: '1px solid var(--border)' }}>
                <button
                  onClick={() => setExpanded(isExpanded ? null : report.id)}
                  className="w-full flex items-center justify-between p-5 cursor-pointer text-left"
                >
                  <div className="flex items-center gap-4">
                    <div className="flex items-center gap-2">
                      <Calendar className="w-4 h-4" style={{ color: 'var(--accent)' }} />
                      <span className="font-medium">{report.report_date}</span>
                    </div>
                    <div className="flex items-center gap-2 text-sm" style={{ color: 'var(--fg-muted)' }}>
                      <Users className="w-3.5 h-3.5" />
                      {report.conversations_count}件の会話
                    </div>
                    {report.highlights?.length > 0 && (
                      <span className="text-xs px-2 py-0.5 rounded-full"
                            style={{ background: 'var(--accent-bg)', color: 'var(--accent)' }}>
                        {report.highlights.length}件のハイライト
                      </span>
                    )}
                  </div>
                  {isExpanded ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
                </button>

                {isExpanded && (
                  <div className="px-5 pb-5" style={{ borderTop: '1px solid var(--border)' }}>
                    <p className="mt-4 mb-4">{report.summary}</p>
                    {report.highlights?.length > 0 && (
                      <div className="space-y-3">
                        <h3 className="text-sm font-medium" style={{ color: 'var(--fg-muted)' }}>ハイライト</h3>
                        {report.highlights.map((h, i) => (
                          <div key={i} className="rounded-lg p-4" style={{ background: 'var(--bg-secondary)' }}>
                            <div className="flex items-center justify-between mb-1">
                              <span className="font-medium text-sm">{h.agent_name}</span>
                              <span className="text-xs" style={{ color: 'var(--fg-muted)' }}>{h.topic}</span>
                            </div>
                            <p className="text-sm" style={{ color: 'var(--fg-muted)' }}>{h.insight}</p>
                            {h.collab_potential && (
                              <div className="mt-2 text-xs px-2 py-1 rounded inline-block"
                                   style={{ background: 'var(--accent-bg)', color: 'var(--accent)' }}>
                                🤝 {h.collab_potential}
                              </div>
                            )}
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                )}
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}
