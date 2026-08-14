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
}

export interface RouterConfig {
  default: string;
  routes: Record<string, string>;
  fallback: { max_retries: number; backoff_base_ms: number; chain: string[] | null };
}

export interface ConfigResponse {
  models: ModelConfig[];
  router: RouterConfig;
}

export interface SessionListItem {
  session_id: string;
  tenant_id: string;
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

export interface SessionDetail {
  session: SessionListItem;
  messages: MessageRecord[];
  tool_calls: {
    id: number;
    message_id: number; // 归属的 assistant 消息 id（回放归属边界）
    tool_name: string;
    params: string;
    result: string;
    seq: number;
  }[];
}

// Agent 流式事件（SSE）
export interface AgentEvent {
  type:
    "session" | "round" | "reasoning" | "content" | "tool_call" | "tool_result" | "done" | "error";
  text?: string; // reasoning/content 增量
  round?: number; // round 事件
  tool?: string; // tool_call/tool_result
  params?: string; // tool_call 参数
  result?: string; // tool_result 结果
  analysis?: AnalysisResult; // done（分析）
  content?: string; // done（追问）/ session id
  error?: string; // error
}
