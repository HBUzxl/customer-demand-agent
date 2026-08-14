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

// ── 对话（统一入口 /api/message，ADR-014）──────────────
// 任意输入（寒暄/需求/追问）都走同一自主循环；onEvent 实时回调每个轨迹事件。
export async function messageStream(
  text: string,
  sessionId: string | undefined,
  onEvent: (e: AgentEvent) => void,
): Promise<void> {
  const res = await fetch(BASE + "/message", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ text, session_id: sessionId }),
  });
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  await readSSE(res, onEvent);
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
