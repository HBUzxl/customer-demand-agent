import { useEffect, useState } from "react";

/**
 * 工具调用轨迹渲染（流式 + 回放共用一条路径）。
 *
 * 设计：时间线（每步 = 一次工具调用，左侧竖轨 + 节点），工具参数与结果
 * 优先语义化渲染（记忆条目是结构化数据，不该甩原始 JSON），
 * 未知形状兜底到语法着色的 JSON 树。
 */

export interface ToolTrace {
  tool: string;
  params: string;
  result?: string;
}

const API_BASE = "/api";

// ── 工具元数据：名称、中文标签、图标 ──────────────────────────────

const TOOL_META: Record<string, { label: string; icon: string }> = {
  memory_search: {
    label: "检索记忆",
    icon: "M11 4a7 7 0 1 0 0 14 7 7 0 0 0 0-14zM21 21l-4.35-4.35",
  },
  memory_recall: {
    label: "联想回忆",
    icon: "M12 3a9 9 0 1 0 9 9 9 9 0 0 0-9-9zM9 9h.01M15 9h.01M8.5 14a5 5 0 0 0 7 0",
  },
  memory_list: { label: "浏览记忆", icon: "M4 6h16M4 12h16M4 18h10" },
  memory_ensure: { label: "固化记忆", icon: "M5 12l4 4L19 6" },
  memory_observe: { label: "记录观察", icon: "M12 20h9M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4z" },
  memory_delete: {
    label: "删除记忆",
    icon: "M3 6h18M8 6V4a1 1 0 0 1 1-1h6a1 1 0 0 1 1 1v2m3 0v14a1 1 0 0 1-1 1H6a1 1 0 0 1-1-1V6",
  },
  analysis_submit: {
    label: "提交需求分析",
    icon: "M9 11l3 3L22 4M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11",
  },
};

const TYPE_LABEL: Record<string, string> = {
  threat: "威胁",
  compliance: "合规",
  industry: "行业",
  customer: "客户",
  product: "产品",
  user: "用户",
};

function safeParse(raw?: string): unknown {
  if (!raw) return undefined;
  try {
    return JSON.parse(raw);
  } catch {
    return undefined;
  }
}

function asObj(v: unknown): Record<string, unknown> | undefined {
  return typeof v === "object" && v !== null && !Array.isArray(v)
    ? (v as Record<string, unknown>)
    : undefined;
}

// ── JSON 兜底渲染（递归语法着色） ─────────────────────────────────

function JsonNode({ k, v, depth }: { k?: string; v: unknown; depth: number }) {
  const [open, setOpen] = useState(depth < 2);
  const isBranch =
    (typeof v === "object" && v !== null && Object.keys(v as object).length > 0) || false;
  if (!isBranch) {
    return (
      <div className="jv-line" style={{ paddingLeft: depth * 12 }}>
        {k !== undefined && <span className="jv-key">{k}</span>}
        {k !== undefined && ": "}
        <span className={`jv-${v === null ? "null" : typeof v}`}>
          {typeof v === "string" ? `"${v}"` : String(v)}
        </span>
      </div>
    );
  }
  const entries = Array.isArray(v)
    ? (v as unknown[]).map((x, i) => [String(i), x] as const)
    : Object.entries(v as Record<string, unknown>);
  return (
    <div>
      {k !== undefined && (
        <div className="jv-line" style={{ paddingLeft: depth * 12 }}>
          <button className="jv-toggle" onClick={() => setOpen((x) => !x)}>
            {open ? "▾" : "▸"}
          </button>
          <span className="jv-key">{k}</span>:{" "}
          {Array.isArray(v) ? `[${entries.length}]` : `{${entries.length}}`}
        </div>
      )}
      {open &&
        entries.map(([ek, ev]) => (
          <JsonNode key={ek} k={ek} v={ev} depth={k !== undefined ? depth + 1 : depth} />
        ))}
    </div>
  );
}

function JsonView({ data }: { data: unknown }) {
  return (
    <div className="jv">
      <JsonNode v={data} depth={0} />
    </div>
  );
}

export { JsonView };

// ── 记忆条目卡（search / recall / list 的结果） ───────────────────

interface MemItem {
  title?: string;
  summary?: string;
  status?: string;
  tags?: string[];
}

function MemResult({ data }: { data: unknown }) {
  const [open, setOpen] = useState(false);
  const obj = asObj(data);
  const count = typeof obj?.count === "number" ? obj.count : 0;
  const items = Array.isArray(obj?.items) ? (obj!.items as MemItem[]) : [];
  if (!obj || (!count && !items.length)) return null;
  const shown = open ? items : items.slice(0, 1);
  return (
    <div className="tt-mem">
      <button className="tt-mem-count" onClick={() => setOpen((x) => !x)}>
        {count} 条命中{items.length > 1 ? (open ? " ▴收起" : ` ▸展开全部 ${items.length} 条`) : ""}
      </button>
      {shown.map((it, i) => (
        <div key={i} className="tt-mem-item">
          <div className="tt-mem-head">
            <span className="tt-mem-title">{it.title || "未命名"}</span>
            {it.status && (
              <span className={`tt-status ${it.status === "verified" ? "ok" : "pending"}`}>
                {it.status === "verified" ? "已验证" : "待审核"}
              </span>
            )}
          </div>
          {it.summary && <div className="tt-mem-summary">{it.summary}</div>}
          {it.tags && it.tags.length > 0 && (
            <div className="tt-mem-tags">
              {it.tags.map((t) => (
                <span key={t}>{t}</span>
              ))}
            </div>
          )}
        </div>
      ))}
    </div>
  );
}

// ── 单步渲染 ─────────────────────────────────────────────────────

// parseNeedsReview 从 result JSON 提取审批信息（F4 内联审核）。
function parseNeedsReview(result?: string): { type: string; title: string } | null {
  if (!result) return null;
  try {
    const r = JSON.parse(result);
    if (r && r.needs_review && r.type && r.title) {
      return { type: r.type, title: r.title };
    }
  } catch {
    /* 非 JSON */
  }
  return null;
}

// ReviewCard：对话内审批卡（通过=verified 即刻生效 / 拒绝=归档 / 忽略）。
function ReviewCard({ type, title }: { type: string; title: string }) {
  const [state, setState] = useState<"pending" | "approved" | "rejected" | "ignored">("pending");
  const [busy, setBusy] = useState(false);
  // 回放/刷新恢复：挂载时拉条目当前状态（已审过的不重复打扰）
  useEffect(() => {
    fetch(`${API_BASE}/memory/${type}/${encodeURIComponent(title)}`)
      .then((r) => (r.ok ? r.json() : null))
      .then((e) => {
        if (e?.status === "verified") setState("approved");
        else if (e?.status === "pending" || e?.status === "pending_review") setState("pending");
      })
      .catch(() => {});
  }, [type, title]);
  async function act(approve: boolean) {
    setBusy(true);
    try {
      const res = await fetch(
        `${API_BASE}/review/${type}/${encodeURIComponent(title)}/${approve ? "approve" : "reject"}`,
        { method: "POST" },
      );
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setState(approve ? "approved" : "rejected");
    } catch {
      /* 失败保持 pending 可重试 */
    } finally {
      setBusy(false);
    }
  }
  if (state === "approved") return <div className="rv-done ok">✓ 已通过，知识已生效</div>;
  if (state === "rejected") return <div className="rv-done no">✗ 已拒绝（归档）</div>;
  if (state === "ignored") return <div className="rv-done">已忽略（稍后可在 /review 处理）</div>;
  return (
    <div className="rv-card">
      <div className="rv-title">📝 Agent 记了一条知识，待你确认</div>
      <div className="rv-entry">
        [{type}] {title}
      </div>
      <div className="rv-btns">
        <button className="btn sm" disabled={busy} onClick={() => act(true)}>
          {busy ? "…" : "通过"}
        </button>
        <button className="btn ghost sm" disabled={busy} onClick={() => act(false)}>
          拒绝
        </button>
        <button className="btn ghost sm" disabled={busy} onClick={() => setState("ignored")}>
          忽略
        </button>
      </div>
    </div>
  );
}

function ToolStep({ trace, index, last }: { trace: ToolTrace; index: number; last: boolean }) {
  const meta = TOOL_META[trace.tool] || {
    label: trace.tool,
    icon: "M12 2v6m0 12v-6M2 12h6m12 0h-6",
  };
  const p = asObj(safeParse(trace.params));
  const r = asObj(safeParse(trace.result));
  const isAnalysis = trace.tool === "analysis_submit";

  // 语义化参数行：query 引用文本 + 类型徽章（search/recall/list）
  const pType = typeof p?.type === "string" ? p.type : undefined;
  const pQuery = typeof p?.query === "string" ? p.query : undefined;
  const pTitle = typeof p?.title === "string" ? p.title : undefined;
  const rMessage = typeof r?.message === "string" ? r.message : undefined;

  const needsReview = parseNeedsReview(trace.result);
  const semResult =
    trace.tool === "memory_search" ||
    trace.tool === "memory_recall" ||
    trace.tool === "memory_list" ? (
      <MemResult data={safeParse(trace.result)} />
    ) : rMessage ? (
      <div className="tt-ok">{rMessage}</div>
    ) : null;

  return (
    <div className="tt-step">
      <div className="tt-rail">
        <span className="tt-dot">
          <svg
            width="13"
            height="13"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
          >
            <path d={meta.icon} />
          </svg>
        </span>
        {!last && <span className="tt-line" />}
      </div>
      <div className="tt-main">
        <div className="tt-head">
          <span className="tt-name">{meta.label}</span>
          {pType && <span className={`tt-type t-${pType}`}>{TYPE_LABEL[pType] || pType}</span>}
          {pQuery && <span className="tt-query">“{pQuery}”</span>}
          {pTitle && !pQuery && <span className="tt-query">“{pTitle}”</span>}
          {isAnalysis && <span className="tt-hint">生成下方分析卡片</span>}
        </div>
        {/* 语义结果优先；未知形状兜底 JSON 树 */}
        {semResult ?? (trace.result && !r ? <div className="tt-raw">{trace.result}</div> : null)}
        {!semResult && r && !rMessage && <JsonView data={r} />}
        {/* F4 内联审核：needs_review 的写入渲染审批卡 */}
        {needsReview && <ReviewCard type={needsReview.type} title={needsReview.title} />}
      </div>
    </div>
  );
}

// ── 对外：完整时间线 ─────────────────────────────────────────────

export default function ToolTimeline({ tools }: { tools: ToolTrace[] }) {
  return (
    <div className="tt">
      {tools.map((t, i) => (
        <ToolStep key={i} trace={t} index={i} last={i === tools.length - 1} />
      ))}
    </div>
  );
}
