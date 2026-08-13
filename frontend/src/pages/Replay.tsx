import { useEffect, useState } from "react";
import { useParams, Link } from "react-router-dom";
import { sessionGet } from "../api/client";
import type { SessionDetail } from "../types";

const roleColor: Record<string, string> = {
  user: "var(--blue)", assistant: "var(--green)", tool: "var(--yellow)",
};

export default function Replay() {
  const { id } = useParams<{ id: string }>();
  const [data, setData] = useState<SessionDetail | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!id) return;
    sessionGet(id).then(setData).catch((e) => setError(e.message));
  }, [id]);

  if (error) return <div className="page"><div className="error">! {error}</div></div>;
  if (!data) return <div className="page"><div className="loading">加载中…</div></div>;

  return (
    <div className="page">
      <Link to="/history" className="faint mono" style={{ fontSize: 12 }}>← 返回列表</Link>
      <div className="page-head" style={{ marginTop: 8 }}>
        <h2>{data.session.title || data.session.session_id}</h2>
        <span className="crumb">REPLAY</span>
        <div className="page-actions"><Link to={`/analyze/${data.session.session_id}`} className="btn ghost sm">继续对话</Link></div>
      </div>
      <div className="page-sub">只读回放：完整消息记录 + 工具调用轨迹。</div>

      <h3 className="mono" style={{ color: "var(--text-dim)", fontSize: 11, letterSpacing: "0.12em", textTransform: "uppercase", marginBottom: 10 }}>消息轨迹</h3>
      {data.messages?.map((m) => (
        <div className="panel" key={m.id} style={{ padding: "12px 16px", marginBottom: 8 }}>
          <div className="row" style={{ marginBottom: 6 }}>
            <span className="badge" style={{ background: `${roleColor[m.role] || "var(--text-dim)"}22`, color: roleColor[m.role] || "var(--text-dim)" }}>{m.role}</span>
            <span className="faint mono" style={{ fontSize: 11 }}>{new Date(m.created_at).toLocaleTimeString()}</span>
          </div>
          <pre className="mono" style={{ whiteSpace: "pre-wrap", fontSize: 12.5, lineHeight: 1.7, margin: 0 }}>{m.content}</pre>
        </div>
      ))}

      {data.tool_calls?.length > 0 && (
        <>
          <h3 className="mono" style={{ color: "var(--text-dim)", fontSize: 11, letterSpacing: "0.12em", textTransform: "uppercase", margin: "20px 0 10px" }}>工具调用（{data.tool_calls.length}）</h3>
          <div className="console">
            {data.tool_calls.map((tc) => (
              <div key={tc.id} className="tc-line">
                <span className="tc-tool">$ {tc.tool_name}</span> <span className="tc-params">{tc.params}</span>
                <div className="tc-result">← {tc.result}</div>
              </div>
            ))}
          </div>
        </>
      )}
    </div>
  );
}
