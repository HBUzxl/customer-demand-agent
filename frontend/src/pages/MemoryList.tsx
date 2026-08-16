import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { memoryDelete, memoryList, memorySearch } from "../api/client";
import ConfirmDialog from "../components/ConfirmDialog";
import type { MemoryEntry } from "../types";

const TYPES = [
  { id: "", label: "全部" },
  { id: "product", label: "产品" },
  { id: "threat", label: "威胁" },
  { id: "compliance", label: "合规" },
  { id: "industry", label: "行业" },
  { id: "customer", label: "客户" },
  { id: "user", label: "使用者" },
];

/** 官方产品线展示顺序（与后端 L3a 目录一致） */
const LINES = ["流量安全", "端点安全", "安全平台", "安全开发", "漏洞扫描"];

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
  const products = useMemo(
    () => (type === "product" && !searching ? items.filter((e) => !e.product) : []),
    [type, searching, items],
  );
  const docCount = (name: string) =>
    items.filter((e) => e.product === name && e.status !== "pending_review").length;

  // 搜索时（含产品子文档）或非产品类型：平铺列表
  const flat = searching ? items : type !== "product" ? items : [];
  // 全选仅在「本页全部选中」时打勾（与 History 行为一致，不做半选态）
  const allFlatSelected =
    flat.length > 0 && flat.every((e) => selected.has(`${e.type}/${e.title}`));

  // 产品按官方产品线分组（主页 tags 首位；未知分类按字典序追加在后）
  const groups = useMemo(() => {
    const map = new Map<string, MemoryEntry[]>();
    for (const p of products) {
      const line = p.tags?.[0] || "其他";
      if (!map.has(line)) map.set(line, []);
      map.get(line)!.push(p);
    }
    return [...map.entries()].sort((a, b) => {
      const ia = LINES.indexOf(a[0]);
      const ib = LINES.indexOf(b[0]);
      const va = ia === -1 ? 99 : ia;
      const vb = ib === -1 ? 99 : ib;
      return va !== vb ? va - vb : a[0].localeCompare(b[0]);
    });
  }, [products]);
  const totalDocs = items.filter((e) => e.product && e.status !== "pending_review").length;

  return (
    <div className="page">
      <div className="page-head">
        <h2>记忆库</h2>
        <div className="page-actions">
          {!loading && !searching && type === "product" && products.length > 0 && (
            <span className="head-meta">
              {products.length} 个产品 · {totalDocs} 篇文档
            </span>
          )}
          <button className="btn sm" onClick={() => navigate("/memory/new")}>
            + 新建
          </button>
        </div>
      </div>

      <div className="tab-bar">
        {TYPES.filter((t) => t.id !== "" || searching).map((t) => (
          <button
            key={t.id || "all"}
            className={`tab ${type === t.id ? "active" : ""}`}
            onClick={() => {
              setType(t.id);
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

      {/* 产品：按产品线分组的产品卡片（标题/简介/文档数 三行等高结构） */}
      {!loading && type === "product" && products.length > 0 && (
        <>
          {groups.map(([line, ps]) => (
            <section key={line} className="prod-group">
              <div className="prod-group-head">
                <span className="pg-name">{line}</span>
                <span className="pg-count">{ps.length} 个产品</span>
              </div>
              <div className="doc-grid">
                {ps.map((p) => {
                  const n = docCount(p.title);
                  return (
                    <div
                      key={p.title}
                      className="doc-card"
                      onClick={() => navigate(`/memory/product/${encodeURIComponent(p.title)}`)}
                    >
                      <div className="dc-title" title={p.title}>
                        {p.title}
                      </div>
                      <div className="dc-desc">{p.summary || "暂无简介"}</div>
                      <div className="dc-foot">
                        <span className={`dc-count${n === 0 ? " zero" : ""}`}>
                          {n > 0 ? `${n} 篇文档` : "暂无文档"}
                        </span>
                        {p.aliases && p.aliases.length > 0 && (
                          <span className="dc-alias" title={p.aliases.join(" / ")}>
                            {p.aliases.join(" / ")}
                          </span>
                        )}
                      </div>
                    </div>
                  );
                })}
              </div>
            </section>
          ))}
        </>
      )}

      {/* 搜索结果 / 非产品类型：工具条（全选+批量操作）+ 平铺卡片 */}
      {!loading && flat.length > 0 && (
        <>
          <div className="list-toolbar">
            <label className="sel-all">
              <input
                type="checkbox"
                className="checkbox"
                checked={allFlatSelected}
                onChange={toggleAllFlat}
              />
              全选本页
            </label>
            <span className={`sel-count${selected.size > 0 ? "" : " dim"}`}>
              {selected.size > 0 ? `已选 ${selected.size} 项` : `共 ${flat.length} 条`}
            </span>
            {/* 批量操作按需出现：工具条高度固定（min-height 28px），按钮为紧凑版，不改变布局 */}
            {selected.size > 0 && (
              <div className="batch-actions">
                <button className="btn sm danger" onClick={() => setBatchOpen(true)}>
                  批量删除
                </button>
                <button className="btn ghost sm" onClick={() => setSelected(new Set())}>
                  取消选择
                </button>
              </div>
            )}
          </div>
          <div className="doc-grid">
            {flat.map((e) => (
              <div
                key={e.title}
                className={`doc-card${selected.has(`${e.type}/${e.title}`) ? " sel" : ""}`}
                onClick={() => navigate(`/memory/${e.type}/${encodeURIComponent(e.title)}`)}
              >
                <div className="dc-main">
                  <input
                    type="checkbox"
                    className="checkbox dc-check"
                    checked={selected.has(`${e.type}/${e.title}`)}
                    onClick={(ev) => ev.stopPropagation()}
                    onChange={() => toggle(`${e.type}/${e.title}`)}
                    aria-label={`选择 ${e.title}`}
                  />
                  <span className="dc-title" title={e.title}>
                    {e.title}
                  </span>
                  {e.status && e.status !== "verified" && (
                    <span className={`badge ${e.status}`}>
                      {e.status === "pending_review" ? "待审" : e.status}
                    </span>
                  )}
                </div>
                {e.summary && <div className="dc-summary">{e.summary}</div>}
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
