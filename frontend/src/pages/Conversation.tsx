import { useEffect, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { analyzeStream, chatStream, sessionGet } from "../api/client";
import type { AnalysisResult, AgentEvent } from "../types";

interface ToolTrace { tool: string; params: string; result?: string; }
interface ChatMsg {
  id: string;
  role: "user" | "assistant";
  kind?: "analyze" | "chat";
  text: string;
  reasoning?: string;
  tools?: ToolTrace[];
  analysis?: AnalysisResult;
  streaming?: boolean;
}

const uid = () => Math.random().toString(36).slice(2);

function reconstruct(messages: { role: string; content: string }[], toolCalls: { tool_name: string; params: string; result: string }[]): ChatMsg[] {
  const out: ChatMsg[] = [];
  for (const m of messages) {
    if (m.role === "user") out.push({ id: uid(), role: "user", text: m.content });
    else if (m.role === "assistant") {
      let analysis: AnalysisResult | undefined;
      try { const r = JSON.parse(m.content); if (r && r.demand_analysis) analysis = r; } catch { /* 散文 */ }
      out.push({ id: uid(), role: "assistant", kind: analysis ? "analyze" : "chat", text: analysis ? "" : m.content, analysis, tools: analysis ? toolCalls.map((t) => ({ tool: t.tool_name, params: t.params, result: t.result })) : undefined });
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
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [sessionId, setSessionId] = useState(routeSid || "");
  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!routeSid) { setMessages([]); setSessionId(""); return; }
    setSessionId(routeSid);
    sessionGet(routeSid).then((d) => setMessages(reconstruct(d.messages || [], d.tool_calls || []))).catch((e) => setError(e.message));
  }, [routeSid]);

  useEffect(() => { scrollRef.current?.scrollTo({ top: scrollRef.current.scrollHeight, behavior: "smooth" }); }, [messages]);

  function patchLast(patch: (m: ChatMsg) => void) {
    setMessages((ms) => { if (!ms.length) return ms; const copy = [...ms]; copy[copy.length - 1] = { ...copy[copy.length - 1] }; patch(copy[copy.length - 1]); return copy; });
  }

  function handleEvent(e: AgentEvent, kind: "analyze" | "chat") {
    switch (e.type) {
      case "session":
        setSessionId(e.content || "");
        if (!routeSid && e.content) navigate(`/analyze/${e.content}`, { replace: true });
        break;
      case "reasoning": patchLast((m) => { m.reasoning = (m.reasoning || "") + (e.text || ""); }); break;
      case "tool_call": patchLast((m) => { m.tools = [...(m.tools || []), { tool: e.tool || "", params: e.params || "" }]; }); break;
      case "tool_result": patchLast((m) => { if (m.tools && m.tools.length) { const t = [...m.tools]; t[t.length - 1] = { ...t[t.length - 1], result: e.result || "" }; m.tools = t; } }); break;
      case "content": if (kind === "chat") patchLast((m) => { m.text = (m.text || "") + (e.text || ""); }); break;
      case "done": patchLast((m) => { if (e.analysis) { m.analysis = e.analysis; m.text = ""; } else if (e.content) m.text = e.content; m.streaming = false; }); break;
      case "error": setError(e.error || "未知错误"); patchLast((m) => { m.streaming = false; }); break;
    }
  }

  async function send(text: string, kind: "analyze" | "chat") {
    const content = text.trim();
    if (!content || loading) return;
    setError("");
    setMessages((m) => [...m, { id: uid(), role: "user", text: content }, { id: uid(), role: "assistant", kind, text: "", streaming: true }]);
    setInput("");
    setLoading(true);
    try {
      if (kind === "analyze") await analyzeStream(content, sessionId || undefined, (e) => handleEvent(e, "analyze"));
      else await chatStream(sessionId, content, (e) => handleEvent(e, "chat"));
    } catch (err: any) { setError(err.message); patchLast((m) => { m.streaming = false; }); }
    finally { setLoading(false); }
  }

  const onKey = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); send(input, messages.length ? "chat" : "analyze"); }
  };
  const grow = (e: React.FormEvent<HTMLTextAreaElement>) => { const t = e.currentTarget; t.style.height = "auto"; t.style.height = Math.min(t.scrollHeight, 160) + "px"; };

  const composer = (
    <div className="composer">
      <textarea rows={1} value={input} onChange={(e) => setInput(e.target.value)} onKeyDown={onKey} onInput={grow}
        placeholder={messages.length ? "追问，回车发送（Shift+回车换行）…" : "粘贴客户沟通原文，回车发送…"} style={{ height: "auto" }} />
      <button className="btn primary send" onClick={() => send(input, messages.length ? "chat" : "analyze")} disabled={loading || !input.trim()} aria-label="发送">
        {loading ? <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="12" cy="12" r="9" opacity=".25" /><path d="M12 3a9 9 0 0 1 9 9" strokeLinecap="round" /></svg>
          : <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M12 19V5M5 12l7-7 7 7" /></svg>}
      </button>
    </div>
  );

  const isEmpty = messages.length === 0 && !loading;

  // ── 空状态：欢迎语 + 输入框 + 示例，垂直居中 ──
  if (isEmpty) {
    return (
      <div className="empty-state">
        <div className="empty-inner">
          <h1>需求分析助手</h1>
          <p className="empty-sub">粘贴客户沟通原文，或直接描述场景。我会理解需求、匹配长亭产品、判断可行性。</p>
          <div className="composer-wrap">{composer}</div>
          <div className="examples-grid">
            {EXAMPLES.map((ex, i) => (
              <button key={i} className="ex-card" onClick={() => send(ex.d, "analyze")}>
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
          {messages.map((m) => m.role === "user" ? (
            <div key={m.id} className="msg user"><div className="bubble">{m.text}</div></div>
          ) : <AssistantMsg key={m.id} m={m} />)}
          {error && <div className="error">! {error}</div>}
        </div>
      </div>
      <div className="chat-input">
        <div className="inner">{composer}<div className="chat-hint">{sessionId ? `会话 ${sessionId.slice(5, 17)}` : "新会话"} · 回车发送</div></div>
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
            <div className="trace-toggle" onClick={() => setOpen((x) => !x)}>
              <span className="dot" />
              {m.streaming ? "思考中…" : "思考过程"}{toolCount > 0 && ` · ${toolCount} 次工具调用`}
              <span style={{ marginLeft: 2 }}>{open ? "▾" : "▸"}</span>
            </div>
            {open && (
              <div className="trace-box">
                {m.reasoning && <div className="t-think">{m.reasoning}</div>}
                {m.tools?.map((tc, i) => (
                  <div key={i}><span className="t-tool">$ {tc.tool}</span> <span className="t-params">{tc.params}</span>
                    {tc.result && <div className="t-result" style={{ paddingLeft: 12 }}>← {tc.result.length > 200 ? tc.result.slice(0, 200) + "…" : tc.result}</div>}</div>
                ))}
              </div>
            )}
          </>
        )}
        {m.analysis ? <ResultCard r={m.analysis} />
          : m.kind === "analyze" && m.streaming ? <span className="muted">正在分析…<span className="cursor" /></span>
          : <p style={{ whiteSpace: "pre-wrap" }}>{m.text}{m.streaming && <span className="cursor" />}</p>}
      </div>
    </div>
  );
}

const feasLabel = (f: string) => ({ direct: "直接覆盖", custom: "需定制", partner: "外部整合", reject: "不建议接" } as Record<string, string>)[f] || f;
const confClass = (c: number) => (c >= 0.85 ? "g" : c >= 0.5 ? "b" : c >= 0.3 ? "y" : "r");

function ResultCard({ r }: { r: AnalysisResult }) {
  return (
    <div className="result">
      <div className="r-block"><h4>需求理解</h4><div style={{ whiteSpace: "pre-wrap" }}>{r.demand_analysis}</div></div>
      <div className="r-block"><h4>可行性判定</h4>
        <div className="feas"><span className={`badge ${r.feasibility}`}>{feasLabel(r.feasibility)}</span>{r.feasibility_detail && <span className="muted" style={{ fontSize: 13 }}>{r.feasibility_detail}</span>}</div>
      </div>
      {r.matched_products && r.matched_products.length > 0 && (
        <div className="r-block"><h4>匹配产品（{r.matched_products.length}）</h4>
          {r.matched_products.map((p, i) => (
            <div key={i} className="prod">
              <div className="row"><span className="pn">{p.name}</span><span className="pc right">{Math.round(p.confidence * 100)}%</span></div>
              <div className="bar" style={{ width: 120, marginTop: 5 }}><i className={confClass(p.confidence)} style={{ width: `${Math.round(p.confidence * 100)}%` }} /></div>
              <div className="pr">{p.reason}</div>{p.suggestion && <div className="ps">{p.suggestion}</div>}
            </div>
          ))}
        </div>
      )}
      {r.missing_info && r.missing_info.length > 0 && (
        <div className="r-block"><h4>待追问</h4><ul className="missing">{r.missing_info.map((s, i) => <li key={i}>{s}</li>)}</ul></div>
      )}
    </div>
  );
}
