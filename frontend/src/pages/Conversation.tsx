import { useEffect, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import {
  ApiError,
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
  askUser?: {
    question: string;
    options: { label: string; value?: string; description?: string }[];
  }; // F3 待答选项
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
  const [sessionCustomer, setSessionCustomer] = useState("");
  const [input, setInput] = useState(() => {
    // G2 草稿：进入时恢复该会话的草稿（新建会话无草稿）
    return "";
  });
  const [loading, setLoading] = useState(false); // 本视图内订阅进行中
  const [error, setError] = useState("");
  const [sessionId, setSessionId] = useState(routeSid || "");
  const [activeRun, setActiveRun] = useState<{ sid: string; rid: string } | null>(null); // F0：本视图发起/恢复订阅的 Run
  const scrollRef = useRef<HTMLDivElement>(null);
  const unsubRef = useRef<(() => void) | null>(null);
  const lastSeqRef = useRef<Map<string, number>>(new Map()); // SSE 游标（按会话隔离）
  const subSessionRef = useRef<string>(""); // 当前订阅的会话（防串台）
  const activeRunRef = useRef<string>(""); // 当前关注的 Run（事件过滤）
  const msgsRef = useRef<ChatMsg[]>([]);
  msgsRef.current = messages;

  useEffect(() => {
    // 路由变化第一件事：退旧订阅（防切换窗口旧会话事件写入新视图）
    if (subSessionRef.current !== routeSid) {
      unsubRef.current?.();
      unsubRef.current = null;
      activeRunRef.current = "";
      setLoading(false);
    }
    if (!routeSid) {
      stopSubscription();
      setMessages([]);
      setSessionId("");
      setSessionCustomer("");
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
        setSessionId(routeSid);
        setSessionCustomer(d.session?.customer || "");
        // F0 切回恢复（竞态安全）：
        // a) running=true 且二次确认仍在跑 → 续订（带 rid 过滤）；
        // b) 查询窗口内 Run 完成（true→false）→ 重拉详情补最终答案；
        // c) running=false 但快照缺 assistant（快照先于完成拉取）→ 重拉补全。
        const running = await sessionRunning(routeSid);
        if (cancelled) return;
        if (running) {
          const r2 = await fetch(`/api/sessions/${routeSid}/running`)
            .then((x) => x.json())
            .catch(() => null);
          if (cancelled) return;
          if (r2?.running && r2?.run_id) {
            setActiveRun({ sid: routeSid, rid: r2.run_id });
            attachStream(routeSid, r2.run_id, { resume: true });
          } else {
            const d2 = await sessionGet(routeSid);
            if (cancelled) return;
            setMessages(reconstruct(d2.messages || [], d2.tool_calls || []));
          }
        } else {
          const sawUser = (d.messages || []).filter(
            (m: { role: string }) => m.role === "user",
          ).length;
          const sawAsst = (d.messages || []).filter(
            (m: { role: string }) => m.role === "assistant",
          ).length;
          if (sawUser > sawAsst) {
            const d2 = await sessionGet(routeSid);
            if (cancelled) return;
            setMessages(reconstruct(d2.messages || [], d2.tool_calls || []));
          }
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
    if (loading) {
      // 流式中：跟随滚底
      scrollRef.current?.scrollTo({ top: scrollRef.current.scrollHeight, behavior: "smooth" });
      return;
    }
    // G8 静止态：恢复该会话的记忆位置（无记忆则滚底）
    const memo = scrollMemoRef.current.get(sessionId);
    if (memo !== undefined && messages.length < 30) {
      scrollRef.current?.scrollTo({ top: memo });
    } else {
      scrollRef.current?.scrollTo({ top: scrollRef.current.scrollHeight });
    }
  }, [messages, loading]);

  // G2 草稿持久化：输入变化即存（key 含会话）；发送后清
  useEffect(() => {
    const key = sessionId ? `draft-${sessionId}` : "draft-new";
    if (input) localStorage.setItem(key, input);
    else localStorage.removeItem(key);
  }, [input, sessionId]);
  // 进入会话时恢复草稿
  useEffect(() => {
    const key = sessionId ? `draft-${sessionId}` : "draft-new";
    const saved = localStorage.getItem(key);
    if (saved) setInput(saved);
    return () => {
      localStorage.setItem(key, input);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sessionId]);

  // G8 滚动位置：离开会话前记录，回来恢复（新消息少于 3 条时才恢复，避免打断流式）
  const scrollMemoRef = useRef<Map<string, number>>(new Map());

  function patchLast(patch: (m: ChatMsg) => void) {
    setMessages((ms) => {
      if (!ms.length) return ms;
      const copy = [...ms];
      copy[copy.length - 1] = { ...copy[copy.length - 1] };
      patch(copy[copy.length - 1]);
      return copy;
    });
  }

  function handleEvent(e: AgentEvent, run: string) {
    // 多轮/切回防串台：只处理当前订阅 Run 的事件——replay 里历史 Run 的
    // 事件（含旧 done）带自身 x-run 归属，在这里被丢弃。
    if (activeRunRef.current && run && run !== activeRunRef.current) return;
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
      case "ask_user":
        patchLast((m) => {
          m.askUser = { question: e.question || "", options: e.options || [] };
        });
        break;
      case "done":
        patchLast((m) => {
          if (m.askUser) m.streaming = false; // 有待答问题时保持卡片展示
          // done.content 是最终答案（服务端已累积全程 content）。
          if (e.content) m.text = e.content;
          // analysis 以提交过的为准（tool_result 已即时渲染，此处最终定稿）。
          if (e.analysis) m.analysis = e.analysis;
          m.streaming = false;
        });
        stopSubscription();
        setActiveRun(null);
        activeRunRef.current = "";
        break;
      case "error":
        setError(e.error || "未知错误");
        patchLast((m) => {
          m.streaming = false;
        });
        stopSubscription();
        setActiveRun(null);
        activeRunRef.current = "";
        break;
    }
  }

  // ── F0 订阅模式：attach 到会话的 Run（事件 → 消息流）──
  // resume=true 时消息流里已有历史重建，事件只 patch 流式中的最后一条。
  function attachStream(sid: string, runID: string, opts?: { resume?: boolean }) {
    unsubRef.current?.();
    setLoading(true);
    subSessionRef.current = sid;
    activeRunRef.current = runID;
    if (!opts?.resume) {
      setMessages((m) => [...m, { id: uid(), role: "assistant", text: "", streaming: true }]);
    } else if (!msgsRef.current.length || !msgsRef.current[msgsRef.current.length - 1].streaming) {
      setMessages((m) => [...m, { id: uid(), role: "assistant", text: "", streaming: true }]);
    }
    // 游标按会话隔离；新 Run 用该会话已见最大 seq（上一轮 done 已记）——
    // 增量订阅天然跳过历史 Run 事件（run_id 过滤双保险见 handleEvent）
    const since = lastSeqRef.current.get(sid) || 0;
    unsubRef.current = subscribeStream(
      sid,
      since,
      handleEvent,
      (seq) => lastSeqRef.current.set(sid, seq),
      // 订阅故障（非主动退订）：结束 loading 并提示——Run 在服务端继续，
      // 切回会话可经恢复链路（running 查询+replay）重新接上。
      (err) => {
        setLoading(false);
        setError("事件流中断：" + err.message + "（任务在服务端继续，切回本会话可恢复）");
      },
    );
  }

  function stopSubscription() {
    unsubRef.current?.();
    unsubRef.current = null;
    setLoading(false);
  }

  // F1：显式停止（cancel Run + 结束本视图订阅；assistant 留中断态）。
  // 失败语义：404=Run 已结束（按停止收尾）；网络/5xx=停止失败，Run 可能
  // 仍在跑——保留订阅与运行态，仅提示，不谎报"已停止"。
  async function stopRun() {
    if (!activeRun) return;
    try {
      await cancelRun(activeRun.sid, activeRun.rid);
    } catch (err) {
      const status = err instanceof ApiError ? err.status : 0;
      if (status !== 404) {
        setError(
          "停止请求失败（任务可能仍在运行）：" + (err instanceof Error ? err.message : String(err)),
        );
        return; // 不退订、不标已停止——与真实运行态一致
      }
      // 404：Run 已自然结束，按完成收尾
    }
    stopSubscription();
    patchLast((m) => {
      m.streaming = false;
      // 中断态恒可辨：无内容时占位；已有部分内容时追加标记
      if (!m.text && !m.analysis) m.text = "（已停止）";
      else if (!/（已停止）$/.test(m.text)) m.text = m.text + "\n\n（已停止）";
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
      const handle = await submitMessage(content, sessionId || undefined);
      setSessionId(handle.session_id);
      setActiveRun({ sid: handle.session_id, rid: handle.run_id });
      if (!routeSid && handle.session_id)
        navigate(`/analyze/${handle.session_id}`, { replace: true });
      attachStream(handle.session_id, handle.run_id);
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

  const custChip = sessionCustomer ? (
    <span className="cust-ro" title={sessionCustomer}>
      客户：{sessionCustomer}
    </span>
  ) : null;

  function onScroll() {
    if (scrollRef.current && sessionId) {
      scrollMemoRef.current.set(sessionId, scrollRef.current.scrollTop);
    }
  }

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
            {custChip}
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
      <div className="chat-scroll" ref={scrollRef} onScroll={onScroll}>
        <div className="chat-inner">
          {messages.map((m) =>
            m.role === "user" ? (
              <div key={m.id} className="msg user msg-wrap" style={{ position: "relative" }}>
                <div className="bubble">{m.text}</div>
                <CopyBtn text={m.text} />
                {!loading && (
                  <button className="edit-resend" onClick={() => editResend(m)} title="编辑后重发">
                    ✎ 编辑重发
                  </button>
                )}
              </div>
            ) : (
              <AssistantMsg key={m.id} m={m} onSend={send} />
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
            {custChip}
          </div>
        </div>
      </div>
    </div>
  );
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

function AssistantMsg({ m, onSend }: { m: ChatMsg; onSend: (t: string) => void }) {
  const [open, setOpen] = useState(false);
  const hasTrace = !!(m.reasoning || (m.tools && m.tools.length));
  const toolCount = m.tools?.length || 0;
  return (
    <div className="msg assistant msg-wrap" style={{ position: "relative" }}>
      <CopyBtn text={m.text} />
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
        {m.askUser && (
          <div className="ask-card">
            <div className="ask-q">❓ {m.askUser.question}</div>
            <div className="ask-opts">
              {m.askUser.options.map((o, i) => (
                <button
                  key={i}
                  className="ask-opt"
                  onClick={() => onSend(o.value || o.label)}
                  title={o.description}
                >
                  {o.label}
                </button>
              ))}
            </div>
          </div>
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
