import { useEffect, useRef, useState } from "react";
import type { MouseEvent } from "react";
import { NavLink, Outlet, useLocation, useNavigate } from "react-router-dom";
import {
  sessionDelete,
  sessionRename,
  sessionsList,
  sessionsSearch,
  clearLocalKeys,
} from "../api/client";
import ConfirmDialog from "../components/ConfirmDialog";
import PasswordInput from "../components/PasswordInput";
import Icon from "../components/Icon";
import { useAuth } from "../auth/AuthProvider";
import { ROLE_ADMIN, ROLE_OWNER, ROLE_PLATFORM_ADMIN } from "../types";
import type { SessionListItem } from "../types";
import { passwordError, PASSWORD_MAX } from "../lib/password";

type RecentSession = SessionListItem & { running?: boolean };

const secondaryNav = [
  {
    to: "/dashboard",
    label: "商机面板",
    icon: "M3 3v18h18M7 14l4-4 3 3 5-6",
  },
  { to: "/memory", label: "记忆库", icon: "M4 4h16v6H4zM4 14h16v6H4zM8 7h.01M8 17h.01" },
  {
    to: "/members",
    label: "成员管理",
    icon: "M16 11a3 3 0 1 0 0-6 3 3 0 0 0 0 6zM8 11a3 3 0 1 0 0-6 3 3 0 0 0 0 6zM6 21v-2a4 4 0 0 1 4-4h4a4 4 0 0 1 4 4v2",
    need: [ROLE_OWNER, ROLE_ADMIN],
  },
  {
    to: "/admin/users",
    label: "用户管理",
    icon: "M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10zM12 8v4M10 10h4",
    need: [ROLE_PLATFORM_ADMIN],
  },
  {
    to: "/observe",
    label: "观测台",
    icon: "M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6zM19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z",
    need: [ROLE_PLATFORM_ADMIN],
  },
  {
    to: "/settings",
    label: "设置",
    icon: "M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6zM19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z",
  },
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
  const { user, lsKey, logout, changePassword, switchTenant } = useAuth();
  const [recent, setRecent] = useState<RecentSession[]>([]);
  const [sessionQuery, setSessionQuery] = useState(""); // G6 会话搜索
  const [searchHits, setSearchHits] = useState<RecentSession[] | null>(null);
  const [confirmTarget, setConfirmTarget] = useState<string | null>(null);
  const [delErr, setDelErr] = useState("");
  // 会话 hover 菜单 + 内联重命名（#17，2026-08-18）
  const [menuPos, setMenuPos] = useState<{ id: string; right: number; top: number } | null>(null);
  const [renamingId, setRenamingId] = useState<string | null>(null);
  const [renameValue, setRenameValue] = useState("");
  const [renameErr, setRenameErr] = useState("");
  const renameDoneRef = useRef(false); // Enter 提交后 onBlur 复用防抖
  // 修改密码弹窗
  const [pwOpen, setPwOpen] = useState(false);
  const [pwOld, setPwOld] = useState("");
  const [pwNew, setPwNew] = useState("");
  const [pwNew2, setPwNew2] = useState("");
  const [pwErr, setPwErr] = useState("");
  const [pwBusy, setPwBusy] = useState(false);
  const [pwDone, setPwDone] = useState(false);
  const [workspaceBusy, setWorkspaceBusy] = useState(false);
  const [workspaceErr, setWorkspaceErr] = useState("");
  // 侧栏折叠态：cda:<tenant>:<user>: 前缀（租户/用户隔离，退出清理）
  const [collapsed, setCollapsed] = useState(
    () => localStorage.getItem(lsKey("sidebar")) === "collapsed",
  );

  function toggle() {
    const c = !collapsed;
    if (c) {
      setSessionQuery("");
      setSearchHits(null);
    }
    setCollapsed(c);
    localStorage.setItem(lsKey("sidebar"), c ? "collapsed" : "expanded");
  }

  async function onWorkspaceChange(tenantId: string) {
    setWorkspaceBusy(true);
    setWorkspaceErr("");
    try {
      await switchTenant(tenantId);
    } catch (e) {
      setWorkspaceErr(e instanceof Error ? e.message : String(e));
      setWorkspaceBusy(false);
    }
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
    setMenuPos(null);
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

  // 会话「⋯」菜单：position:fixed 锚定按钮右缘（避免被 .conv-list overflow 裁切）。
  function toggleMenu(e: MouseEvent, id: string) {
    e.stopPropagation();
    if (menuPos?.id === id) {
      setMenuPos(null);
      return;
    }
    const r = (e.currentTarget as HTMLElement).getBoundingClientRect();
    setMenuPos({ id, right: window.innerWidth - r.right + 2, top: r.bottom + 2 });
  }
  // 点击其他区域关闭菜单（菜单内部 mousedown 不关——否则按钮 click 前菜单已被卸载）
  useEffect(() => {
    if (!menuPos) return;
    const onDocDown = (e: globalThis.MouseEvent) => {
      if ((e.target as HTMLElement).closest?.(".conv-menu")) return;
      setMenuPos(null);
    };
    document.addEventListener("mousedown", onDocDown);
    return () => document.removeEventListener("mousedown", onDocDown);
  }, [menuPos]);

  function startRename(id: string, title: string) {
    renameDoneRef.current = false;
    setRenamingId(id);
    setRenameValue(title);
    setMenuPos(null);
  }
  function cancelRename() {
    if (renameDoneRef.current) return;
    renameDoneRef.current = true;
    setRenamingId(null);
    setRenameValue("");
    setRenameErr("");
  }
  async function commitRename(id: string) {
    if (renameDoneRef.current) return; // Enter 触发后 onBlur 复用
    const v = renameValue.trim();
    if (!v) {
      cancelRename();
      return;
    }
    if (v.length > 100) {
      setRenameErr("会话名称需为 1~100 字符");
      return;
    }
    renameDoneRef.current = true;
    setRenameErr("");
    try {
      await sessionRename(id, v);
      const patch = (xs: RecentSession[]) =>
        xs.map((s) => (s.session_id === id ? { ...s, title: v } : s));
      setRecent(patch);
      setSearchHits((hs) => (hs ? patch(hs) : hs));
    } catch (err) {
      setRenameErr(`重命名失败：${err instanceof Error ? err.message : err}`);
      renameDoneRef.current = false; // 允许重试
    } finally {
      setRenamingId(null);
    }
  }

  async function onLogout() {
    await logout(); // 内部清 cda:* 本地键
    navigate("/login", { replace: true });
    // SPA 导航卸载页面组件时，草稿等 unmount cleanup 可能在 wipe 之后把 key 写回；
    // 导航提交后再次 wipe，保证退出后本地敏感态彻底清空（覆盖非空草稿）。
    setTimeout(clearLocalKeys, 0);
  }

  async function onPassword() {
    setPwErr("");
    if (pwNew !== pwNew2) {
      setPwErr("两次输入的新密码不一致");
      return;
    }
    const pwErr = passwordError(pwNew);
    if (pwErr) {
      setPwErr(pwErr);
      return;
    }
    setPwBusy(true);
    try {
      await changePassword(pwOld, pwNew);
      setPwDone(true);
      setPwOld("");
      setPwNew("");
      setPwNew2("");
      setTimeout(() => {
        setPwOpen(false);
        setPwDone(false);
      }, 1200);
    } catch (e) {
      setPwErr(e instanceof Error ? e.message : String(e));
    } finally {
      setPwBusy(false);
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
            <Icon
              d="M3 5h18a1 1 0 0 1 1 1v12a1 1 0 0 1-1 1H3a1 1 0 0 1-1-1V6a1 1 0 0 1 1-1zM9 5v14"
              size={18}
            />
          </button>
        </div>

        <button className="newchat" onClick={() => navigate("/analyze")} title="新建对话">
          <Icon d="M12 5v14M5 12h14" />
          {!collapsed && <span className="lbl">新建对话</span>}
        </button>

        {collapsed && (
          <div className="conv-dots">
            {recent.slice(0, 8).map((s) => (
              <button
                key={s.session_id}
                className={`conv-dot${location.pathname === `/analyze/${s.session_id}` ? " on" : ""}`}
                title={`${s.title || "未命名对话"}${s.customer ? ` · ${s.customer}` : ""}${s.running ? " · 运行中" : ""}`}
                onClick={() => navigate(`/analyze/${s.session_id}`)}
              >
                {(s.title || "?").slice(0, 1)}
                {s.running && <i className="run-dot" />}
              </button>
            ))}
          </div>
        )}
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
            {(searchHits ?? recent).map((s) => {
              const renaming = renamingId === s.session_id;
              return (
                <div
                  key={s.session_id}
                  className={`conv-item${renaming ? " renaming" : ""}`}
                  role="button"
                  tabIndex={0}
                  onClick={() => !renaming && navigate(`/analyze/${s.session_id}`)}
                  onKeyDown={(e) =>
                    e.key === "Enter" && !renaming && navigate(`/analyze/${s.session_id}`)
                  }
                  title={renaming ? undefined : s.title || s.session_id}
                >
                  {renaming ? (
                    <input
                      className="conv-rename-input"
                      value={renameValue}
                      autoFocus
                      maxLength={100}
                      onClick={(e) => e.stopPropagation()}
                      onChange={(e) => setRenameValue(e.target.value)}
                      onKeyDown={(e) => {
                        e.stopPropagation();
                        if (e.key === "Enter") commitRename(s.session_id);
                        else if (e.key === "Escape") cancelRename();
                      }}
                      onBlur={() => commitRename(s.session_id)}
                    />
                  ) : (
                    <>
                      <span className="conv-title">{s.title || "未命名对话"}</span>
                      {s.snippet && <span className="conv-snippet">{s.snippet}</span>}
                    </>
                  )}
                  {renameErr && renaming && <span className="conv-rename-err">{renameErr}</span>}
                  <span className="conv-time">
                    {s.customer && (
                      <i className="conv-cust" title={s.customer}>
                        {s.customer.slice(0, 6)}
                      </i>
                    )}
                    {s.running ? <i className="run-dot" title="运行中" /> : null}
                    {relTime(s.updated_at)}
                  </span>
                  {!renaming && (
                    <button
                      className="conv-more"
                      title="更多操作"
                      aria-label={`更多操作 ${s.title || s.session_id}`}
                      onClick={(e) => toggleMenu(e, s.session_id)}
                    >
                      <Icon
                        d="M12 5a1 1 0 1 1 0 2 1 1 0 0 1 0-2zm0 6a1 1 0 1 1 0 2 1 1 0 0 1 0-2zm0 6a1 1 0 1 1 0 2 1 1 0 0 1 0-2z"
                        size={14}
                      />
                    </button>
                  )}
                  {menuPos?.id === s.session_id && (
                    <div
                      className="conv-menu"
                      style={{ right: menuPos.right, top: menuPos.top }}
                      onClick={(e) => e.stopPropagation()}
                    >
                      <button
                        className="conv-menu-item"
                        onClick={() => startRename(s.session_id, s.title || "")}
                      >
                        <Icon d="M12 20h9M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4z" size={13} />
                        重命名
                      </button>
                      <button
                        className="conv-menu-item danger"
                        onClick={(e) => deleteSession(e, s.session_id)}
                      >
                        <Icon
                          d="M3 6h18M8 6V4a1 1 0 0 1 1-1h6a1 1 0 0 1 1 1v2m3 0v14a1 1 0 0 1-1 1H6a1 1 0 0 1-1-1V6h14M10 11v6M14 11v6"
                          size={13}
                        />
                        删除
                      </button>
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        )}

        <div className="side-bottom">
          {secondaryNav
            // 角色门禁：无 need 恒显；有 need 则当前用户 roles 含其一（成员=owner/admin，用户管理/观测台=platform_admin，§7.5）
            .filter((n) => !n.need || (user?.roles ?? []).some((r) => n.need!.includes(r)))
            .map((n) => (
              <NavLink key={n.to} to={n.to} className="sec-nav" title={n.label}>
                <Icon d={n.icon} size={collapsed ? 19 : 16} />
                {!collapsed && <span className="lbl">{n.label}</span>}
              </NavLink>
            ))}

          {user && (
            <div className="side-user">
              <div className="side-user-info" title={`${user.display_name} · ${user.email}`}>
                <div className="side-user-avatar">{(user.display_name || "?").slice(0, 1)}</div>
                {!collapsed && (
                  <div className="side-user-text">
                    <div className="side-user-name">{user.display_name}</div>
                    <div className="side-user-meta" title={workspaceErr || undefined}>
                      {(user.workspaces || []).length > 1 ? (
                        <select
                          className="workspace-select"
                          aria-label="切换工作空间"
                          value={user.tenant_id}
                          disabled={workspaceBusy}
                          onChange={(e) => void onWorkspaceChange(e.target.value)}
                        >
                          {user.workspaces.map((w) => (
                            <option key={w.tenant_id} value={w.tenant_id}>
                              {w.tenant_name} · {w.role}
                            </option>
                          ))}
                        </select>
                      ) : (
                        <>
                          {user.tenant_name}
                          <span className="tenant-id" title={`租户 ID：${user.tenant_id}`}>
                            · {user.tenant_id.slice(0, 8)}
                          </span>
                        </>
                      )}
                      {user.roles.map((r) => (
                        <span
                          key={r}
                          className={`role-chip${r === ROLE_PLATFORM_ADMIN ? " platform" : ""}`}
                          style={{ marginLeft: 6 }}
                        >
                          {r === ROLE_PLATFORM_ADMIN ? "平台管理" : r}
                        </span>
                      ))}
                    </div>
                  </div>
                )}
              </div>
              {!collapsed && (
                <div className="side-user-actions">
                  <button className="btn" onClick={() => setPwOpen(true)} title="修改密码">
                    改密
                  </button>
                  <button className="btn danger" onClick={onLogout} title="退出登录">
                    退出
                  </button>
                </div>
              )}
            </div>
          )}
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

      {pwOpen && (
        <div className="cd-mask" onClick={() => !pwBusy && setPwOpen(false)} role="presentation">
          <div
            className="cd-box"
            role="dialog"
            aria-modal="true"
            aria-label="修改密码"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="cd-title">{pwDone ? "密码已修改" : "修改密码"}</div>
            <div className="cd-body">
              {pwDone ? (
                <div style={{ color: "var(--green)" }}>新密码已生效。</div>
              ) : (
                <div className="pw-form">
                  <label>
                    当前密码
                    <PasswordInput
                      value={pwOld}
                      autoComplete="current-password"
                      onChange={(e) => setPwOld(e.target.value)}
                    />
                  </label>
                  <label>
                    新密码（8~16 字符，需含大小写字母、数字、符号中至少三种）
                    <PasswordInput
                      value={pwNew}
                      maxLength={PASSWORD_MAX}
                      autoComplete="new-password"
                      onChange={(e) => setPwNew(e.target.value)}
                    />
                  </label>
                  <label>
                    确认新密码
                    <PasswordInput
                      value={pwNew2}
                      autoComplete="new-password"
                      onChange={(e) => setPwNew2(e.target.value)}
                    />
                  </label>
                  {pwErr && <div className="error">! {pwErr}</div>}
                </div>
              )}
            </div>
            {!pwDone && (
              <div className="cd-btns">
                <button className="btn ghost sm" disabled={pwBusy} onClick={() => setPwOpen(false)}>
                  取消
                </button>
                <button className="btn sm" disabled={pwBusy} onClick={onPassword}>
                  {pwBusy ? "提交中…" : "确认修改"}
                </button>
              </div>
            )}
          </div>
        </div>
      )}

      <main className="main">
        <Outlet />
      </main>
    </>
  );
}
