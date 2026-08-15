import { useEffect, useState } from "react";
import ConfirmDialog from "../components/ConfirmDialog";
import { reviewPending, reviewApprove, reviewReject } from "../api/client";
import type { ReviewItem } from "../types";

export default function Review() {
  const [items, setItems] = useState<ReviewItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [msg, setMsg] = useState("");
  const [rejectTarget, setRejectTarget] = useState<ReviewItem | null>(null);

  async function load() {
    setLoading(true);
    setError("");
    try {
      const r = await reviewPending();
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

  async function approve(it: ReviewItem) {
    try {
      await reviewApprove(it.type, it.title);
      setMsg(`已批准：${it.title}`);
      load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }
  function reject(it: ReviewItem) {
    setRejectTarget(it);
  }
  async function doReject() {
    const it = rejectTarget;
    if (!it) return;
    try {
      await reviewReject(it.type, it.title);
      setMsg(`已拒绝：${it.title}`);
      load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setRejectTarget(null);
    }
  }

  return (
    <div className="page">
      <div className="page-head">
        <h2>审核队列</h2>
        <span className="crumb">PENDING REVIEW</span>
      </div>
      <div className="page-sub">
        AI 写入的威胁 / 合规 / 行业记忆标记为待审核。批准转正式知识，拒绝则删除。
      </div>

      {msg && (
        <div className="panel" style={{ borderColor: "var(--green)", color: "var(--green)" }}>
          <span className="mono">{msg}</span>
        </div>
      )}
      {error && <div className="error">! {error}</div>}
      {loading && <div className="loading">加载中…</div>}
      {!loading && items.length === 0 && (
        <div className="panel">
          <div className="empty">队列为空 — 暂无待审核记忆</div>
        </div>
      )}

      {items.map((it, i) => (
        <div className="panel" key={i}>
          <div className="row">
            <strong style={{ fontSize: 15 }}>{it.title}</strong>
            <span className="tag">{it.type}</span>
            <span className="badge pending_review right">待审核</span>
          </div>
          <div className="faint mono" style={{ fontSize: 12, marginTop: 6 }}>
            {it.summary}
          </div>
          <pre
            className="verdict-detail mono"
            style={{ fontSize: 12.5, marginTop: 10, whiteSpace: "pre-wrap" }}
          >
            {it.content}
          </pre>
          {it.tags && it.tags.length > 0 && (
            <div style={{ marginTop: 8 }}>
              {it.tags.map((t, j) => (
                <span key={j} className="tag">
                  {t}
                </span>
              ))}
            </div>
          )}
          <div className="row" style={{ marginTop: 14 }}>
            <button className="btn green sm" onClick={() => approve(it)}>
              批准
            </button>
            <button className="btn danger sm" onClick={() => reject(it)}>
              拒绝
            </button>
          </div>
        </div>
      ))}
      <ConfirmDialog
        open={!!rejectTarget}
        title={`拒绝并删除：${rejectTarget?.title ?? ""}？`}
        body="拒绝将删除该待审条目。"
        confirmText="拒绝"
        onConfirm={doReject}
        onCancel={() => setRejectTarget(null)}
      />
    </div>
  );
}
