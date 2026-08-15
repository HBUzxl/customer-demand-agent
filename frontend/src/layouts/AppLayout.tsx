import { useEffect, useState } from "react";
import type { MouseEvent } from "react";
import { NavLink, Outlet, useLocation, useNavigate } from "react-router-dom";
import { sessionDelete, sessionsList, sessionsSearch } from "../api/client";
import ConfirmDialog from "../components/ConfirmDialog";
import type { SessionListItem } from "../types";

type RecentSession = SessionListItem & { running?: boolean };

const Ico = ({ d, size = 18 }: { d: string; size?: number }) => (
  <svg
    width={size}
    height={size}
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="1.6"
    strokeLinecap="round"
    strokeLinejoin="round"
  >
    <path d={d} />
  </svg>
);

const secondaryNav = [
  { to: "/memory", label: "记忆库", icon: "M4 4h16v6H4zM4 14h16v6H4zM8 7h.01M8 17h.01" },
  {
    to: "/review",
    label: "审核队列",
    icon: "M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10zM9 12l2 2 4-4",
  },
  { to: "/settings", label: "设置", icon: "M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6z" },
];

function relTime(iso: string) {
  const d = new Date(iso);
  const h = (Date.now() - d.getTime()) / 36e5;
  if (h < 1) return "刚刚";
  if (h < 24) return Math.floor(h) + " 小时前";
  if (h < 48) return "昨天";
  if (h < 168) return Math.floor(h / 24) + " 天前";
  return d.toLocaleDateString();
}

export default function AppLayout() {
  const navigate = useNavigate();
  const location = useLocation();
  const [recent, setRecent] = useState<RecentSession[]>([]);
  const [sessionQuery, setSessionQuery] = useState(""); // G6 会话搜索
  const [searchHits, setSearchHits] = useState<RecentSession[] | null>(null);
  const [confirmTarget, setConfirmTarget] = useState<string | null>(null);
  const [delErr, setDelErr] = useState("");
  const [collapsed, setCollapsed] = useState(() => localStorage.getItem("sidebar") === "collapsed");

  function toggle() {
    const c = !collapsed;
    setCollapsed(c);
    localStorage.setItem("sidebar", c ? "collapsed" : "expanded");
  }

  async function loadRecent() {
    try {
      const r = await sessionsList(12, 0);
      setRecent((r.items as RecentSession[]) || []);
    } catch {
      /* */
    }
  }
  useEffect(() => {
    loadRecent();
  }, [location.pathname]);
  // F0 运行指示：有会话在跑时快刷列表（3s），否则慢刷（15s）
  useEffect(() => {
    const anyRunning = recent.some((s) => s.running);
    const t = setInterval(loadRecent, anyRunning ? 3000 : 15000);
    return () => clearInterval(t);
  }, [recent]);

  // G2b 内容搜索：防抖 300ms 走后端（标题命中优先+消息内容 snippet）
  useEffect(() => {
    const q = sessionQuery.trim();
    if (!q) {
      setSearchHits(null);
      return;
    }
    const t = setTimeout(async () => {
      const hits = await sessionsSearch(q, 15);
      setSearchHits(hits.map((h) => ({ ...h, created_at: h.updated_at })));
    }, 300);
    return () => clearTimeout(t);
  }, [sessionQuery]);

  function deleteSession(e: MouseEvent, id: string) {
    e.stopPropagation();
    setConfirmTarget(id);
  }
  async function doDelete() {
    const id = confirmTarget;
    if (!id) return;
    try {
      await sessionDelete(id);
      setRecent((rs) => rs.filter((s) => s.session_id !== id));
      if (location.pathname === `/analyze/${id}`) navigate("/analyze");
    } catch (err) {
      setDelErr(`删除失败：${err instanceof Error ? err.message : err}`);
    } finally {
      setConfirmTarget(null);
    }
  }

  return (
    <>
      <aside className="sidebar" data-collapsed={collapsed}>
        <div className="side-head">
          {!collapsed && (
            <div className="brand">
              <div className="logo">长</div>
              <div className="name">
                需求分析<span> · 长亭</span>
              </div>
            </div>
          )}
          <button
            className="side-toggle"
            onClick={toggle}
            title={collapsed ? "展开侧栏" : "收起侧栏"}
            aria-label="切换侧栏"
          >
            <Ico
              d="M3 5h18a1 1 0 0 1 1 1v12a1 1 0 0 1-1 1H3a1 1 0 0 1-1-1V6a1 1 0 0 1 1-1zM9 5v14"
              size={18}
            />
          </button>
        </div>

        <button className="newchat" onClick={() => navigate("/analyze")} title="新建对话">
          <Ico d="M12 5v14M5 12h14" />
          {!collapsed && <span className="lbl">新建对话</span>}
        </button>

        {!collapsed && (
          <div className="conv-list">
            <div className="conv-head">
              <span>最近对话</span>
              <NavLink to="/history" className="conv-all">
                全部
              </NavLink>
            </div>
            {searchHits && searchHits.length === 0 && <div className="conv-empty">无匹配对话</div>}
            {!collapsed && (
              <input
                className="conv-search"
                placeholder="搜索会话…"
                value={sessionQuery}
                onChange={(e) => setSessionQuery(e.target.value)}
              />
            )}
            {recent.length === 0 && !searchHits && <div className="conv-empty">暂无对话</div>}
            {(searchHits ?? recent).map((s) => (
              <div
                key={s.session_id}
                className="conv-item"
                role="button"
                tabIndex={0}
                onClick={() => navigate(`/analyze/${s.session_id}`)}
                onKeyDown={(e) => e.key === "Enter" && navigate(`/analyze/${s.session_id}`)}
                title={s.title || s.session_id}
              >
                <span className="conv-title">
                  {s.customer && (
                    <i className="conv-cust" title={s.customer}>
                      {s.customer.slice(0, 6)}
                    </i>
                  )}
                  {s.title || "未命名对话"}
                </span>
                {s.snippet && <span className="conv-snippet">{s.snippet}</span>}
                <span className="conv-time">
                  {s.running ? <i className="run-dot" title="运行中" /> : null}
                  {relTime(s.updated_at)}
                </span>
                <button
                  className="conv-del"
                  title="删除对话"
                  aria-label={`删除对话 ${s.title || s.session_id}`}
                  onClick={(e) => deleteSession(e, s.session_id)}
                >
                  <Ico
                    d="M3 6h18M8 6V4a1 1 0 0 1 1-1h6a1 1 0 0 1 1 1v2m3 0v14a1 1 0 0 1-1 1H6a1 1 0 0 1-1-1V6h14M10 11v6M14 11v6"
                    size={14}
                  />
                </button>
              </div>
            ))}
          </div>
        )}

        <div className="side-bottom">
          {secondaryNav.map((n) => (
            <NavLink key={n.to} to={n.to} className="sec-nav" title={n.label}>
              <Ico d={n.icon} size={collapsed ? 19 : 16} />
              {!collapsed && <span className="lbl">{n.label}</span>}
            </NavLink>
          ))}
        </div>
      </aside>
      <ConfirmDialog
        open={!!confirmTarget}
        title="删除这个对话？"
        body="删除后不可恢复。"
        confirmText="删除"
        onConfirm={doDelete}
        onCancel={() => setConfirmTarget(null)}
      />
      <ConfirmDialog
        open={!!delErr}
        title="删除失败"
        body={delErr}
        danger={false}
        confirmText="知道了"
        cancelText="关闭"
        onConfirm={() => setDelErr("")}
        onCancel={() => setDelErr("")}
      />

      <main className="main">
        <Outlet />
      </main>
    </>
  );
}
