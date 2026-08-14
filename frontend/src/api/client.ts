// 后端 API 封装。开发时走 vite 代理 /api → localhost:8080。
import type {
  AgentEvent,
  ConfigResponse,
  MemoryEntry,
  ReviewItem,
  SessionDetail,
  SessionListItem,
} from "../types";

const BASE = "/api";

async function req<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(BASE + path, {
    ...init,
    headers: { "Content-Type": "application/json", ...(init?.headers as Record<string, string>) },
  });
  const text = await res.text();
  let data: { error?: string } | null = null;
  try {
    data = text ? JSON.parse(text) : null;
  } catch {
    data = null;
  }
  if (!res.ok) {
    throw new Error(data?.error || `HTTP ${res.status}`);
  }
  return data as unknown as T;
}

// ── 对话（统一入口 /api/message，ADR-014 + F0 Run 任务）────────
// F0：POST 创建 Run 立即返回 202 {session_id, run_id}（执行与连接解耦）；
// 事件通过订阅式 SSE GET /api/sessions/{id}/stream?since=N 消费（replay+live）。

export interface RunHandle {
  session_id: string;
  run_id: string;
}

// submitMessage 提交消息并启动 Run（不等待执行）。
export async function submitMessage(
  text: string,
  sessionId: string | undefined,
  customer?: string,
): Promise<RunHandle> {
  const res = await fetch(BASE + "/message", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ text, session_id: sessionId, customer: customer || undefined }),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: `HTTP ${res.status}` }));
    throw new Error(err.error || `HTTP ${res.status}`);
  }
  return res.json();
}

// subscribeStream 订阅会话事件流；返回 abort 函数（只取消订阅，不影响 Run）。
export function subscribeStream(
  sessionId: string,
  since: number,
  onEvent: (e: AgentEvent) => void,
): () => void {
  const ac = new AbortController();
  (async () => {
    const res = await fetch(`${BASE}/sessions/${sessionId}/stream?since=${since}`, {
      signal: ac.signal,
    });
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    await readSSE(res, onEvent);
  })().catch(() => {
    /* 订阅中止（组件卸载/切换会话）——正常 */
  });
  return () => ac.abort();
}

// cancelRun 显式停止会话的活跃 Run（唯一停止途径）。
export async function cancelRun(sessionId: string, runId: string): Promise<void> {
  const res = await fetch(`${BASE}/sessions/${sessionId}/runs/${runId}/cancel`, {
    method: "POST",
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: `HTTP ${res.status}` }));
    throw new Error(err.error || `HTTP ${res.status}`);
  }
}

// truncateMessages 删除会话中 seq 及之后的消息（编辑重发用）。
export async function truncateMessages(sessionId: string, seq: number): Promise<void> {
  const res = await fetch(`${BASE}/sessions/${sessionId}/messages/after?seq=${seq}`, {
    method: "DELETE",
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: `HTTP ${res.status}` }));
    throw new Error(err.error || `HTTP ${res.status}`);
  }
}

// sessionRunning 查询会话是否有活跃 Run（侧栏运行指示/切回恢复用）。
export async function sessionRunning(sessionId: string): Promise<boolean> {
  try {
    const r = await req<{ running?: boolean }>(`/sessions/${sessionId}/running`);
    return !!r.running;
  } catch {
    return false;
  }
}

// readSSE 从 fetch 响应体读 text/event-stream，逐事件回调。
async function readSSE(res: Response, onEvent: (e: AgentEvent) => void): Promise<void> {
  const reader = res.body!.getReader();
  const decoder = new TextDecoder();
  let buf = "";
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    buf += decoder.decode(value, { stream: true });
    let idx: number;
    while ((idx = buf.indexOf("\n\n")) >= 0) {
      const raw = buf.slice(0, idx).trim();
      buf = buf.slice(idx + 2);
      if (!raw.startsWith("data:")) continue;
      const payload = raw.slice(5).trim();
      if (!payload) continue;
      try {
        onEvent(JSON.parse(payload));
      } catch {
        /* 忽略解析失败的行 */
      }
    }
  }
}

// ── 记忆管理 ─────────────────────────────────
export const memorySearch = (q: string, type = "", limit = 10) =>
  req<{ count: number; items: MemoryEntry[] }>(
    `/memory/search?q=${encodeURIComponent(q)}&type=${type}&limit=${limit}`,
  );

export const memoryList = (type: string, offset = 0, limit = 50) =>
  req<{ count: number; items: MemoryEntry[] }>(
    `/memory/list?type=${type}&offset=${offset}&limit=${limit}`,
  );

export const memoryGet = (type: string, title: string) =>
  req<MemoryEntry>(`/memory/${type}/${encodeURIComponent(title)}`);

export const memoryUpsert = (entry: Partial<MemoryEntry> & { type: string; title: string }) =>
  req<{ status: string }>("/memory", {
    method: "POST",
    body: JSON.stringify(entry),
  });

export const memoryDelete = (type: string, title: string, archive = true) =>
  req<{ status: string }>(`/memory/${type}/${encodeURIComponent(title)}?archive=${archive}`, {
    method: "DELETE",
  });

// ── 审核 ─────────────────────────────────────
export const reviewPending = () => req<{ count: number; items: ReviewItem[] }>("/review/pending");

export const reviewApprove = (type: string, title: string) =>
  req<{ status: string }>(`/review/${type}/${encodeURIComponent(title)}/approve`, {
    method: "POST",
  });

export const reviewReject = (type: string, title: string) =>
  req<{ status: string }>(`/review/${type}/${encodeURIComponent(title)}/reject`, {
    method: "POST",
  });

// ── 配置 ─────────────────────────────────────
export const configGet = () => req<ConfigResponse>("/config");

export const configPut = (cfg: ConfigResponse) =>
  req<ConfigResponse>("/config", {
    method: "PUT",
    body: JSON.stringify({ models: cfg.models, router: cfg.router }),
  });

// 测试连接：发个 hi 验证模型配置
export const configTest = (probe: {
  name?: string;
  endpoint: string;
  api_key?: string;
  protocol: string;
  model: string;
}) =>
  req<{ ok: boolean; latency_ms: number; reply?: string; error?: string }>("/config/test", {
    method: "POST",
    body: JSON.stringify(probe),
  });

// 自动获取：代理网关 /models 拉可用模型列表
export const listModels = (probe: {
  name?: string;
  endpoint: string;
  api_key?: string;
  protocol: string;
}) =>
  req<{ models: string[] }>("/models", {
    method: "POST",
    body: JSON.stringify(probe),
  });

// ── 会话历史 ─────────────────────────────────
export const sessionsList = (limit = 20, offset = 0) =>
  req<{ count: number; items: SessionListItem[] }>(`/sessions?limit=${limit}&offset=${offset}`);

export const sessionGet = (id: string) => req<SessionDetail>(`/sessions/${id}`);

export const sessionDelete = (id: string) =>
  req<{ status: string }>(`/sessions/${id}`, { method: "DELETE" });
