import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { memoryList, memorySearch } from "../api/client";
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
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

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
    items.filter((e) => e.product === name && e.status !== "pending_review").length + 1;

  // 搜索时（含产品子文档）或非产品类型：直接展示 items 平铺列表
  const flat = searching ? items : type !== "product" ? items : [];

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
          placeholder="搜索…"
          onKeyDown={(e) => e.key === "Enter" && load()}
        />
      </div>

      {error && <div className="error">! {error}</div>}
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
                <span className="tag" style={{ margin: 0 }}>
                  {docCount(p.title)} 篇
                </span>
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
        <div className="doc-grid">
          {flat.map((e) => (
            <div
              key={e.title}
              className="doc-card"
              onClick={() => navigate(`/memory/${e.type}/${encodeURIComponent(e.title)}`)}
            >
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
      )}
    </div>
  );
}
