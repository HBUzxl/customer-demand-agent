import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { sessionsList, sessionDelete } from "../api/client";
import type { SessionListItem } from "../types";

export default function History() {
  const [items, setItems] = useState<SessionListItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  async function load() {
    setLoading(true);
    try {
      const r = await sessionsList(50, 0);
      setItems(r.items || []);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }
  useEffect(() => {
    load();
  }, []);

  async function handleDelete(id: string) {
    if (!confirm("删除该会话及其全部记录？")) return;
    try {
      await sessionDelete(id);
      load();
    } catch (e) {
      alert(e instanceof Error ? e.message : String(e));
    }
  }

  return (
    <div className="page">
      <div className="page-head">
        <h2>对话历史</h2>
        <span className="crumb">SESSION LOG</span>
        <div className="page-actions">
          <Link to="/analyze" className="btn sm">
            + 新对话
          </Link>
        </div>
      </div>
      <div className="page-sub">列出全部历史对话。可继续追问或回放完整轨迹。</div>

      {loading && <div className="loading">加载中…</div>}
      {error && <div className="error">! {error}</div>}
      {!loading && items.length === 0 && (
        <div className="panel">
          <div className="empty">暂无对话记录</div>
        </div>
      )}
      {items.length > 0 && (
        <div className="panel flush">
          <table>
            <thead>
              <tr>
                <th>标题</th>
                <th>客户</th>
                <th>更新</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {items.map((s) => (
                <tr key={s.session_id}>
                  <td>
                    <Link to={`/analyze/${s.session_id}`}>{s.title || s.session_id}</Link>
                  </td>
                  <td className="faint">{s.customer || "—"}</td>
                  <td className="faint mono" style={{ fontSize: 12 }}>
                    {new Date(s.updated_at).toLocaleString()}
                  </td>
                  <td>
                    <div className="row" style={{ justifyContent: "flex-end" }}>
                      <Link to={`/analyze/${s.session_id}`} className="btn ghost sm">
                        继续
                      </Link>
                      <Link to={`/history/${s.session_id}`} className="btn ghost sm">
                        回放
                      </Link>
                      <button className="btn ghost sm" onClick={() => handleDelete(s.session_id)}>
                        删除
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
