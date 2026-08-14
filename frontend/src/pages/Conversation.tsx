import { useEffect, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import {
  submitMessage,
  subscribeStream,
  cancelRun,
  truncateMessages,
  sessionRunning,
  sessionGet,
} from "../api/client";
import type { AnalysisResult, AgentEvent } from "../types";
import MarkdownView from "../components/MarkdownView";
import ResultCard from "../components/ResultCard";
import ToolTimeline from "../components/ToolTrace";
import type { ToolTrace } from "../components/ToolTrace";

interface ChatMsg {
  id: string; // 稳定渲染 key（"m{seq}" 或 uid()）
  seq?: number; // 持久化序号（编辑重发截断用，本地新消息无）
  role: "user" | "assistant";
  text: string;
  reasoning?: string;
  tools?: ToolTrace[];
  analysis?: AnalysisResult;
  streaming?: boolean;
}

const uid = () => Math.random().toString(36).slice(2);

// ANALYSIS_TOOL 与后端 agent.ToolAnalysisSubmit 对齐（ADR-013）。
const ANALYSIS_TOOL = "analysis_submit";

// parseSubmitParams 从 analysis_submit 的参数还原结构化分析（回放用）。
function parseSubmitParams(params: string): AnalysisResult | undefined {
  try {
    const r = JSON.parse(params);
    if (r && r.demand_analysis) return r as AnalysisResult;
  } catch {
    /* 非 JSON 参数 */
  }
  return undefined;
}

// reconstruct 从会话历史重建消息流（新系统：assistant content 为自然语言，
// 工具调用按 message_id 精确归属到对应 assistant 消息（服务端持久化时写入），
// analysis 从该轮的 analysis_submit params 还原；不做老数据兼容——用户裁决）。
function reconstruct(
  messages: { id: number; role: string; content: string; seq: number }[],
  toolCalls: { message_id: number; tool_name: string; params: string; result: string }[],
): ChatMsg[] {
  const byMessage = new Map<number, ToolTrace[]>();
  for (const c of toolCalls) {
    const list = byMessage.get(c.message_id) || [];
    list.push({ tool: c.tool_name, params: c.params, result: c.result });
    byMessage.set(c.message_id, list);
  }
  const out: ChatMsg[] = [];
  for (const m of messages) {
    if (m.role === "user") {
      out.push({ id: `m${m.seq}`, seq: m.seq, role: "user", text: m.content });
    } else if (m.role === "assistant") {
      const msg: ChatMsg = { id: `m${m.seq}`, seq: m.seq, role: "assistant", text: m.content };
      // system 消息间的 assistant：工具按 message_id（数据库 id）归属
      const mine = byMessage.get(m.id);
      if (mine && mine.length > 0) {
        msg.tools = mine;
        for (let j = mine.length - 1; j >= 0; j--) {
          if (mine[j].tool === ANALYSIS_TOOL) {
            msg.analysis = parseSubmitParams(mine[j].params);
            break;
          }
        }
      }
      out.push(msg);
    }
  }
  return out;
}

const EXAMPLES = [
  { t: "电商 · CC 攻击", d: "大促网站被 CC 打慢，过等保二级" },
  { t: "政务 · 挂马篡改", d: "官网被挂马，要过等保三级" },
  { t: "金融 · API 滥用", d: "接口被恶意调用，数据泄露风险" },
];

export default function Conversation() {
  const { sessionId: routeSid } = useParams<{ sessionId: string }>();
  const navigate = useNavigate();
  const [messages, setMessages] = useState<ChatMsg[]>([]);
  const [input, setInput] = useState("");
  const [loading, setLoading] = useState(false); // 本视图内订阅进行中
  const [error, setError] = useState("");
  const [sessionId, setSessionId] = useState(routeSid || "");
  const [customer, setCustomer] = useState(""); // 会话关联客户（身份行注入 ADR-016 L2）
  const [editingCustomer, setEditingCustomer] = useState(false);
  const [activeRun, setActiveRun] = useState<{ sid: string; rid: string } | null>(null); // F0：本视图发起/恢复订阅的 Run
  const scrollRef = useRef<HTMLDivElement>(null);
  const unsubRef = useRef<(() => void) | null>(null);
  const msgsRef = useRef<ChatMsg[]>([]);
  msgsRef.current = messages;

  useEffect(() => {
    if (!routeSid) {
      stopSubscription();
      setMessages([]);
      setSessionId("");
      setCustomer("");
      setActiveRun(null);
      return;
    }
    // 流式订阅进行中不做历史重载（切回自己会话）；切换到别的会话先停旧订阅
    let cancelled = false;
    (async () => {
      try {
        const d = await sessionGet(routeSid);
        if (cancelled) return;
        setMessages(reconstruct(d.messages || [], d.tool_calls || []));
        setCustomer(d.session.customer || "");
        setSessionId(routeSid);
        // F0 切回恢复：会话还在跑 → 重建流式中的 assistant 消息并续订
        const running = await sessionRunning(routeSid);
        if (cancelled) return;
        if (running) {
          setActiveRun(null); // rid 由 running 接口给（stop 用兜底路径）
          const r2 = await fetch(`/api/sessions/${routeSid}/running`)
            .then((x) => x.json())
            .catch(() => null);
          if (r2?.run_id) setActiveRun({ sid: routeSid, rid: r2.run_id });
          attachStream(routeSid, { resume: true });
        }
      } catch (e) {
        if (!cancelled) setError(e instanceof Error ? e.message : String(e));
      }
    })();
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [routeSid]);

  // 组件卸载：只取消订阅（Run 在服务端继续，F0）
  useEffect(() => () => unsubRef.current?.(), []);

  useEffect(() => {
    scrollRef.current?.scrollTo({ top: scrollRef.current.scrollHeight, behavior: "smooth" });
  }, [messages]);

  function patchLast(patch: (m: ChatMsg) => void) {
    setMessages((ms) => {
      if (!ms.length) return ms;
      const copy = [...ms];
      copy[copy.length - 1] = { ...copy[copy.length - 1] };
      patch(copy[copy.length - 1]);
      return copy;
    });
  }

  function handleEvent(e: AgentEvent) {
    switch (e.type) {
      case "session":
        setSessionId(e.content || "");
        if (!routeSid && e.content) navigate(`/analyze/${e.content}`, { replace: true });
        break;
      case "reasoning":
        patchLast((m) => {
          m.reasoning = (m.reasoning || "") + (e.text || "");
        });
        break;
      case "tool_call":
        patchLast((m) => {
          m.tools = [...(m.tools || []), { tool: e.tool || "", params: e.params || "" }];
        });
        break;
      case "tool_result":
        patchLast((m) => {
          if (m.tools && m.tools.length) {
            const t = [...m.tools];
            t[t.length - 1] = { ...t[t.length - 1], result: e.result || "" };
            m.tools = t;
            // ResultCard 即时渲染（用户裁决：过程透明优先）：
            // analysis_submit 的结果到达时立刻解析其 params 展示卡片。
            if (e.tool === ANALYSIS_TOOL) {
              const sub = t[t.length - 1];
              const analysis = parseSubmitParams(sub.params);
              if (analysis) m.analysis = analysis;
            }
          }
        });
        break;
      case "content":
        // content 始终累积（自主模式下 Agent 的自然语言说明全程可见）。
        patchLast((m) => {
          m.text = (m.text || "") + (e.text || "");
        });
        break;
      case "done":
        patchLast((m) => {
          // done.content 是最终答案（服务端已累积全程 content）。
          if (e.content) m.text = e.content;
          // analysis 以提交过的为准（tool_result 已即时渲染，此处最终定稿）。
          if (e.analysis) m.analysis = e.analysis;
          m.streaming = false;
        });
        stopSubscription();
        setActiveRun(null);
        break;
      case "error":
        setError(e.error || "未知错误");
        patchLast((m) => {
          m.streaming = false;
        });
        stopSubscription();
        setActiveRun(null);
        break;
    }
  }

  // ── F0 订阅模式：attach 到会话的 Run（事件 → 消息流）──
  // resume=true 时消息流里已有历史重建，事件只 patch 流式中的最后一条。
  function attachStream(sid: string, opts?: { resume?: boolean }) {
    unsubRef.current?.();
    setLoading(true);
    if (!opts?.resume) {
      setMessages((m) => [...m, { id: uid(), role: "assistant", text: "", streaming: true }]);
    } else if (!msgsRef.current.length || !msgsRef.current[msgsRef.current.length - 1].streaming) {
      setMessages((m) => [...m, { id: uid(), role: "assistant", text: "", streaming: true }]);
    }
    unsubRef.current = subscribeStream(sid, 0, handleEvent);
  }

  function stopSubscription() {
    unsubRef.current?.();
    unsubRef.current = null;
    setLoading(false);
  }

  // F1：显式停止（cancel Run + 结束本视图订阅；assistant 留中断态）
  async function stopRun() {
    if (!activeRun) return;
    try {
      await cancelRun(activeRun.sid, activeRun.rid);
    } catch {
      /* Run 可能刚自然结束 */
    }
    stopSubscription();
    patchLast((m) => {
      m.streaming = false;
      if (!m.text && !m.analysis) m.text = "（已停止）";
    });
    setActiveRun(null);
  }

  async function send(text: string) {
    const content = text.trim();
    if (!content || loading) return;
    setError("");
    setMessages((m) => [...m, { id: uid(), role: "user", text: content }]);
    setInput("");
    setLoading(true);
    try {
      const handle = await submitMessage(content, sessionId || undefined, customer || undefined);
      setSessionId(handle.session_id);
      setActiveRun({ sid: handle.session_id, rid: handle.run_id });
      if (!routeSid && handle.session_id)
        navigate(`/analyze/${handle.session_id}`, { replace: true });
      attachStream(handle.session_id);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      setLoading(false);
    }
  }

  // F1：编辑重发——截断该 user 消息及其之后，把原文填回输入框
  async function editResend(msg: ChatMsg) {
    if (loading) return;
    try {
      if (!msg.seq) {
        // 本地新消息（无持久化 seq）：直接本地移除
        setMessages((m) => m.filter((x) => x.id !== msg.id));
        setInput(msg.text);
        return;
      }
      await truncateMessages(sessionId, msg.seq); // 真实持久化 seq（≠数据库 id）
      // 本地重放：删掉该条及之后
      const idx = messages.findIndex((m) => m.id === msg.id);
      if (idx >= 0) setMessages((m) => m.slice(0, idx));
      setInput(msg.text);
      setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  const onKey = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      send(input);
    }
  };
  const grow = (e: React.FormEvent<HTMLTextAreaElement>) => {
    const t = e.currentTarget;
    t.style.height = "auto";
    t.style.height = Math.min(t.scrollHeight, 160) + "px";
  };

  const composer = (
    <div className="composer">
      <textarea
        rows={1}
        value={input}
        onChange={(e) => setInput(e.target.value)}
        onKeyDown={onKey}
        onInput={grow}
        placeholder="输入消息，回车发送（Shift+回车换行）…"
        style={{ height: "auto" }}
      />
      <button
        className={`btn send${loading ? " stop" : " primary"}`}
        onClick={() => (loading ? stopRun() : send(input))}
        disabled={!loading && !input.trim()}
        aria-label={loading ? "停止" : "发送"}
        title={loading ? "停止生成" : "发送"}
      >
        {loading ? (
          <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor" aria-hidden>
            <rect x="6" y="6" width="12" height="12" rx="2" />
          </svg>
        ) : (
          <svg
            width="18"
            height="18"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
          >
            <path d="M12 19V5M5 12l7-7 7 7" />
          </svg>
        )}
      </button>
    </div>
  );

  const isEmpty = messages.length === 0 && !loading;

  // 空状态也提供客户名关联（P0：首轮分析就要带客户身份）
  const customerChip = editingCustomer ? (
    <input
      className="cust-input"
      autoFocus
      value={customer}
      placeholder="客户名（如：某跨境电商）"
      onChange={(e) => setCustomer(e.target.value)}
      onBlur={() => setEditingCustomer(false)}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === "Escape") setEditingCustomer(false);
      }}
      title="关联客户后，Agent 每轮知道在与谁对话（画像细节自动检索）"
    />
  ) : (
    <button
      className="cust-chip"
      onClick={() => setEditingCustomer(true)}
      title="设置会话关联的客户"
    >
      {customer ? `客户：${customer}` : "+ 关联客户"}
    </button>
  );

  // ── 空状态：欢迎语 + 输入框 + 示例，垂直居中 ──
  if (isEmpty) {
    return (
      <div className="empty-state">
        <div className="empty-inner">
          <h1>需求分析助手</h1>
          <p className="empty-sub">
            粘贴客户沟通原文，或直接描述场景。我会理解需求、匹配长亭产品、判断可行性——也可以直接跟我聊。
          </p>
          <div className="composer-wrap">{composer}</div>
          <div className="empty-aux">
            {customerChip}
            <span className="empty-hint">先关联客户，首轮分析即可结合其画像</span>
          </div>
          <div className="examples-grid">
            {EXAMPLES.map((ex, i) => (
              <button key={i} className="ex-card" onClick={() => send(ex.d)}>
                <div className="ex-t">{ex.t}</div>
                <div className="ex-d">{ex.d}</div>
              </button>
            ))}
          </div>
        </div>
      </div>
    );
  }

  // ── 对话中：消息流 + 底部输入 ──
  return (
    <div className="chat">
      <div className="chat-scroll" ref={scrollRef}>
        <div className="chat-inner">
          {messages.map((m) =>
            m.role === "user" ? (
              <div key={m.id} className="msg user">
                <div className="bubble">{m.text}</div>
                {!loading && (
                  <button
                    className="edit-resend"
                    onClick={() => editResend(m)}
                    title="编辑后重发（删除此消息及之后的对话）"
                  >
                    ✎ 编辑重发
                  </button>
                )}
              </div>
            ) : (
              <AssistantMsg key={m.id} m={m} />
            ),
          )}
          {error && <div className="error">! {error}</div>}
        </div>
      </div>
      <div className="chat-input">
        <div className="inner">
          {composer}
          <div className="chat-hint">
            {sessionId ? `会话 ${sessionId.slice(5, 17)}` : "新会话"} · 回车发送
            {customerChip}
          </div>
        </div>
      </div>
    </div>
  );
}

function AssistantMsg({ m }: { m: ChatMsg }) {
  const [open, setOpen] = useState(false);
  const hasTrace = !!(m.reasoning || (m.tools && m.tools.length));
  const toolCount = m.tools?.length || 0;
  return (
    <div className="msg assistant">
      <div className="avatar">需求分析助手</div>
      <div className="body">
        {hasTrace && (
          <>
            <div
              className={`trace-toggle${m.streaming ? " thinking" : ""}`}
              onClick={() => setOpen((x) => !x)}
            >
              <span className="dot" />
              {m.streaming ? "思考中…" : "思考过程"}
              {toolCount > 0 && ` · ${toolCount} 次工具调用`}
              <span style={{ marginLeft: 2 }}>{open ? "▾" : "▸"}</span>
            </div>
            {(open || m.streaming) && (
              <div className="trace-box">
                {m.reasoning && (
                  <div className="t-think">
                    {m.reasoning}
                    {m.streaming && <span className="cursor" />}
                  </div>
                )}
                {m.tools && m.tools.length > 0 && <ToolTimeline tools={m.tools} />}
              </div>
            )}
          </>
        )}
        {m.analysis && <ResultCard r={m.analysis} />}
        {m.text ? (
          <MarkdownView>{m.text}</MarkdownView>
        ) : m.streaming ? (
          <span className="muted">
            正在回应…
            <span className="cursor" />
          </span>
        ) : null}
      </div>
    </div>
  );
}
