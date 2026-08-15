import { useState } from "react";
import { useEffect } from "react";
import { useParams, Link } from "react-router-dom";
import { sessionGet } from "../api/client";
import type { AnalysisResult, CheckpointRec, SessionDetail } from "../types";
import MarkdownView from "../components/MarkdownView";
import ResultCard from "../components/ResultCard";
import { JsonView } from "../components/ToolTrace";
import type { ToolTrace } from "../components/ToolTrace";

const ANALYSIS_TOOL = "analysis_submit";

function parseSubmitParams(params: string): AnalysisResult | undefined {
  try {
    const r = JSON.parse(params);
    if (r && r.demand_analysis) return r as AnalysisResult;
  } catch {
    /* 非 JSON 参数 */
  }
  return undefined;
}

function safeParse(raw?: string): unknown {
  if (!raw) return undefined;
  try {
    return JSON.parse(raw);
  } catch {
    return undefined;
  }
}

/** 时间线上的一行事件。 */
type Row =
  | { kind: "user"; id: number; time: string; content: string }
  | { kind: "system"; id: number; time: string; content: string }
  | { kind: "tool"; id: number; time: string; tc: ToolTrace }
  | { kind: "assistant"; id: number; time: string; content: string; analysis?: AnalysisResult }
  | { kind: "checkpoint"; id: string; time: string; cp: CheckpointRec };

interface Msg {
  id: number;
  role: string;
  content: string;
  created_at: string;
}

/**
 * 把会话历史展开成时间线行：user → system（本轮系统提示词）→ 工具调用（按
 * message_id 归属、调换到 assistant 之前——真实发生顺序）→ assistant 回复。
 */
function buildRows(d: SessionDetail): Row[] {
  // P4：checkpoint 行（记忆系统在做什么）——近似插在对应位置（初始/重分析在
  // user 行前、followup 在 assistant 行后）按创建顺序穿插
  const cps = d.checkpoints || [];
  let cpIdx = 0;
  const byMessage = new Map<number, ToolTrace[]>();
  for (const c of d.tool_calls || []) {
    const list = byMessage.get(c.message_id) || [];
    list.push({ tool: c.tool_name, params: c.params, result: c.result });
    byMessage.set(c.message_id, list);
  }
  const rows: Row[] = [];
  let pendingSystem: Row | null = null;
  for (const m of d.messages || []) {
    const msg = m as Msg;
    const time = new Date(msg.created_at).toLocaleTimeString("zh-CN", { hour12: false });
    if (msg.role === "user") {
      pendingSystem = null; // 用户开新轮，丢弃未消费的 system（防御）
      // P4：先弹 initial/reanalysis 类 checkpoint（它们跟随需求输入产生）
      while (cpIdx < cps.length && cps[cpIdx].type !== "followup") {
        rows.push({
          kind: "checkpoint",
          id: cps[cpIdx].id,
          time: new Date(cps[cpIdx].created_at).toLocaleTimeString("zh-CN", { hour12: false }),
          cp: cps[cpIdx],
        });
        cpIdx++;
      }
      rows.push({ kind: "user", id: msg.id, time, content: msg.content });
    } else if (msg.role === "system") {
      pendingSystem = { kind: "system", id: msg.id, time, content: msg.content };
    } else if (msg.role === "assistant") {
      if (pendingSystem) {
        rows.push(pendingSystem);
        pendingSystem = null;
      }
      const tools = byMessage.get(msg.id) || [];
      for (let i = 0; i < tools.length; i++) {
        rows.push({ kind: "tool", id: msg.id * 1000 + i, time, tc: tools[i] });
      }
      let analysis: AnalysisResult | undefined;
      for (let j = tools.length - 1; j >= 0; j--) {
        if (tools[j].tool === ANALYSIS_TOOL) {
          analysis = parseSubmitParams(tools[j].params);
          break;
        }
      }
      rows.push({ kind: "assistant", id: msg.id, time, content: msg.content, analysis });
      // P4：followup checkpoint 跟随 assistant 轮次
      while (cpIdx < cps.length && cps[cpIdx].type === "followup") {
        rows.push({
          kind: "checkpoint",
          id: cps[cpIdx].id,
          time: new Date(cps[cpIdx].created_at).toLocaleTimeString("zh-CN", { hour12: false }),
          cp: cps[cpIdx],
        });
        cpIdx++;
      }
    }
  }
  return rows;
}

const KIND_META: Record<string, { label: string; cls: string }> = {
  user: { label: "用户", cls: "k-user" },
  system: { label: "系统提示词", cls: "k-system" },
  tool: { label: "工具调用", cls: "k-tool" },
  assistant: { label: "助手", cls: "k-assistant" },
  checkpoint: { label: "记忆检查点", cls: "k-cp" },
};

function summaryOf(r: Row): string {
  switch (r.kind) {
    case "user":
      return r.content.replace(/\s+/g, " ").slice(0, 80) || "—";
    case "system":
      return `System Prompt · ${r.content.length} 字`;
    case "tool":
      return r.tc.tool;
    case "assistant":
      return r.content.replace(/\s+/g, " ").slice(0, 80) || "—";
    case "checkpoint": {
      const t =
        { initial: "首次分析", followup: "追问", reanalysis: "重分析" }[r.cp.type] || r.cp.type;
      return `${t}${r.cp.has_analysis ? " · 含结构化分析" : ""}${r.cp.question ? ` · Q: ${r.cp.question.slice(0, 30)}` : ""}`;
    }
  }
}

function CopyBtn({ text }: { text: string }) {
  const [done, setDone] = useState(false);
  return (
    <button
      className={`msg-copy${done ? " done" : ""}`}
      title="复制"
      onClick={() => {
        navigator.clipboard.writeText(text).then(() => {
          setDone(true);
          setTimeout(() => setDone(false), 1200);
        });
      }}
    >
      {done ? "✓" : "⧉"}
    </button>
  );
}

function Detail({ row }: { row: Row }) {
  switch (row.kind) {
    case "user":
      return (
        <div className="rp-full msg-wrap" style={{ position: "relative" }}>
          {row.content}
          <CopyBtn text={row.content} />
        </div>
      );
    case "system":
      return <pre className="rp-full mono">{row.content}</pre>;
    case "assistant":
      return (
        <div className="rp-full msg-wrap" style={{ position: "relative" }}>
          <CopyBtn text={row.content} />
          {row.analysis && <ResultCard r={row.analysis} />}
          <MarkdownView>{row.content}</MarkdownView>
        </div>
      );
    case "checkpoint":
      return (
        <div className="rp-full">
          <div className="rp-kv">
            <span>类型</span>
            <span>
              {(
                { initial: "首次分析", followup: "追问", reanalysis: "重新分析" } as Record<
                  string,
                  string
                >
              )[row.cp.type] || row.cp.type}
            </span>
          </div>
          {row.cp.has_analysis && (
            <div className="rp-kv">
              <span>分析</span>
              <span>本轮提交了结构化需求分析（见上方助手行）</span>
            </div>
          )}
          {row.cp.question && (
            <div className="rp-kv">
              <span>问题</span>
              <span>{row.cp.question}</span>
            </div>
          )}
          {row.cp.answer && (
            <div className="rp-kv">
              <span>回答</span>
              <span>{row.cp.answer}</span>
            </div>
          )}
        </div>
      );
    case "tool": {
      const p = safeParse(row.tc.params);
      const rv = safeParse(row.tc.result);
      return (
        <div className="rp-full">
          <div className="rp-kv">
            <span>参数</span>
            {p !== undefined ? <JsonView data={p} /> : <pre className="mono">{row.tc.params}</pre>}
          </div>
          <div className="rp-kv">
            <span>结果</span>
            {rv !== undefined ? (
              <JsonView data={rv} />
            ) : (
              <pre className="mono">{row.tc.result || "（无）"}</pre>
            )}
          </div>
        </div>
      );
    }
  }
}

function TraceRow({ row, seq }: { row: Row; seq: number }) {
  const [open, setOpen] = useState(false);
  const meta = KIND_META[row.kind];
  return (
    <>
      <tr className={`rp-row ${open ? "open" : ""}`} onClick={() => setOpen((x) => !x)}>
        <td className="mono faint" style={{ whiteSpace: "nowrap" }}>
          {row.time}
        </td>
        <td className="mono faint">{seq}</td>
        <td>
          <span className={`rp-kind ${meta.cls}`}>{meta.label}</span>
        </td>
        <td className="rp-sum">{summaryOf(row)}</td>
        <td className="faint" style={{ fontSize: 11 }}>
          {row.kind === "tool" ? (row.tc.result ? "完成" : "…") : open ? "收起 ▴" : "展开 ▸"}
        </td>
      </tr>
      {open && (
        <tr className="rp-detail">
          <td colSpan={5}>
            <Detail row={row} />
          </td>
        </tr>
      )}
    </>
  );
}

export default function Replay() {
  const { id } = useParams<{ id: string }>();
  const [data, setData] = useState<SessionDetail | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!id) return;
    sessionGet(id, currentBranch || undefined)
      .then(setData)
      .catch((e) => setError(e.message));
  }, [id, currentBranch]);

  if (error)
    return (
      <div className="page">
        <div className="error">! {error}</div>
      </div>
    );
  if (!data)
    return (
      <div className="page">
        <div className="loading">加载中…</div>
      </div>
    );

  const rows = buildRows(data);
  const toolTotal = (data.tool_calls || []).length;
  const branches = data.branches || ["main"];
  const showBranchSwitch = branches.length > 1;

  return (
    <div className="page">
      <Link to="/history" className="faint mono" style={{ fontSize: 12 }}>
        ← 返回列表
      </Link>
      <div className="page-head" style={{ marginTop: 8 }}>
        <h2>{data.session.title || data.session.session_id}</h2>
        <span className="crumb">REPLAY</span>
        <div className="page-actions">
          <Link to={`/analyze/${data.session.session_id}`} className="btn ghost sm">
            继续对话
          </Link>
        </div>
      </div>
      {showBranchSwitch && (
        <div className="branch-switch">
          <span className="faint mono">分支</span>
          {branches.map((b) => (
            <button
              key={b}
              className={`chip ${b === currentBranch ? "active" : ""}`}
              onClick={() => setCurrentBranch(b)}
              title={b === "main" ? "主线（最初始的对话路径）" : `分支 ${b}`}
            >
              {b === "main" ? "主线" : b}
            </button>
          ))}
        </div>
      )}
      <div className="page-sub">
        调用时间线 · {rows.length} 个事件 · {toolTotal} 次工具调用
      </div>

      <div className="panel flush" style={{ marginTop: 14 }}>
        <table className="rp-table">
          <thead>
            <tr>
              <th>时间</th>
              <th>#</th>
              <th>类型</th>
              <th style={{ width: "50%" }}>内容</th>
              <th>状态</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((r, i) => (
              <TraceRow key={`${r.kind}-${r.id}`} row={r} seq={i + 1} />
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
