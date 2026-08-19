// 后端 API 封装。开发时走 vite 代理 /api → localhost:8080。
// 多租户（M4b）：所有请求 credentials: "same-origin"（Session Cookie）；
// 修改类请求带 X-CSRF-Token（来自 /auth/me 的 csrf 字段）；统一 401 → 清本地
// 敏感态并跳登录（保留安全原路由）。
import type {
  AdminTenant,
  AdminUser,
  AgentEvent,
  AuthUser,
  ConfigResponse,
  CurrentTenant,
  LeadManagerConfig,
  MemberView,
  MemoryEntry,
  ReviewItem,
  SessionDetail,
  SessionListItem,
} from "../types";

const BASE = "/api";

// ── 会话 / CSRF / 401 处理 ──────────────────────────────
let csrfToken = "";
let unauthHandled = false;

// setCSRFToken 由 AuthProvider 在 me/login/register 成功后写入；同一会话稳定，
// 切换工作空间或恢复初始化时由后端轮换。
export function setCSRFToken(t: string) {
  csrfToken = t;
}

// resetUnauthFlag 登录成功后复位，使后续会话过期可再次触发统一跳转。
export function resetUnauthFlag() {
  unauthHandled = false;
}

// clearLocalKeys 清空本用户/本租户的本地敏感态（cda:* 前缀 + 遗留旧 key）。
export function clearLocalKeys() {
  try {
    const doomed: string[] = [];
    for (let i = 0; i < localStorage.length; i++) {
      const k = localStorage.key(i);
      if (k && (k.startsWith("cda:") || k.startsWith("draft-") || k === "sidebar")) doomed.push(k);
    }
    doomed.forEach((k) => localStorage.removeItem(k));
  } catch {
    /* 隐私模式等场景忽略 */
  }
  try {
    const doomed: string[] = [];
    for (let i = 0; i < sessionStorage.length; i++) {
      const k = sessionStorage.key(i);
      if (k && k.startsWith("cda:")) doomed.push(k);
    }
    doomed.forEach((k) => sessionStorage.removeItem(k));
  } catch {
    /* 隐私模式等场景忽略 */
  }
}

// handleUnauthorized 统一 401 出口：只触发一次（避免后台轮询连环跳转）。
function handleUnauthorized() {
  if (unauthHandled) return;
  unauthHandled = true;
  clearLocalKeys();
  const from = encodeURIComponent(window.location.pathname + window.location.search);
  window.location.assign(`/login?from=${from}`);
}

// isMutation 是否修改类请求（需 CSRF）。
function isMutation(init?: RequestInit): boolean {
  const m = (init?.method || "GET").toUpperCase();
  return m !== "GET" && m !== "HEAD" && m !== "OPTIONS";
}

async function req<T>(
  path: string,
  init?: RequestInit,
  opts?: { skipAuthRedirect?: boolean },
): Promise<T> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(init?.headers as Record<string, string>),
  };
  if (isMutation(init) && csrfToken) headers["X-CSRF-Token"] = csrfToken;
  const res = await fetch(BASE + path, {
    ...init,
    credentials: "same-origin",
    headers,
  });
  const text = await res.text();
  let data: { error?: string } | null = null;
  try {
    data = text ? JSON.parse(text) : null;
  } catch {
    data = null;
  }
  if (!res.ok) {
    if (res.status === 401 && !opts?.skipAuthRedirect) handleUnauthorized();
    throw new ApiError(data?.error || `HTTP ${res.status}`, res.status);
  }
  return data as unknown as T;
}

// ── 身份（M4b：登录 / 注册 / 退出 / me / 改密）────────────
// 登录失败（401 凭证错误）与注册冲突不触发统一跳转——由页面展示错误。

export const authMe = () => req<AuthUser>("/auth/me", undefined, { skipAuthRedirect: true });

export const authLogin = (email: string, password: string) =>
  req<AuthUser>(
    "/auth/login",
    { method: "POST", body: JSON.stringify({ email, password }) },
    { skipAuthRedirect: true },
  );

export const authRegister = (
  orgName: string,
  displayName: string,
  email: string,
  password: string,
) =>
  req<AuthUser>(
    "/auth/register",
    {
      method: "POST",
      body: JSON.stringify({ org_name: orgName, display_name: displayName, email, password }),
    },
    { skipAuthRedirect: true },
  );

export const authLogout = () => req<{ status: string }>("/auth/logout", { method: "POST" });

export const authChangePassword = (oldPassword: string, newPassword: string) =>
  req<{ status: string }>("/auth/password", {
    method: "PUT",
    body: JSON.stringify({ old_password: oldPassword, new_password: newPassword }),
  });

export const authSwitchTenant = (tenantId: string) =>
  req<AuthUser>("/auth/switch-tenant", {
    method: "POST",
    body: JSON.stringify({ tenant_id: tenantId }),
  });

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
  branch?: string,
): Promise<RunHandle> {
  return req<RunHandle>("/message", {
    method: "POST",
    body: JSON.stringify({
      text,
      session_id: sessionId,
      customer: customer || undefined,
      branch: branch || undefined,
    }),
  });
}

// subscribeStream 订阅会话事件流；返回 abort 函数（只取消订阅，不影响 Run）。
export function subscribeStream(
  sessionId: string,
  since: number,
  onEvent: (e: AgentEvent, run: string) => void,
  onSeq?: (seq: number) => void,
  onError?: (err: Error) => void,
): () => void {
  const ac = new AbortController();
  (async () => {
    const res = await fetch(`${BASE}/sessions/${sessionId}/stream?since=${since}`, {
      signal: ac.signal,
      credentials: "same-origin",
    });
    if (!res.ok) {
      if (res.status === 401) handleUnauthorized();
      throw new Error(`HTTP ${res.status}`);
    }
    await readSSE(res, onEvent, onSeq);
  })().catch((err) => {
    // AbortError=主动退订（卸载/切换），静默；其余为真实订阅故障，上报
    if (err instanceof Error && err.name === "AbortError") return;
    onError?.(err instanceof Error ? err : new Error(String(err)));
  });
  return () => ac.abort();
}

// ApiError 携带 HTTP 状态码（调用方按 status 分支，如 cancel 的 404=已结束）。
export class ApiError extends Error {
  status: number;
  constructor(message: string, status: number) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

// cancelRun 显式停止会话的活跃 Run（唯一停止途径）。404=Run 已结束。
export const cancelRun = (sessionId: string, runId: string) =>
  req<void>(`/sessions/${sessionId}/runs/${runId}/cancel`, { method: "POST" });

// truncateMessages 删除会话中 seq 及之后的消息（编辑重发用）。
export const truncateMessages = (sessionId: string, seq: number) =>
  req<{ branch_id: string }>(`/sessions/${sessionId}/messages/after?seq=${seq}`, {
    method: "DELETE",
  });

// sessionRunning 查询会话是否有活跃 Run（侧栏运行指示/切回恢复用）。
export async function sessionRunning(sessionId: string): Promise<boolean> {
  try {
    const r = await req<{ running?: boolean }>(`/sessions/${sessionId}/running`);
    return !!r.running;
  } catch {
    return false;
  }
}

export const sessionRunningState = (sessionId: string) =>
  req<{ running: boolean; run_id?: string }>(`/sessions/${sessionId}/running`);

// readSSE 从 fetch 响应体读 text/event-stream，逐事件回调；
// 解析 SSE id: 行作为游标（onSeq 回调，增量续传用）。
async function readSSE(
  res: Response,
  onEvent: (e: AgentEvent, run: string) => void,
  onSeq?: (seq: number) => void,
): Promise<void> {
  const reader = res.body!.getReader();
  const decoder = new TextDecoder();
  let buf = "";
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    buf += decoder.decode(value, { stream: true });
    let idx: number;
    while ((idx = buf.indexOf("\n\n")) >= 0) {
      const raw = buf.slice(0, idx);
      buf = buf.slice(idx + 2);
      let seq = -1;
      let payload = "";
      let run = "";
      for (const line of raw.split("\n")) {
        if (line.startsWith("id:")) {
          const n = parseInt(line.slice(3).trim(), 10);
          if (!Number.isNaN(n)) seq = n;
        } else if (line.startsWith("x-run:")) {
          run = line.slice(6).trim();
        } else if (line.startsWith("data:")) {
          payload = line.slice(5).trim();
        }
      }
      if (seq >= 0 && onSeq) onSeq(seq);
      if (!payload) continue;
      try {
        onEvent(JSON.parse(payload), run); // (事件, 产生它的 Run)——配对，replay 历史事件带自身归属
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

export interface CustomerCreateInput {
  name: string;
  industry?: string;
  scale?: string;
  tech_stack?: string[];
  existing_security?: string[];
  pain_points?: string[];
  procurement_pref?: string;
  notes?: string;
}

// 新会话客户登记专用入口：所有租户成员可创建，但不能覆盖同名客户。
export const customerCreate = (customer: CustomerCreateInput) =>
  req<{ status: string; name: string }>("/customers", {
    method: "POST",
    body: JSON.stringify(customer),
  });

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

// ── 后台任务（观测台）────────────────────────
export const taskLint = () =>
  req<{ id: string; type: string; status: string }>("/tasks/lint", { method: "POST" });

// ── 配置 ─────────────────────────────────────
// 租户安全视图 GET /api/config（model 别名 + 掩码 key，§10.1）；
// 完整配置/修改/探测/模型列表走 /api/platform/**（platform_admin 专用）。

export const configGet = () => req<ConfigResponse>("/config");

// ── 商机面板（Dashboard，只读代理）──────────
// 「高价值」口径 stage=mql（与 leads_search 一致）；未接入时 enabled=false。
// items 字段名随平台实况透传（后端包络容错解析），页面按候选名容错提取。
export interface LeadsDashboard {
  enabled: boolean;
  stage?: string;
  count?: number;
  total?: number | null; // null = 平台未返回总数
  page?: number;
  page_size?: number;
  items?: Record<string, unknown>[];
}

// 统计 best-effort：403/平台错误随 error 返回（200），面板降级展示原因。
export interface LeadsStats {
  enabled: boolean;
  metric?: string;
  data?: unknown;
  error?: string;
}

export const leadsDashboard = (page = 1, pageSize = 20, stage = "mql") =>
  req<LeadsDashboard>(
    `/leads/dashboard?stage=${encodeURIComponent(stage)}&page=${page}&page_size=${pageSize}`,
  );

export const leadsStats = (metric = "summary") =>
  req<LeadsStats>(`/leads/stats?metric=${encodeURIComponent(metric)}`);

export const configPut = (cfg: ConfigResponse) =>
  req<ConfigResponse>("/platform/config", {
    method: "PUT",
    body: JSON.stringify({
      models: cfg.models,
      router: cfg.router,
      lead_manager: cfg.lead_manager,
    }),
  });

export const platformConsoleConfig = <T>() => req<T>("/platform/config");

export const observeTasks = <T>() => req<T>("/tasks");
export const observeLLMAudit = <T>() => req<T>("/platform/llm-audit");
export const observeHealth = <T>() => req<T>("/console/health");

export function subscribePlatformLogs(
  onLog: (line: string) => void,
  onError?: (error: Error) => void,
): () => void {
  const ac = new AbortController();
  (async () => {
    const res = await fetch(`${BASE}/platform/logs`, {
      signal: ac.signal,
      credentials: "same-origin",
    });
    if (!res.ok) {
      if (res.status === 401) handleUnauthorized();
      throw new ApiError(`HTTP ${res.status}`, res.status);
    }
    if (!res.body) throw new Error("日志流响应体为空");
    const reader = res.body.getReader();
    const decoder = new TextDecoder();
    let buf = "";
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      buf += decoder.decode(value, { stream: true });
      let idx: number;
      while ((idx = buf.indexOf("\n\n")) >= 0) {
        const raw = buf.slice(0, idx);
        buf = buf.slice(idx + 2);
        for (const line of raw.split("\n")) {
          if (line.startsWith("data:")) onLog(line.slice(5).trim());
        }
      }
    }
  })().catch((err) => {
    if (err instanceof Error && err.name === "AbortError") return;
    onError?.(err instanceof Error ? err : new Error(String(err)));
  });
  return () => ac.abort();
}

// 仅更新商机平台配置（部分更新：不动 models/router）
export const leadManagerPut = (lm: LeadManagerConfig) =>
  req<ConfigResponse>("/platform/config", {
    method: "PUT",
    body: JSON.stringify({ lead_manager: lm }),
  });

// 仅更新行为参数（配置中心三态化：只动传了的字段）
export const configPutBehavior = (b: { llm_timeout_sec?: number; agent_max_iterations?: number }) =>
  req<ConfigResponse>("/platform/config", {
    method: "PUT",
    body: JSON.stringify(b),
  });

// 测试连接：发个 hi 验证模型配置（platform_admin）
export const configTest = (probe: {
  name?: string;
  endpoint: string;
  api_key?: string;
  protocol: string;
  model: string;
}) =>
  req<{ ok: boolean; latency_ms: number; reply?: string; error?: string }>(
    "/platform/config/test",
    {
      method: "POST",
      body: JSON.stringify(probe),
    },
  );

// 自动获取：代理网关 /models 拉可用模型列表（platform_admin）
export interface SessionSearchHit {
  session_id: string;
  title: string;
  customer: string;
  snippet: string;
  role: string;
  hit_title: boolean;
  updated_at: string;
}

export const sessionsSearch = async (q: string, limit = 20): Promise<SessionSearchHit[]> => {
  try {
    const r = await req<{ items: SessionSearchHit[] }>(
      `/sessions/search?q=${encodeURIComponent(q)}&limit=${limit}`,
    );
    return r.items || [];
  } catch {
    return [];
  }
};

export const listModels = (probe: {
  name?: string;
  endpoint: string;
  api_key?: string;
  protocol: string;
}) =>
  req<{ models: string[] }>("/platform/models", {
    method: "POST",
    body: JSON.stringify(probe),
  });

// ── 会话历史 ─────────────────────────────────
export const sessionsList = (limit = 20, offset = 0) =>
  req<{ count: number; items: SessionListItem[] }>(`/sessions?limit=${limit}&offset=${offset}`);

export const sessionGet = (id: string, branch?: string) =>
  req<SessionDetail>(
    branch ? `/sessions/${id}?branch=${encodeURIComponent(branch)}` : `/sessions/${id}`,
  );

export const sessionDelete = (id: string) =>
  req<{ status: string }>(`/sessions/${id}`, { method: "DELETE" });

export const sessionRename = (id: string, title: string) =>
  req<{ status: string }>(`/sessions/${id}`, {
    method: "PATCH",
    body: JSON.stringify({ title }),
  });

// ── 用户管理（平台级，platform_admin，§4.2）──────────────────
export const platformUsers = () => req<{ count: number; items: AdminUser[] }>("/platform/users");

export const platformUserCreate = (input: {
  email: string;
  display_name: string;
  password: string;
  tenant_id: string;
  role: string;
  platform_admin?: boolean;
}) =>
  req<{ status: string; user_id: string }>("/platform/users", {
    method: "POST",
    body: JSON.stringify(input),
  });

export const platformUserUpdate = (userId: string, email: string, displayName: string) =>
  req<{ status: string }>(`/platform/users/${userId}`, {
    method: "PATCH",
    body: JSON.stringify({ email, display_name: displayName }),
  });

export const platformUserDelete = (userId: string) =>
  req<{ status: string }>(`/platform/users/${userId}`, { method: "DELETE" });

export const platformTenants = () =>
  req<{ count: number; items: AdminTenant[] }>("/platform/tenants");

export const platformTenantCreate = (input: {
  name: string;
  slug?: string;
  owner_email: string;
  owner_display_name?: string;
  owner_password?: string;
}) =>
  req<{ status: string; tenant_id: string }>("/platform/tenants", {
    method: "POST",
    body: JSON.stringify(input),
  });

export const platformTenantUpdate = (tenantId: string, name: string, slug: string) =>
  req<{ status: string }>(`/platform/tenants/${tenantId}`, {
    method: "PATCH",
    body: JSON.stringify({ name, slug }),
  });

export const platformTenantStatus = (tenantId: string, status: string) =>
  req<{ status: string }>(`/platform/tenants/${tenantId}/status`, {
    method: "PATCH",
    body: JSON.stringify({ status }),
  });

export const platformTenantDelete = (tenantId: string) =>
  req<{ status: string }>(`/platform/tenants/${tenantId}`, { method: "DELETE" });

export const platformTenantMembers = (tenantId: string) =>
  req<{ count: number; items: MemberView[] }>(`/platform/tenants/${tenantId}/members`);

export const platformTenantMemberAdd = (
  tenantId: string,
  input: { email: string; display_name?: string; password?: string; role: string },
) =>
  req<{ status: string }>(`/platform/tenants/${tenantId}/members`, {
    method: "POST",
    body: JSON.stringify(input),
  });

export const platformTenantMemberRole = (tenantId: string, userId: string, role: string) =>
  req<{ status: string }>(`/platform/tenants/${tenantId}/members/${userId}/role`, {
    method: "PATCH",
    body: JSON.stringify({ role }),
  });

export const platformTenantMemberRemove = (tenantId: string, userId: string) =>
  req<{ status: string }>(`/platform/tenants/${tenantId}/members/${userId}`, {
    method: "DELETE",
  });

export const platformUserStatus = (userId: string, status: string) =>
  req<{ status: string }>(`/platform/users/${userId}/status`, {
    method: "PATCH",
    body: JSON.stringify({ status }),
  });

export const platformUserResetPassword = (userId: string, password: string) =>
  req<{ status: string }>(`/platform/users/${userId}/password`, {
    method: "POST",
    body: JSON.stringify({ password }),
  });

export const platformUserSetAdmin = (userId: string, on: boolean) =>
  req<{ status: string }>(`/platform/users/${userId}/platform-admin`, {
    method: "POST",
    body: JSON.stringify({ on }),
  });

// ── 租户成员管理（owner/admin，§4.2）────────────────────────
export const membersList = () => req<{ count: number; items: MemberView[] }>("/members");

export const membersAdd = (email: string, displayName: string, password: string, role: string) =>
  req<{ status: string }>("/members", {
    method: "POST",
    body: JSON.stringify({ email, display_name: displayName, password, role }),
  });

export const membersSetRole = (userId: string, role: string) =>
  req<{ status: string }>(`/members/${userId}/role`, {
    method: "PATCH",
    body: JSON.stringify({ role }),
  });

export const membersRemove = (userId: string) =>
  req<{ status: string }>(`/members/${userId}`, { method: "DELETE" });

export const currentTenantGet = () => req<CurrentTenant>("/tenant");

export const currentTenantUpdate = (name: string, slug: string) =>
  req<{ status: string }>("/tenant", {
    method: "PATCH",
    body: JSON.stringify({ name, slug }),
  });
