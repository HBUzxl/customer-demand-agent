import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { memoryDelete, memoryList, memorySearch } from "../api/client";
import ConfirmDialog from "../components/ConfirmDialog";
import type { MemoryEntry } from "../types";

const TYPES = [
  { id: "product", label: "产品" },
  { id: "threat", label: "威胁" },
  { id: "compliance", label: "合规" },
  { id: "industry", label: "行业" },
  { id: "customer", label: "客户" },
  { id: "user", label: "使用者" },
];

export default function MemoryList() {
  const navigate = useNavigate();
  const [type, setType] = useState("product");
  const [items, setItems] = useState<MemoryEntry[]>([]);
  const [query, setQuery] = useState("");
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [batchOpen, setBatchOpen] = useState(false);
  const [batchBusy, setBatchBusy] = useState(false);
  const [batchErr, setBatchErr] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  function toggle(key: string) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  }
  function toggleAllFlat() {
    setSelected(allFlatSelected ? new Set() : new Set(flat.map((e) => `${e.type}/${e.title}`)));
  }
  async function doBatchDelete() {
    setBatchBusy(true);
    const keys = [...selected];
    const results = await Promise.allSettled(
      keys.map((k) => {
        const i = k.indexOf("/");
        return memoryDelete(k.slice(0, i), k.slice(i + 1), false);
      }),
    );
    const okN = results.filter((r) => r.status === "fulfilled").length;
    const failN = keys.length - okN;
    setBatchOpen(false);
    setSelected(new Set());
    if (failN > 0) {
      const fe = results.find((r) => r.status === "rejected") as PromiseRejectedResult | undefined;
      setBatchErr(
        `批量删除：成功 ${okN}，失败 ${failN}（${fe?.reason instanceof Error ? fe.reason.message : String(fe?.reason)}）`,
      );
    } else {
      setBatchErr("");
    }
    setBatchBusy(false);
    load();
  }

  async function load() {
    setLoading(true);
    setError("");
    try {
      const r = query.trim()
        ? await memorySearch(query, type, 30)
        : await memoryList(type, 0, 2000);
      setItems(r.items || []);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }
  useEffect(() => {
    load();
  }, [type]);

  const searching = query.trim().length > 0;
  // 产品类型：主产品（无 product 字段）；统计 verified 文档数
  const products = type === "product" && !searching ? items.filter((e) => !e.product) : [];
  const docCount = (name: string) =>
    items.filter((e) => e.product === name && e.status !== "pending_review").length;

  // 搜索时（含产品子文档）或非产品类型：直接展示 items 平铺列表
  const flat = searching ? items : type !== "product" ? items : [];
  const allFlatSelected =
    flat.length > 0 && flat.every((e) => selected.has(`${e.type}/${e.title}`));

  return (
    <div className="page">
      <div className="page-head">
        <h2>记忆库</h2>
        <div className="page-actions">
          <button className="btn sm" onClick={() => navigate("/memory/new")}>
            + 新建
          </button>
        </div>
      </div>

      <div className="tab-bar">
        {TYPES.map((t) => (
          <button
            key={t.id}
            className={`tab ${type === t.id ? "active" : ""}`}
            onClick={() => {
              setType(t.id);
              setQuery("");
            }}
          >
            {t.label}
          </button>
        ))}
        <input
          className="tab-search"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="搜索全部内容（含子文档正文）…"
          onKeyDown={(e) => e.key === "Enter" && load()}
        />
      </div>

      {error && <div className="error">! {error}</div>}
      {searching && !loading && (
        <div className="search-meta">
          全库搜索「{query.trim()}」 · {items.length} 条结果（含产品子文档）
          {items.length === 0 && (
            <button
              className="btn ghost sm"
              onClick={() =>
                navigate(`/memory/new?type=${type}&title=${encodeURIComponent(query.trim())}`)
              }
            >
              新建「{query.trim()}」
            </button>
          )}
        </div>
      )}

      {loading && <div className="loading">加载中…</div>}
      {!loading && items.length === 0 && (
        <div className="panel">
          <div className="empty">该类型暂无条目</div>
        </div>
      )}

      {/* 产品：卡片（标题 + 别名 + 文档数）*/}
      {!loading && type === "product" && products.length > 0 && (
        <div className="doc-grid">
          {products.map((p) => (
            <div
              key={p.title}
              className="doc-card"
              onClick={() => navigate(`/memory/product/${encodeURIComponent(p.title)}`)}
            >
              <div className="row" style={{ justifyContent: "space-between" }}>
                <span className="dc-title" style={{ fontSize: 15 }}>
                  {p.title}
                </span>
                {docCount(p.title) > 0 && (
                  <span className="tag" style={{ margin: 0 }}>
                    {docCount(p.title)} 篇文档
                  </span>
                )}
              </div>
              {p.aliases && p.aliases.length > 0 && (
                <div className="dc-summary">{p.aliases.join(" / ")}</div>
              )}
            </div>
          ))}
        </div>
      )}

      {/* 非产品：轻量卡片（标题 + 标签 + 状态）*/}
      {!loading && flat.length > 0 && (
        <>
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
          <div className="doc-grid">
            <div className="doc-grid-actions">
              <label
                style={{
                  fontSize: 12,
                  color: "var(--text-faint)",
                  display: "flex",
                  alignItems: "center",
                  gap: 6,
                }}
              >
                <input type="checkbox" checked={allFlatSelected} onChange={toggleAllFlat} />
                全选本页
              </label>
            </div>
            {flat.map((e) => (
              <div
                key={e.title}
                className={`doc-card${selected.has(`${e.type}/${e.title}`) ? " sel" : ""}`}
                onClick={() => navigate(`/memory/${e.type}/${encodeURIComponent(e.title)}`)}
              >
                <input
                  type="checkbox"
                  style={{ position: "absolute", top: 8, left: 8 }}
                  checked={selected.has(`${e.type}/${e.title}`)}
                  onClick={(ev) => ev.stopPropagation()}
                  onChange={() => toggle(`${e.type}/${e.title}`)}
                  aria-label={`选择 ${e.title}`}
                />
                <div className="row" style={{ justifyContent: "space-between" }}>
                  <span className="dc-title" style={{ fontSize: 14 }}>
                    {e.title}
                  </span>
                  {e.status && e.status !== "verified" && (
                    <span className={`badge ${e.status}`}>
                      {e.status === "pending_review" ? "待审" : e.status}
                    </span>
                  )}
                </div>
                {e.tags && e.tags.length > 0 && (
                  <div className="dc-tags">
                    {e.tags.slice(0, 4).map((t, j) => (
                      <span key={j} className="tag">
                        #{t}
                      </span>
                    ))}
                  </div>
                )}
              </div>
            ))}
          </div>
        </>
      )}
      <ConfirmDialog
        open={batchOpen}
        title={`删除 ${selected.size} 条记忆？`}
        body="物理删除（不可恢复，不进归档）。"
        confirmText="全部删除"
        busy={batchBusy}
        onConfirm={doBatchDelete}
        onCancel={() => setBatchOpen(false)}
      />
      {batchErr && (
        <div className="err-banner" onClick={() => setBatchErr("")}>
          {batchErr}（点击关闭）
        </div>
      )}
    </div>
  );
}
