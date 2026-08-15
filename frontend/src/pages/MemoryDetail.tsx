import { useEffect, useState } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import { memoryGet, memoryList, memoryUpsert, memoryDelete } from "../api/client";
import type { MemoryEntry } from "../types";
import MemoryForm, { toValues, fromValues, type MemoryValues } from "../components/MemoryForm";
import MarkdownView from "../components/MarkdownView";
import ConfirmDialog from "../components/ConfirmDialog";

const typeLabel: Record<string, string> = {
  product: "产品",
  threat: "威胁",
  compliance: "合规",
  industry: "行业",
  customer: "客户",
  user: "使用者",
};

function docLabel(title: string, product: string): string {
  if (title === product) return "概览";
  if (title.startsWith(product + "-")) return title.slice(product.length + 1);
  return title;
}

export default function MemoryDetail() {
  const { type = "", title = "" } = useParams<{ type: string; title: string }>();
  const decodedTitle = decodeURIComponent(title);
  const navigate = useNavigate();
  const [entry, setEntry] = useState<MemoryEntry | null>(null);
  const [subDocs, setSubDocs] = useState<MemoryEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [confirmArchive, setConfirmArchive] = useState(false);
  const [archErr, setArchErr] = useState("");
  const [editing, setEditing] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  async function load() {
    setLoading(true);
    setError("");
    try {
      const e = await memoryGet(type, decodedTitle);
      setEntry(e);
      if (type === "product" && !e.product) {
        const all = await memoryList("product", 0, 2000);
        setSubDocs((all.items || []).filter((d) => d.product === decodedTitle));
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }
  useEffect(() => {
    load();
  }, [type, decodedTitle]);

  async function onSubmit(v: MemoryValues) {
    setSubmitting(true);
    setError("");
    try {
      const e = fromValues(v);
      await memoryUpsert(e);
      setEditing(false);
      if (e.title !== decodedTitle || e.type !== type)
        navigate(`/memory/${e.type}/${encodeURIComponent(e.title)}`, { replace: true });
      else load();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSubmitting(false);
    }
  }

  function handleArchive() {
    if (entry) setConfirmArchive(true);
  }
  async function doArchive() {
    if (!entry) return;
    try {
      await memoryDelete(entry.type, entry.title, true);
      navigate("/memory");
    } catch (e) {
      setArchErr(e instanceof Error ? e.message : String(e));
    } finally {
      setConfirmArchive(false);
    }
  }

  if (loading)
    return (
      <div className="page">
        <div className="loading">加载中…</div>
      </div>
    );
  if (error && !entry)
    return (
      <div className="page">
        <div className="error">! {error}</div>
      </div>
    );
  if (!entry)
    return (
      <div className="page">
        <div className="empty">条目不存在</div>
      </div>
    );

  const isMainProduct = type === "product" && !entry.product;
  // 售前可见的子文档（排除 internal/engineering = pending_review），排除自身
  const docs = subDocs.filter((d) => d.status !== "pending_review" && d.title !== decodedTitle);

  return (
    <div className="page">
      <div className="crumb-bar">
        <Link to="/memory" className="faint">
          记忆库
        </Link>
        {!isMainProduct && entry.product && (
          <>
            <span className="faint">/</span>
            <Link to={`/memory/product/${encodeURIComponent(entry.product)}`} className="faint">
              {entry.product}
            </Link>
          </>
        )}
        <span className="faint">/</span>
        <span style={{ fontSize: 13, color: "var(--text)" }}>
          {isMainProduct ? entry.title : docLabel(entry.title, entry.product || "")}
        </span>
      </div>

      <div className="mem-detail-head">
        <h2>{isMainProduct ? entry.title : docLabel(entry.title, entry.product || "")}</h2>
        {entry.status && entry.status !== "verified" && (
          <span className={`badge ${entry.status}`}>
            {entry.status === "pending_review" ? "待审核" : entry.status}
          </span>
        )}
        <div className="page-actions">
          <button className="btn ghost sm" onClick={() => setEditing((x) => !x)}>
            {editing ? "取消" : "编辑"}
          </button>
          {entry.type !== "product" && entry.type !== "user" && (
            <button className="btn ghost sm" onClick={handleArchive}>
              归档
            </button>
          )}
        </div>
      </div>

      {/* 元数据（顶部紧凑）*/}
      {!editing && (
        <div className="doc-meta">
          <div className="dm">
            <span className="dm-k">类型</span>
            <span className="dm-v">{typeLabel[entry.type] || entry.type}</span>
          </div>
          {entry.category && (
            <div className="dm">
              <span className="dm-k">分类</span>
              <span className="dm-v">{entry.category}</span>
            </div>
          )}
          {entry.aliases && entry.aliases.length > 0 && (
            <div className="dm">
              <span className="dm-k">别名</span>
              <span className="dm-v">{entry.aliases.join("、")}</span>
            </div>
          )}
          {entry.tags && entry.tags.length > 0 && (
            <div className="dm">
              <span className="dm-k">标签</span>
              <span className="dm-tags">
                {entry.tags.map((t, i) => (
                  <span key={i} className="tag">
                    #{t}
                  </span>
                ))}
              </span>
            </div>
          )}
        </div>
      )}

      {error && <div className="error">! {error}</div>}

      {editing ? (
        <MemoryForm
          initial={toValues(entry)}
          submitting={submitting}
          onSubmit={onSubmit}
          onCancel={() => setEditing(false)}
        />
      ) : (
        <>
          {/* 主产品：文档卡片网格 */}
          {isMainProduct && docs.length > 0 && (
            <div className="doc-grid">
              {docs.map((doc) => (
                <div
                  key={doc.title}
                  className="doc-card"
                  onClick={() => navigate(`/memory/product/${encodeURIComponent(doc.title)}`)}
                >
                  <div className="dc-title">{docLabel(doc.title, entry.title)}</div>
                  {doc.summary && <div className="dc-summary">{doc.summary}</div>}
                </div>
              ))}
            </div>
          )}

          {/* 正文 markdown 渲染（文档详情+主产品页均显示；主产品页在画像下方、文档卡片上方）*/}
          <div className="panel">
            <MarkdownView>{entry.content || ""}</MarkdownView>
          </div>
        </>
      )}
      <ConfirmDialog
        open={!!confirmArchive}
        title={`归档 ${entry?.title ?? ""}？`}
        body="归档后不再参与检索，可从记忆库恢复。"
        confirmText="归档"
        danger={false}
        onConfirm={doArchive}
        onCancel={() => setConfirmArchive(false)}
      />
      {archErr && (
        <div className="err-banner" onClick={() => setArchErr("")}>
          {archErr}（点击关闭）
        </div>
      )}
      <ConfirmDialog
        open={!!confirmArchive}
        title={`归档 ${entry?.title ?? ""}？`}
        body="归档后不再参与检索。"
        confirmText="归档"
        danger={false}
        onConfirm={doArchive}
        onCancel={() => setConfirmArchive(false)}
      />
      {archErr && (
        <div className="err-banner" onClick={() => setArchErr("")}>
          {archErr}（点击关闭）
        </div>
      )}
    </div>
  );
}
