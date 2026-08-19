// 类型定义，与后端 domain / api 结构对齐。

export interface MatchedProduct {
  name: string;
  confidence: number;
  reason: string;
  suggestion: string;
}

export type Feasibility = "direct" | "custom" | "partner" | "reject";

export interface AnalysisResult {
  demand_analysis: string;
  matched_products: MatchedProduct[];
  feasibility: Feasibility;
  feasibility_detail: string;
  missing_info: string[];
}

export interface MemoryEntry {
  type: string;
  title: string;
  user_id?: string;
  aliases?: string[];
  tags?: string[];
  summary?: string;
  content?: string;
  category?: string;
  product?: string;
  status?: string;
  relevance?: number;
}

// （AnalyzeResponse/ToolCallTrace 已由 SSE 流式取代，见 AgentEvent）

export interface ReviewItem {
  type: string;
  title: string;
  summary: string;
  tags?: string[];
  content: string;
  status: string;
}

export interface ModelConfig {
  name: string;
  endpoint: string;
  api_key?: string;
  protocol: string; // openai-chat / openai-response / anthropic
  model: string;
  temperature: number;
  max_tokens: number;
  context_window?: number; // 最大上下文 token（可选）
  timeout_sec?: number; // 单次调用超时（可选；缺省取全局 LLMTimeoutSec）
  max_retries?: number; // 重试次数（可选；缺省取全局 fallback）
  enabled?: boolean; // false=禁用（路由时跳过）
  remark?: string; // 备注
}

export interface RouterConfig {
  default: string;
  routes: Record<string, string>;
  fallback: { max_retries: number; backoff_base_ms: number; chain: string[] | null };
}

// 商机平台（Lead Manager）配置——api_key 掩码语义同 LLM key：
// 已设置显示 "********"，未设置为空；PUT 原样发回 = 不改
export interface LeadManagerConfig {
  enabled: boolean;
  base_url: string;
  api_key?: string;
  timeout_sec?: number;
  rate_per_min?: number;
  burst?: number;
}

export interface ConfigResponse {
  models: ModelConfig[];
  router: RouterConfig;
  lead_manager?: LeadManagerConfig;
}

export interface SessionListItem {
  snippet?: string; // 内容搜索命中片段
  hit_title?: boolean;
  session_id: string;
  title: string;
  customer: string;
  created_at: string;
  updated_at: string;
}

export interface MessageRecord {
  id: number;
  role: string;
  content: string;
  tool_call_id?: string;
  seq: number;
  created_at: string;
}

export interface CheckpointRec {
  id: string;
  type: string; // initial / followup / reanalysis
  has_analysis: boolean;
  question?: string;
  answer?: string;
  created_at: string;
}

export interface SessionDetail {
  session: SessionListItem;
  messages: MessageRecord[];
  checkpoints: CheckpointRec[];
  branches: string[];
  tool_calls: {
    id: number;
    message_id: number; // 归属的 assistant 消息 id（回放归属边界）
    tool_name: string;
    params: string;
    result: string;
    seq: number;
  }[];
}

// ── 身份 / 多租户（M4b）────────────────────────────
export const ROLE_PLATFORM_ADMIN = "platform_admin";
export const ROLE_OWNER = "owner";
export const ROLE_ADMIN = "admin";
export const ROLE_ANALYST = "analyst";
export const ROLE_REVIEWER = "reviewer";

// /api/auth/me、/api/auth/login、/api/auth/register 的最小响应视图
// （不含密码 Hash / Token / 内部目录，§10.1）。
export interface AuthUser {
  user_id: string;
  email: string;
  display_name: string;
  tenant_id: string;
  tenant_name: string;
  slug: string;
  roles: string[];
  workspaces: WorkspaceView[];
  csrf: string; // CSRF raw（轮换下发，修改类请求须回传 X-CSRF-Token）
}

export interface WorkspaceView {
  tenant_id: string;
  tenant_name: string;
  slug: string;
  role: string;
  status: string;
}

// ── 用户/成员管理（平台 + 租户，§4.2）────────────────
export interface AdminMembership {
  tenant_id: string;
  tenant_name: string;
  slug: string;
  role: string;
  status: string;
}

export interface AdminUser {
  user_id: string;
  email: string;
  display_name: string;
  status: string;
  platform_admin: boolean;
  last_login_at?: string;
  created_at: string;
  memberships: AdminMembership[];
}

export interface AdminTenant {
  tenant_id: string;
  name: string;
  slug: string;
  status: string;
  member_count: number;
  created_by?: string;
  created_at: string;
}

export interface CurrentTenant {
  tenant_id: string;
  name: string;
  slug: string;
  status: string;
  created_at: string;
}

export interface MemberView {
  user_id: string;
  email: string;
  display_name: string;
  role: string;
  status: string;
  last_login_at?: string;
  created_at: string;
}

export const MEMBER_ROLES = [
  { value: ROLE_OWNER, label: "所有者" },
  { value: ROLE_ADMIN, label: "管理员" },
  { value: ROLE_ANALYST, label: "分析人员" },
  { value: ROLE_REVIEWER, label: "审核人员" },
] as const;

// Agent 流式事件（SSE）
export interface AgentEvent {
  type:
    | "session"
    | "round"
    | "reasoning"
    | "content"
    | "tool_call"
    | "tool_result"
    | "done"
    | "error"
    | "ask_user";
  text?: string; // reasoning/content 增量
  round?: number; // round 事件
  tool?: string; // tool_call/tool_result
  params?: string; // tool_call 参数
  result?: string; // tool_result 结果
  cached?: boolean; // tool_result 命中本轮只读缓存
  analysis?: AnalysisResult; // done（分析）
  content?: string; // done（追问）/ session id
  error?: string; // error
  question?: string; // ask_user 事件：问题
  options?: { label: string; value?: string; description?: string; input?: string }[]; // ask_user 选项（input=选此项需补充信息的提示）
  run?: string;
}
