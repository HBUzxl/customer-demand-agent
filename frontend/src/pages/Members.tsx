import { useCallback, useEffect, useState } from "react";
import {
  currentTenantGet,
  currentTenantUpdate,
  membersAdd,
  membersList,
  membersRemove,
  membersSetRole,
} from "../api/client";
import { useAuth } from "../auth/AuthProvider";
import PasswordInput from "../components/PasswordInput";
import ConfirmDialog from "../components/ConfirmDialog";
import Pager from "../components/Pager";
import { MEMBER_ROLES, ROLE_ADMIN, ROLE_ANALYST, ROLE_OWNER } from "../types";
import type { CurrentTenant, MemberView } from "../types";
import { PASSWORD_MAX, passwordError } from "../lib/password";

const ROLE_LABEL: Record<string, string> = {
  [ROLE_OWNER]: "所有者",
  [ROLE_ADMIN]: "管理员",
  [ROLE_ANALYST]: "分析人员",
  reviewer: "审核人员",
};

/**
 * 租户成员管理（owner/admin，§4.2）。
 * 页面级门禁：非 owner/admin 直接访问显示无权限；Service 层仍做最终角色校验。
 */
export default function Members() {
  const { user } = useAuth();
  const isOwner = !!user?.roles.includes(ROLE_OWNER);
  const isManager = isOwner || !!user?.roles.includes(ROLE_ADMIN);
  const [list, setList] = useState<MemberView[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [msg, setMsg] = useState("");
  // 添加成员弹窗
  const [addOpen, setAddOpen] = useState(false);
  const [addMode, setAddMode] = useState<"existing" | "new">("existing");
  const [email, setEmail] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [password, setPassword] = useState("");
  const [role, setRole] = useState(ROLE_ANALYST);
  const [addErr, setAddErr] = useState("");
  const [addBusy, setAddBusy] = useState(false);
  // 当前租户资料
  const [tenant, setTenant] = useState<CurrentTenant | null>(null);
  const [tenantOpen, setTenantOpen] = useState(false);
  const [tenantName, setTenantName] = useState("");
  const [tenantSlug, setTenantSlug] = useState("");
  const [tenantErr, setTenantErr] = useState("");
  const [tenantBusy, setTenantBusy] = useState(false);
  // 移除确认
  const [removeTarget, setRemoveTarget] = useState<MemberView | null>(null);
  const [removeErr, setRemoveErr] = useState("");
  // 成员列表分页（客户端 slice）
  const [memberPage, setMemberPage] = useState(0);
  const PAGE_SIZE = 10;

  // owner 可分配任意角色；admin 只能分配 analyst/reviewer（Service 层同样限制）
  const assignableRoles = isOwner
    ? MEMBER_ROLES
    : MEMBER_ROLES.filter((r) => r.value !== ROLE_OWNER && r.value !== ROLE_ADMIN);

  // 当前用户能否管理该成员（admin 不能动 owner/admin）
  const canManageMember = (m: MemberView) =>
    isOwner || m.role === ROLE_ANALYST || m.role === "reviewer";

  const load = useCallback(async () => {
    if (!isManager) return;
    setError("");
    try {
      const [r, t] = await Promise.all([membersList(), currentTenantGet()]);
      setList(r.items || []);
      setTenant(t);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, [isManager]);
  useEffect(() => {
    void load();
  }, [load]);

  function openAdd() {
    setAddMode("existing");
    setEmail("");
    setDisplayName("");
    setPassword("");
    setRole(ROLE_ANALYST);
    setAddErr("");
    setAddOpen(true);
  }
  async function doAdd() {
    if (!email.trim()) {
      setAddErr("请输入邮箱");
      return;
    }
    if (addMode === "new") {
      if (!displayName.trim()) {
        setAddErr("请输入姓名");
        return;
      }
      const pwErr = passwordError(password);
      if (pwErr) {
        setAddErr(pwErr);
        return;
      }
    }
    setAddBusy(true);
    setAddErr("");
    try {
      await membersAdd(
        email.trim(),
        addMode === "new" ? displayName.trim() : "",
        addMode === "new" ? password : "",
        role,
      );
      setAddOpen(false);
      setMsg(
        addMode === "new"
          ? `已新建并添加：${displayName.trim()}（${ROLE_LABEL[role] || role}）`
          : `已将 ${email.trim()} 加入本租户（${ROLE_LABEL[role] || role}）`,
      );
      void load();
    } catch (e) {
      setAddErr(e instanceof Error ? e.message : String(e));
    } finally {
      setAddBusy(false);
    }
  }

  function openTenantEdit() {
    if (!tenant) return;
    setTenantName(tenant.name);
    setTenantSlug(tenant.slug);
    setTenantErr("");
    setTenantOpen(true);
  }

  async function saveTenant() {
    setTenantBusy(true);
    setTenantErr("");
    try {
      await currentTenantUpdate(tenantName.trim(), tenantSlug.trim());
      setTenant((t) => (t ? { ...t, name: tenantName.trim(), slug: tenantSlug.trim() } : t));
      setTenantOpen(false);
      setMsg("租户资料已更新；工作空间名称将在下次刷新身份信息后同步显示");
    } catch (e) {
      setTenantErr(e instanceof Error ? e.message : String(e));
    } finally {
      setTenantBusy(false);
    }
  }

  async function changeRole(m: MemberView, next: string) {
    setError("");
    setMsg("");
    try {
      await membersSetRole(m.user_id, next);
      setList((xs) => xs.map((x) => (x.user_id === m.user_id ? { ...x, role: next } : x)));
      setMsg(`已将 ${m.display_name} 设为 ${ROLE_LABEL[next] || next}`);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  function openRemove(m: MemberView) {
    setRemoveErr("");
    setRemoveTarget(m);
  }
  async function doRemove() {
    const m = removeTarget;
    if (!m) return;
    setRemoveErr("");
    try {
      await membersRemove(m.user_id);
      setList((xs) => xs.filter((x) => x.user_id !== m.user_id));
      setMsg(`已移除成员：${m.display_name}`);
      setRemoveTarget(null);
    } catch (e) {
      setRemoveErr(e instanceof Error ? e.message : String(e));
    }
  }

  // 非 owner/admin 直接访问也拒绝展示（导航已隐藏，双保险）
  if (!isManager) {
    return (
      <div className="page">
        <div className="panel">
          <h3>成员管理</h3>
          <p className="muted">成员管理需租户所有者或管理员权限（§4.2），当前账号无权访问。</p>
        </div>
      </div>
    );
  }

  // 客户端分页：slice 当前页（页码越界时收敛到末页）
  const totalPages = Math.max(1, Math.ceil(list.length / PAGE_SIZE));
  const safePage = Math.min(memberPage, totalPages - 1);
  const pageList = list.slice(safePage * PAGE_SIZE, safePage * PAGE_SIZE + PAGE_SIZE);

  return (
    <div className="page">
      <div className="page-head">
        <h2>成员管理</h2>
        <span className="crumb">MEMBERS</span>
        <div className="page-actions">
          <button className="btn ghost sm" onClick={openTenantEdit} disabled={!tenant}>
            编辑租户
          </button>
          <button className="btn sm" onClick={openAdd}>
            + 添加成员
          </button>
        </div>
      </div>
      <div className="page-sub">
        管理本租户成员与角色 —— 添加成员（邮箱已注册则复用）、调整角色、移除成员。
        {user?.tenant_id && (
          <span
            className="tenant-id"
            title={`租户 ID：${user.tenant_id}`}
            style={{ marginLeft: 8 }}
          >
            租户 {user.tenant_name} #{user.tenant_id.slice(0, 8)}
          </span>
        )}
      </div>

      {error && <div className="error">! {error}</div>}
      {msg && <div className="panel admin-msg">{msg}</div>}
      {loading && <div className="loading">加载中…</div>}

      <div className="panel flush">
        <table>
          <thead>
            <tr>
              <th>姓名</th>
              <th>邮箱</th>
              <th>角色</th>
              <th>状态</th>
              <th>最近登录</th>
              <th style={{ width: 200 }}>操作</th>
            </tr>
          </thead>
          <tbody>
            {list.length === 0 && !loading && (
              <tr>
                <td colSpan={6} className="empty">
                  暂无成员
                </td>
              </tr>
            )}
            {pageList.map((m) => {
              const isSelf = m.user_id === user?.user_id;
              const manageable = canManageMember(m) && !isSelf;
              return (
                <tr key={m.user_id}>
                  <td>
                    {m.display_name}
                    {isSelf && (
                      <span className="faint" style={{ marginLeft: 6 }}>
                        （我）
                      </span>
                    )}
                  </td>
                  <td className="mono">{m.email}</td>
                  <td>
                    {manageable ? (
                      <select
                        className="member-role-select"
                        value={m.role}
                        onChange={(e) => changeRole(m, e.target.value)}
                      >
                        {assignableRoles.map((r) => (
                          <option key={r.value} value={r.value}>
                            {r.label}
                          </option>
                        ))}
                      </select>
                    ) : (
                      <span className="role-chip">{ROLE_LABEL[m.role] || m.role}</span>
                    )}
                  </td>
                  <td>
                    <span className={`badge ${m.status === "active" ? "verified" : "reject"}`}>
                      {m.status === "active"
                        ? "正常"
                        : m.status === "disabled"
                          ? "已禁用"
                          : m.status}
                    </span>
                  </td>
                  <td className="muted">
                    {m.last_login_at ? new Date(m.last_login_at).toLocaleString() : "—"}
                  </td>
                  <td>
                    {manageable && (
                      <button className="btn ghost sm danger-text" onClick={() => openRemove(m)}>
                        移除
                      </button>
                    )}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
        <Pager page={safePage} totalPages={totalPages} onChange={setMemberPage} />
      </div>

      {/* 添加成员弹窗 */}
      {addOpen && (
        <div className="cd-mask" onClick={() => !addBusy && setAddOpen(false)} role="presentation">
          <div
            className="cd-box"
            role="dialog"
            aria-modal="true"
            aria-label="添加成员"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="cd-title">添加成员</div>
            <div className="cd-body">
              <div className="pw-form">
                <div className="tab-bar compact">
                  <button
                    className={`tab${addMode === "existing" ? " active" : ""}`}
                    onClick={() => setAddMode("existing")}
                  >
                    添加已有用户
                  </button>
                  <button
                    className={`tab${addMode === "new" ? " active" : ""}`}
                    onClick={() => setAddMode("new")}
                  >
                    新建用户
                  </button>
                </div>
                <label>
                  {addMode === "existing" ? "已有用户邮箱" : "新用户邮箱"}
                  <input
                    value={email}
                    placeholder="name@example.com"
                    autoComplete="off"
                    onChange={(e) => setEmail(e.target.value)}
                  />
                </label>
                {addMode === "new" && (
                  <label>
                    姓名
                    <input
                      value={displayName}
                      placeholder="如：张三"
                      autoComplete="off"
                      onChange={(e) => setDisplayName(e.target.value)}
                    />
                  </label>
                )}
                {addMode === "new" && (
                  <label>
                    初始密码（8~16 字符，需含大小写字母、数字、符号中至少三种）
                    <PasswordInput
                      value={password}
                      maxLength={PASSWORD_MAX}
                      autoComplete="new-password"
                      onChange={(e) => setPassword(e.target.value)}
                    />
                  </label>
                )}
                <label>
                  角色
                  <select value={role} onChange={(e) => setRole(e.target.value)}>
                    {assignableRoles.map((r) => (
                      <option key={r.value} value={r.value}>
                        {r.label}
                      </option>
                    ))}
                  </select>
                </label>
                {addErr && (
                  <div className="error" style={{ marginTop: 4 }}>
                    ! {addErr}
                  </div>
                )}
              </div>
            </div>
            <div className="cd-btns">
              <button className="btn ghost sm" disabled={addBusy} onClick={() => setAddOpen(false)}>
                取消
              </button>
              <button className="btn sm" disabled={addBusy} onClick={doAdd}>
                {addBusy ? "提交中…" : "确认添加"}
              </button>
            </div>
          </div>
        </div>
      )}

      {tenantOpen && (
        <div
          className="cd-mask"
          onClick={() => !tenantBusy && setTenantOpen(false)}
          role="presentation"
        >
          <div
            className="cd-box"
            role="dialog"
            aria-modal="true"
            aria-label="编辑租户"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="cd-title">编辑本租户</div>
            <div className="cd-body pw-form">
              <label>
                租户名称
                <input
                  value={tenantName}
                  maxLength={80}
                  onChange={(e) => setTenantName(e.target.value)}
                />
              </label>
              <label>
                Slug
                <input
                  value={tenantSlug}
                  maxLength={40}
                  onChange={(e) => setTenantSlug(e.target.value.toLowerCase())}
                />
              </label>
              <div className="muted">
                Slug 仅用于展示，支持小写字母、数字和连字符；租户 ID 与数据目录不会改变。
              </div>
              {tenantErr && <div className="error">! {tenantErr}</div>}
            </div>
            <div className="cd-btns">
              <button
                className="btn ghost sm"
                disabled={tenantBusy}
                onClick={() => setTenantOpen(false)}
              >
                取消
              </button>
              <button className="btn sm" disabled={tenantBusy} onClick={saveTenant}>
                {tenantBusy ? "保存中…" : "保存"}
              </button>
            </div>
          </div>
        </div>
      )}

      <ConfirmDialog
        open={!!removeTarget}
        title="移除这个成员？"
        body={
          removeTarget
            ? `将把 ${removeTarget.display_name}（${removeTarget.email}）从本租户移除，其登录本租户的会话立即失效。`
            : ""
        }
        confirmText="移除"
        onConfirm={doRemove}
        onCancel={() => setRemoveTarget(null)}
      />
      <ConfirmDialog
        open={!!removeErr}
        title="移除失败"
        body={removeErr}
        danger={false}
        confirmText="知道了"
        cancelText="关闭"
        onConfirm={() => setRemoveErr("")}
        onCancel={() => setRemoveErr("")}
      />
    </div>
  );
}
