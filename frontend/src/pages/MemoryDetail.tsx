import { useEffect, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { memoryGet, memoryUpsert, memoryDelete } from "../api/client";
import type { MemoryEntry } from "../types";
import MemoryForm, { toValues, fromValues, type MemoryValues } from "../components/MemoryForm";

// /memory/:type/:title —— 查看 / 编辑单条记忆
export default function MemoryDetail() {
  const { type = "", title = "" } = useParams<{ type: string; title: string }>();
  const decodedTitle = decodeURIComponent(title);
  const navigate = useNavigate();
  const [entry, setEntry] = useState<MemoryEntry | null>(null);
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  async function load() {
    setLoading(true); setError("");
    try { const e = await memoryGet(type, decodedTitle); setEntry(e); }
    catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }
  useEffect(() => { load(); }, [type, decodedTitle]);

  async function onSubmit(v: MemoryValues) {
    setSubmitting(true); setError("");
    try {
      const e = fromValues(v);
      await memoryUpsert(e);
      setEditing(false);
      if (e.title !== decodedTitle || e.type !== type) {
        navigate(`/memory/${e.type}/${encodeURIComponent(e.title)}`, { replace: true });
      } else { load(); }
    } catch (err: any) { setError(err.message); }
    finally { setSubmitting(false); }
  }

  async function handleArchive() {
    if (!entry) return;
    if (!confirm(`归档 ${entry.type}/${entry.title}？`)) return;
    try { await memoryDelete(entry.type, entry.title, true); navigate("/memory"); }
    catch (e: any) { alert(e.message); }
  }

  if (loading) return <div className="page"><div className="loading">加载中…</div></div>;
  if (error && !entry) return <div className="page"><div className="error">! {error}</div></div>;
  if (!entry) return <div className="page"><div className="empty">条目不存在</div></div>;

  return (
    <div className="page">
      <div className="page-head">
        <h2>{entry.title}</h2>
        <span className="crumb">{entry.type.toUpperCase()}</span>
        <span className={`badge ${entry.status || "verified"}`}>{entry.status || "verified"}</span>
        <div className="page-actions">
          <button className="btn ghost sm" onClick={() => setEditing((x) => !x)}>{editing ? "取消编辑" : "编辑"}</button>
          <button className="btn ghost sm" onClick={handleArchive}>归档</button>
        </div>
      </div>

      {error && <div className="error">! {error}</div>}

      {editing ? (
        <MemoryForm initial={toValues(entry)} submitting={submitting} onSubmit={onSubmit} onCancel={() => setEditing(false)} />
      ) : (
        <>
          {entry.aliases && entry.aliases.length > 0 && (
            <div className="panel"><h3><span className="dot" />别名</h3>{entry.aliases.map((a, i) => <span key={i} className="tag">{a}</span>)}</div>
          )}
          {entry.tags && entry.tags.length > 0 && (
            <div className="panel"><h3><span className="dot" />标签</h3>{entry.tags.map((t, i) => <span key={i} className="tag">{t}</span>)}</div>
          )}
          <div className="panel">
            <h3><span className="dot b" />正文</h3>
            <pre className="mono" style={{ whiteSpace: "pre-wrap", fontSize: 13, lineHeight: 1.7, margin: 0 }}>{entry.content}</pre>
          </div>
        </>
      )}
    </div>
  );
}
