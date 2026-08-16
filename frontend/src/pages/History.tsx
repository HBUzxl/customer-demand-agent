import { useEffect, useState } from "react";
import ConfirmDialog from "../components/ConfirmDialog";
import { Link } from "react-router-dom";
import { sessionsList, sessionDelete, sessionsSearch } from "../api/client";
import type { SessionListItem } from "../types";

export default function History() {
  const [items, setItems] = useState<SessionListItem[]>([]);
  const [confirmTarget, setConfirmTarget] = useState<string | null>(null);
  const [err, setErr] = useState("");
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [query, setQuery] = useState("");
  const [searching, setSearching] = useState(false);
  const [hits, setHits] = useState<SessionListItem[] | null>(null);
  const [batchOpen, setBatchOpen] = useState(false);
  const [batchBusy, setBatchBusy] = useState(false);
  const view = hits ?? items;
  const allSelected = view.length > 0 && selected.size === view.length;
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

  // 内容搜索（标题+消息正文，防抖 300ms；空=回退全量列表）
  useEffect(() => {
    const q = query.trim();
    if (!q) {
      setHits(null);
      setSearching(false);
      return;
    }
    setSearching(true);
    const t = setTimeout(async () => {
      const r = await sessionsSearch(q, 30);
      setHits(r.map((x) => ({ ...x, created_at: x.updated_at })));
    }, 300);
    return () => clearTimeout(t);
  }, [query]);

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
    setSelected(allSelected ? new Set() : new Set(view.map((i) => i.session_id)));
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
      <input
        className="tab-search"
        placeholder="搜索标题与对话内容…"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        style={{ marginBottom: 12 }}
      />

      {/* 批量操作条常驻占位：无选中时隐藏，避免出现/消失导致表格跳动 */}
      <div className={`batch-bar${selected.size === 0 ? " empty" : ""}`}>
        <span>已选 {selected.size} 项</span>
        <button className="btn sm danger" onClick={() => setBatchOpen(true)}>
          批量删除
        </button>
        <button className="btn ghost sm" onClick={() => setSelected(new Set())}>
          取消选择
        </button>
      </div>

      {loading && <div className="loading">加载中…</div>}
      {error && <div className="error">! {error}</div>}
      {!loading && !searching && items.length === 0 && (
        <div className="panel">
          <div className="empty">暂无对话记录</div>
        </div>
      )}
      {(hits ?? items).length > 0 && (
        <div className="panel flush">
          <table>
            <thead>
              <tr>
                <th style={{ width: 32, textAlign: "center" }}>
                  <input
                    type="checkbox"
                    className="checkbox"
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
              {(hits ?? items).map((s) => (
                <tr key={s.session_id} className={selected.has(s.session_id) ? "sel" : ""}>
                  <td style={{ textAlign: "center", verticalAlign: "middle" }}>
                    <input
                      type="checkbox"
                      className="checkbox"
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
