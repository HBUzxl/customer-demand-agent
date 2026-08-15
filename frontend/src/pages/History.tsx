import { useEffect, useState } from "react";
import ConfirmDialog from "../components/ConfirmDialog";
import { Link } from "react-router-dom";
import { sessionsList, sessionDelete } from "../api/client";
import type { SessionListItem } from "../types";

export default function History() {
  const [items, setItems] = useState<SessionListItem[]>([]);
  const [confirmTarget, setConfirmTarget] = useState<string | null>(null);
  const [err, setErr] = useState("");
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [batchOpen, setBatchOpen] = useState(false);
  const [batchBusy, setBatchBusy] = useState(false);
  const allSelected = items.length > 0 && selected.size === items.length;
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
    setConfirmTarget(id);
  }
  function toggle(id: string) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }
  function toggleAll() {
    setSelected(allSelected ? new Set() : new Set(items.map((i) => i.session_id)));
  }
  async function doBatchDelete() {
    setBatchBusy(true);
    const ids = [...selected];
    const results = await Promise.allSettled(ids.map((id) => sessionDelete(id)));
    const okN = results.filter((r) => r.status === "fulfilled").length;
    const failN = ids.length - okN;
    setBatchOpen(false);
    setSelected(new Set());
    if (failN === 0) setErr("");
    else {
      const firstErr = results.find((r) => r.status === "rejected") as
        PromiseRejectedResult | undefined;
      setErr(
        `批量删除：成功 ${okN}，失败 ${failN}（${firstErr?.reason instanceof Error ? firstErr.reason.message : String(firstErr?.reason)}）`,
      );
    }
    setBatchBusy(false);
    load();
  }

  async function doDelete() {
    if (!confirmTarget) return;
    try {
      await sessionDelete(confirmTarget);
      load();
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setConfirmTarget(null);
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

      {selected.size > 0 && (
        <div className="batch-bar">
          <span>已选 {selected.size} 项</span>
          <button className="btn sm danger" onClick={() => setBatchOpen(true)}>
            批量删除
          </button>
          <button className="btn ghost sm" onClick={() => setSelected(new Set())}>
            取消选择
          </button>
        </div>
      )}

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
                <th style={{ width: 32 }}>
                  <input
                    type="checkbox"
                    checked={allSelected}
                    onChange={toggleAll}
                    aria-label="全选"
                  />
                </th>
                <th>标题</th>
                <th>客户</th>
                <th>更新</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {items.map((s) => (
                <tr key={s.session_id} className={selected.has(s.session_id) ? "sel" : ""}>
                  <td>
                    <input
                      type="checkbox"
                      checked={selected.has(s.session_id)}
                      onChange={() => toggle(s.session_id)}
                      aria-label={`选择 ${s.title || s.session_id}`}
                    />
                  </td>
                  <td>
                    <Link to={`/analyze/${s.session_id}`}>{s.title || s.session_id}</Link>
                  </td>
                  <td className="faint cust-cell" title={s.customer}>
                    {s.customer || "—"}
                  </td>
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
      {err && (
        <div className="err-banner" onClick={() => setErr("")}>
          {err}（点击关闭）
        </div>
      )}
      <ConfirmDialog
        open={batchOpen}
        title={`删除 ${selected.size} 个会话？`}
        body="删除后不可恢复。"
        confirmText="全部删除"
        busy={batchBusy}
        onConfirm={doBatchDelete}
        onCancel={() => setBatchOpen(false)}
      />
      <ConfirmDialog
        open={!!confirmTarget}
        title="删除该会话及其全部记录？"
        body="删除后不可恢复。"
        confirmText="删除"
        onConfirm={doDelete}
        onCancel={() => setConfirmTarget(null)}
      />
    </div>
  );
}
