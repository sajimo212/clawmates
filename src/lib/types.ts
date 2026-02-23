export interface Profile {
  id: string
  display_name: string
  avatar_url: string | null
  bio: string | null
  role: string | null
  company: string | null
  title: string | null
  social_links: Record<string, string>
  created_at: string
}

export const ROLE_OPTIONS = [
  '企業人事', '学生', 'エンジニア', 'デザイナー', 'VC・投資家', '起業家', 'リサーチャー', 'その他',
] as const

export const ROLE_COLORS: Record<string, string> = {
  '企業人事': 'bg-blue-100 text-blue-700',
  '学生': 'bg-green-100 text-green-700',
  'エンジニア': 'bg-purple-100 text-purple-700',
  'デザイナー': 'bg-pink-100 text-pink-700',
  'VC・投資家': 'bg-amber-100 text-amber-700',
  '起業家': 'bg-red-100 text-red-700',
  'リサーチャー': 'bg-cyan-100 text-cyan-700',
  'その他': 'bg-gray-100 text-gray-700',
}

export interface Agent {
  id: string
  owner_id: string
  name: string
  persona: string | null
  goals: string[]
  skills: string[]
  interests: string[]
  api_key: string
  status: 'active' | 'inactive' | 'paused'
  created_at: string
  profiles?: Profile
}

export interface Conversation {
  id: string
  agent_a: string
  agent_b: string
  status: 'active' | 'closed' | 'archived'
  topic: string | null
  compatibility_score: number | null
  created_at: string
}

export interface Message {
  id: string
  conversation_id: string
  sender_agent_id: string
  content: string
  metadata: Record<string, unknown>
  created_at: string
}

export interface DailyReport {
  id: string
  agent_id: string
  report_date: string
  summary: string
  conversations_count: number
  highlights: ReportHighlight[]
  created_at: string
}

export interface ReportHighlight {
  agent_name: string
  topic: string
  insight: string
  collab_potential: string
}

export interface Directive {
  id: string
  agent_id: string
  instruction: string
  status: 'pending' | 'in_progress' | 'completed' | 'cancelled'
  result: string | null
  created_at: string
  completed_at: string | null
}
