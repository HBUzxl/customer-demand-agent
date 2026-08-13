import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { memoryList, memorySearch, memoryDelete } from "../api/client";
import type { MemoryEntry } from "../types";

const TYPES = ["product", "threat", "compliance", "industry", "customer", "user"];
const typeLabel: Record<string, string> = {
  product: "产品", threat: "威胁", compliance: "合规", industry: "行业", customer: "客户", user: "使用者",
};

export default function MemoryList() {
  const navigate = useNavigate();
  const [type, setType] = useState("product");
  const [items, setItems] = useState<MemoryEntry[]>([]);
  const [query, setQuery] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  async function loadList() {
    setLoading(true); setError("");
    try { const r = await memoryList(type, 0, 100); setItems(r.items || []); }
    catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }
  async function handleSearch() {
    if (!query.trim()) return loadList();
    setLoading(true); setError("");
    try { const r = await memorySearch(query, type, 20); setItems(r.items || []); }
    catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }
  async function handleDelete(e: MemoryEntry) {
    if (!confirm(`归档 ${e.type}/${e.title}？`)) return;
    try { await memoryDelete(e.type, e.title, true); loadList(); }
    catch (err: any) { alert(err.message); }
  }

  useEffect(() => { loadList(); }, [type]);

  return (
    <div className="page">
      <div className="page-head"><h2>记忆库</h2><span className="crumb">KNOWLEDGE BASE</span>
        <div className="page-actions"><button className="btn sm" onClick={() => navigate("/memory/new")}>+ 新建</button></div>
      </div>
      <div className="page-sub">产品 / 威胁 / 合规 / 行业 / 客户 / 使用者 六类长期记忆。点条目进入详情编辑。</div>

      <div className="panel">
        <div className="row wrap" style={{ marginBottom: 12 }}>
          {TYPES.map((t) => (
            <button key={t} className={`btn sm ${type === t ? "" : "ghost"}`} onClick={() => setType(t)}>{typeLabel[t]}</button>
          ))}
        </div>
        <div className="row">
          <input value={query} onChange={(e) => setQuery(e.target.value)} placeholder="关键词搜索…" onKeyDown={(e) => e.key === "Enter" && handleSearch()} />
          <button className="btn ghost" onClick={handleSearch}>搜索</button>
          <button className="btn ghost" onClick={loadList}>刷新</button>
        </div>
      </div>

      {error && <div className="error">! {error}</div>}
      {loading && <div className="loading">加载中…</div>}
      {!loading && items.length === 0 && <div className="panel"><div className="empty">无条目</div></div>}
      {items.map((e, i) => (
        <div key={i} className="panel" style={{ padding: 14, cursor: "pointer" }} onClick={() => navigate(`/memory/${e.type}/${encodeURIComponent(e.title)}`)}>
          <div className="row">
            <strong>{e.title}</strong>
            {e.status && <span className={`badge ${e.status} right`}>{e.status}</span>}
          </div>
          <div className="faint" style={{ marginTop: 4, fontSize: 13 }}>{e.summary}</div>
          {e.tags && e.tags.length > 0 && <div style={{ marginTop: 4 }}>{e.tags.map((t, j) => <span key={j} className="tag">{t}</span>)}</div>}
        </div>
      ))}
    </div>
  );
}
