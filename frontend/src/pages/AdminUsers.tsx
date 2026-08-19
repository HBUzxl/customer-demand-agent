import { useCallback, useEffect, useState } from "react";
import type { MouseEvent } from "react";
import {
  platformTenantCreate,
  platformTenantDelete,
  platformTenantMemberAdd,
  platformTenantMemberRemove,
  platformTenantMemberRole,
  platformTenantMembers,
  platformTenantStatus,
  platformTenantUpdate,
  platformTenants,
  platformUserCreate,
  platformUserDelete,
  platformUserResetPassword,
  platformUserSetAdmin,
  platformUserUpdate,
  platformUsers,
  platformUserStatus,
} from "../api/client";
import { useAuth } from "../auth/AuthProvider";
import PasswordInput from "../components/PasswordInput";
import Icon from "../components/Icon";
import ConfirmDialog from "../components/ConfirmDialog";
import Pager from "../components/Pager";
import { ROLE_ADMIN, ROLE_ANALYST, ROLE_OWNER, ROLE_PLATFORM_ADMIN, ROLE_REVIEWER } from "../types";
import type { AdminTenant, AdminUser, MemberView } from "../types";
import { PASSWORD_MAX, passwordError } from "../lib/password";

const ROLE_LABEL: Record<string, string> = {
  [ROLE_PLATFORM_ADMIN]: "平台管理员",
  [ROLE_OWNER]: "所有者",
  [ROLE_ADMIN]: "管理员",
  [ROLE_ANALYST]: "分析人员",
  [ROLE_REVIEWER]: "审核人员",
};

const USER_STATUS_LABEL: Record<string, string> = { active: "正常", disabled: "已禁用" };
const TENANT_STATUS_LABEL: Record<string, string> = {
  active: "正常",
  disabled: "已停用",
  provisioning: "初始化中",
  provisioning_failed: "初始化失败",
  deleting: "删除中",
  deleted: "已删除",
};

const DOTS_PATH =
  "M12 5a1 1 0 1 1 0 2 1 1 0 0 1 0-2zm0 6a1 1 0 1 1 0 2 1 1 0 0 1 0-2zm0 6a1 1 0 1 1 0 2 1 1 0 0 1 0-2z";

type Tab = "users" | "tenants";

/** 列表每页条数（用户表/租户表/成员抽屉客户端分页）。 */
const PAGE_SIZE = 10;

/**
 * 平台用户/租户管理（platform_admin，§4.2）。
 * 平台管理员只拥有平台级运维元数据权限，不是"超级租户"——不进入任何租户数据面。
 */
export default function AdminUsers() {
  const { user, isPlatformAdmin } = useAuth();
  const [tab, setTab] = useState<Tab>("users");
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [tenants, setTenants] = useState<AdminTenant[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [msg, setMsg] = useState("");
  // 列表分页（客户端 slice）
  const [userPage, setUserPage] = useState(0);
  const [tenantPage, setTenantPage] = useState(0);
  const [memberPage, setMemberPage] = useState(0);
  // 重置密码弹窗
  const [resetTarget, setResetTarget] = useState<AdminUser | null>(null);
  const [resetPw, setResetPw] = useState("");
  const [resetErr, setResetErr] = useState("");
  const [resetBusy, setResetBusy] = useState(false);
  // 租户成员抽屉
  const [membersFor, setMembersFor] = useState<AdminTenant | null>(null);
  const [members, setMembers] = useState<MemberView[]>([]);
  const [membersErr, setMembersErr] = useState("");
  // 用户新增/编辑
  const [userFormOpen, setUserFormOpen] = useState(false);
  const [editUser, setEditUser] = useState<AdminUser | null>(null);
  const [userEmail, setUserEmail] = useState("");
  const [userName, setUserName] = useState("");
  const [userPassword, setUserPassword] = useState("");
  const [userTenantID, setUserTenantID] = useState("");
  const [userRole, setUserRole] = useState(ROLE_ANALYST);
  const [userPlatformAdmin, setUserPlatformAdmin] = useState(false);
  const [userFormErr, setUserFormErr] = useState("");
  const [userFormBusy, setUserFormBusy] = useState(false);
  // 租户新增/编辑
  const [tenantFormOpen, setTenantFormOpen] = useState(false);
  const [editTenant, setEditTenant] = useState<AdminTenant | null>(null);
  const [tenantName, setTenantName] = useState("");
  const [tenantSlug, setTenantSlug] = useState("");
  const [ownerEmail, setOwnerEmail] = useState("");
  const [ownerName, setOwnerName] = useState("");
  const [ownerPassword, setOwnerPassword] = useState("");
  const [tenantFormErr, setTenantFormErr] = useState("");
  const [tenantFormBusy, setTenantFormBusy] = useState(false);
  // 指定租户添加成员
  const [memberAddOpen, setMemberAddOpen] = useState(false);
  const [memberMode, setMemberMode] = useState<"existing" | "new">("existing");
  const [memberEmail, setMemberEmail] = useState("");
  const [memberName, setMemberName] = useState("");
  const [memberPassword, setMemberPassword] = useState("");
  const [memberRole, setMemberRole] = useState(ROLE_ANALYST);
  const [memberBusy, setMemberBusy] = useState(false);
  // 统一危险操作确认
  const [confirmAction, setConfirmAction] = useState<{
    title: string;
    body: string;
    run: () => Promise<void>;
  } | null>(null);
  // 用户行「⋯」菜单（position:fixed 锚定触发按钮，避免被表格 overflow 裁切）
  const [menuFor, setMenuFor] = useState<{ user_id: string; right: number; top: number } | null>(
    null,
  );

  const load = useCallback(async () => {
    if (!isPlatformAdmin) return; // 平台功能：租户用户不拉取平台数据（避免无谓 403）
    setError("");
    try {
      const [u, t] = await Promise.all([platformUsers(), platformTenants()]);
      setUsers(u.items || []);
      setTenants(t.items || []);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, [isPlatformAdmin]);
  useEffect(() => {
    void load();
  }, [load]);

  async function setStatus(u: AdminUser, status: string) {
    setError("");
    setMsg("");
    try {
      await platformUserStatus(u.user_id, status);
      setUsers((xs) => xs.map((x) => (x.user_id === u.user_id ? { ...x, status } : x)));
      setMsg(`已${status === "disabled" ? "禁用" : "启用"} ${u.display_name}`);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  async function setPlatformAdmin(u: AdminUser, on: boolean) {
    setError("");
    setMsg("");
    try {
      await platformUserSetAdmin(u.user_id, on);
      setUsers((xs) => xs.map((x) => (x.user_id === u.user_id ? { ...x, platform_admin: on } : x)));
      setMsg(on ? `已授予平台管理员：${u.display_name}` : `已撤销平台管理员：${u.display_name}`);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  function openReset(u: AdminUser) {
    setResetErr("");
    setResetPw("");
    setResetTarget(u);
  }
  async function doReset() {
    if (!resetTarget) return;
    const pwErr = passwordError(resetPw);
    if (pwErr) {
      setResetErr(pwErr);
      return;
    }
    setResetBusy(true);
    setResetErr("");
    try {
      await platformUserResetPassword(resetTarget.user_id, resetPw);
      setMsg(`已重置 ${resetTarget.display_name} 的密码（旧会话已失效）`);
      setResetTarget(null);
    } catch (e) {
      setResetErr(e instanceof Error ? e.message : String(e));
    } finally {
      setResetBusy(false);
    }
  }

  async function openMembers(t: AdminTenant) {
    setMembersFor(t);
    setMembersErr("");
    setMembers([]);
    setMemberPage(0);
    try {
      const r = await platformTenantMembers(t.tenant_id);
      setMembers(r.items || []);
    } catch (e) {
      setMembersErr(e instanceof Error ? e.message : String(e));
    }
  }

  function openCreateUser() {
    setEditUser(null);
    setUserEmail("");
    setUserName("");
    setUserPassword("");
    setUserTenantID(tenants.find((t) => t.status === "active")?.tenant_id || "");
    setUserRole(ROLE_ANALYST);
    setUserPlatformAdmin(false);
    setUserFormErr("");
    setUserFormOpen(true);
  }

  function openEditUser(u: AdminUser) {
    setEditUser(u);
    setUserEmail(u.email);
    setUserName(u.display_name);
    setUserFormErr("");
    setUserFormOpen(true);
  }

  async function saveUser() {
    if (!userEmail.trim() || !userName.trim()) {
      setUserFormErr("邮箱和姓名不能为空");
      return;
    }
    if (!editUser) {
      const pwErr = passwordError(userPassword);
      if (pwErr) {
        setUserFormErr(pwErr);
        return;
      }
      if (!userTenantID) {
        setUserFormErr("请选择所属租户");
        return;
      }
    }
    setUserFormBusy(true);
    setUserFormErr("");
    try {
      if (editUser) {
        await platformUserUpdate(editUser.user_id, userEmail.trim(), userName.trim());
        setMsg(`已更新用户：${userName.trim()}`);
      } else {
        await platformUserCreate({
          email: userEmail.trim(),
          display_name: userName.trim(),
          password: userPassword,
          tenant_id: userTenantID,
          role: userRole,
          platform_admin: userPlatformAdmin,
        });
        setMsg(`已创建用户并加入租户：${userName.trim()}`);
      }
      setUserFormOpen(false);
      await load();
    } catch (e) {
      setUserFormErr(e instanceof Error ? e.message : String(e));
    } finally {
      setUserFormBusy(false);
    }
  }

  function confirmDeleteUser(u: AdminUser) {
    setConfirmAction({
      title: "停用这个用户？",
      body: `将停用 ${u.display_name}（${u.email}）并废止其全部会话；身份和历史数据会保留。`,
      run: async () => {
        await platformUserDelete(u.user_id);
        setMsg(`已停用用户：${u.display_name}`);
        await load();
      },
    });
  }

  function openCreateTenant() {
    setEditTenant(null);
    setTenantName("");
    setTenantSlug("");
    setOwnerEmail("");
    setOwnerName("");
    setOwnerPassword("");
    setTenantFormErr("");
    setTenantFormOpen(true);
  }

  function openEditTenant(t: AdminTenant) {
    setEditTenant(t);
    setTenantName(t.name);
    setTenantSlug(t.slug);
    setTenantFormErr("");
    setTenantFormOpen(true);
  }

  async function saveTenant() {
    if (!tenantName.trim()) {
      setTenantFormErr("租户名称不能为空");
      return;
    }
    setTenantFormBusy(true);
    setTenantFormErr("");
    try {
      if (editTenant) {
        await platformTenantUpdate(editTenant.tenant_id, tenantName.trim(), tenantSlug.trim());
        setMsg(`已更新租户：${tenantName.trim()}`);
      } else {
        if (!ownerEmail.trim()) {
          setTenantFormErr("请指定租户所有者邮箱");
          return;
        }
        if (ownerPassword) {
          const pwErr = passwordError(ownerPassword);
          if (pwErr) {
            setTenantFormErr(pwErr);
            return;
          }
        }
        await platformTenantCreate({
          name: tenantName.trim(),
          slug: tenantSlug.trim() || undefined,
          owner_email: ownerEmail.trim(),
          owner_display_name: ownerName.trim() || undefined,
          owner_password: ownerPassword || undefined,
        });
        setMsg(`已创建租户：${tenantName.trim()}`);
      }
      setTenantFormOpen(false);
      await load();
    } catch (e) {
      setTenantFormErr(e instanceof Error ? e.message : String(e));
    } finally {
      setTenantFormBusy(false);
    }
  }

  async function setTenantStatus(t: AdminTenant, status: string) {
    setError("");
    try {
      await platformTenantStatus(t.tenant_id, status);
      setMsg(`已${status === "active" ? "启用" : "停用"}租户：${t.name}`);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  function confirmDeleteTenant(t: AdminTenant) {
    setConfirmAction({
      title: "删除这个租户？",
      body: `租户 ${t.name} 将被软删除并立即失效，成员不能再进入；业务数据目录和审计历史不会被物理删除。`,
      run: async () => {
        await platformTenantDelete(t.tenant_id);
        setMsg(`已软删除租户：${t.name}`);
        await load();
      },
    });
  }

  function openMemberAdd() {
    setMemberMode("existing");
    setMemberEmail("");
    setMemberName("");
    setMemberPassword("");
    setMemberRole(ROLE_ANALYST);
    setMembersErr("");
    setMemberAddOpen(true);
  }

  async function addTenantMember() {
    if (!membersFor || !memberEmail.trim()) return;
    if (memberMode === "new") {
      if (!memberName.trim()) {
        setMembersErr("新建用户时姓名不能为空");
        return;
      }
      const pwErr = passwordError(memberPassword);
      if (pwErr) {
        setMembersErr(pwErr);
        return;
      }
    }
    setMemberBusy(true);
    setMembersErr("");
    try {
      await platformTenantMemberAdd(membersFor.tenant_id, {
        email: memberEmail.trim(),
        display_name: memberMode === "new" ? memberName.trim() : undefined,
        password: memberMode === "new" ? memberPassword : undefined,
        role: memberRole,
      });
      setMemberAddOpen(false);
      await openMembers(membersFor);
      await load();
    } catch (e) {
      setMembersErr(e instanceof Error ? e.message : String(e));
    } finally {
      setMemberBusy(false);
    }
  }

  async function changeTenantMemberRole(m: MemberView, role: string) {
    if (!membersFor) return;
    setMembersErr("");
    try {
      await platformTenantMemberRole(membersFor.tenant_id, m.user_id, role);
      setMembers((xs) => xs.map((x) => (x.user_id === m.user_id ? { ...x, role } : x)));
    } catch (e) {
      setMembersErr(e instanceof Error ? e.message : String(e));
    }
  }

  function confirmRemoveTenantMember(m: MemberView) {
    if (!membersFor) return;
    const tenant = membersFor;
    setConfirmAction({
      title: "移除租户成员？",
      body: `将把 ${m.display_name} 从 ${tenant.name} 移除；其当前租户会话会立即失效。`,
      run: async () => {
        await platformTenantMemberRemove(tenant.tenant_id, m.user_id);
        await openMembers(tenant);
        await load();
      },
    });
  }

  // 用户行「⋯」菜单：position:fixed 锚定按钮右缘（复用 AppLayout 会话菜单模式）。
  function toggleMenu(e: MouseEvent<HTMLButtonElement>, u: AdminUser) {
    e.stopPropagation();
    if (menuFor?.user_id === u.user_id) {
      setMenuFor(null);
      return;
    }
    const r = (e.currentTarget as HTMLElement).getBoundingClientRect();
    setMenuFor({ user_id: u.user_id, right: window.innerWidth - r.right + 2, top: r.bottom + 2 });
  }
  // 点击菜单外部关闭（菜单内部 mousedown 不关——否则按钮 click 前菜单已被卸载）
  useEffect(() => {
    if (!menuFor) return;
    const onDocDown = (e: globalThis.MouseEvent) => {
      if ((e.target as HTMLElement).closest?.(".admin-user-menu")) return;
      setMenuFor(null);
    };
    document.addEventListener("mousedown", onDocDown);
    return () => document.removeEventListener("mousedown", onDocDown);
  }, [menuFor]);

  // 平台功能：非平台管理员直接访问也拒绝展示（导航已隐藏，双保险）
  if (!isPlatformAdmin) {
    return (
      <div className="page">
        <div className="panel">
          <h3>用户管理</h3>
          <p className="muted">用户管理为平台管理功能（§4.2），仅平台管理员可查看。</p>
        </div>
      </div>
    );
  }

  // 客户端分页：slice 当前页（页码越界时收敛到末页）
  const totalUserPages = Math.max(1, Math.ceil(users.length / PAGE_SIZE));
  const safeUserPage = Math.min(userPage, totalUserPages - 1);
  const pageUsers = users.slice(safeUserPage * PAGE_SIZE, safeUserPage * PAGE_SIZE + PAGE_SIZE);
  const totalTenantPages = Math.max(1, Math.ceil(tenants.length / PAGE_SIZE));
  const safeTenantPage = Math.min(tenantPage, totalTenantPages - 1);
  const pageTenants = tenants.slice(
    safeTenantPage * PAGE_SIZE,
    safeTenantPage * PAGE_SIZE + PAGE_SIZE,
  );
  const totalMemberPages = Math.max(1, Math.ceil(members.length / PAGE_SIZE));
  const safeMemberPage = Math.min(memberPage, totalMemberPages - 1);
  const pageMembers = members.slice(
    safeMemberPage * PAGE_SIZE,
    safeMemberPage * PAGE_SIZE + PAGE_SIZE,
  );

  return (
    <div className="page">
      <div className="page-head">
        <h2>用户管理</h2>
        <span className="crumb">ADMIN</span>
        {msg && <span className="admin-msg">{msg}</span>}
        <div className="page-actions">
          <button className="btn sm" onClick={tab === "users" ? openCreateUser : openCreateTenant}>
            {tab === "users" ? "+ 新建用户" : "+ 新建租户"}
          </button>
        </div>
      </div>
      <div className="page-sub">
        平台用户、租户与成员关系全生命周期管理。删除采用停用/软删除，历史业务数据保留。
      </div>

      <div className="tab-bar">
        <button
          className={`tab${tab === "users" ? " active" : ""}`}
          onClick={() => setTab("users")}
        >
          用户
        </button>
        <button
          className={`tab${tab === "tenants" ? " active" : ""}`}
          onClick={() => setTab("tenants")}
        >
          租户
        </button>
      </div>

      {error && <div className="error">! {error}</div>}
      {loading && <div className="loading">加载中…</div>}

      {tab === "users" && !loading && (
        <div className="panel flush">
          <table>
            <thead>
              <tr>
                <th>邮箱</th>
                <th>姓名</th>
                <th>状态</th>
                <th>平台管理员</th>
                <th>所属租户</th>
                <th>最近登录</th>
                <th style={{ width: 44 }}>操作</th>
              </tr>
            </thead>
            <tbody>
              {users.length === 0 && (
                <tr>
                  <td colSpan={7} className="empty">
                    暂无用户
                  </td>
                </tr>
              )}
              {pageUsers.map((u) => {
                const isSelf = u.user_id === user?.user_id;
                return (
                  <tr key={u.user_id}>
                    <td className="mono">{u.email}</td>
                    <td>{u.display_name}</td>
                    <td>
                      <span className={`badge ${u.status === "active" ? "verified" : "reject"}`}>
                        {USER_STATUS_LABEL[u.status] || u.status}
                      </span>
                    </td>
                    <td>{u.platform_admin ? <Icon name="check" size={13} /> : "—"}</td>
                    <td>
                      {u.memberships.length === 0 ? (
                        <span className="muted">—</span>
                      ) : (
                        u.memberships.map((m, i) => (
                          <span
                            key={i}
                            className="admin-tenant-chip"
                            title={`${m.tenant_name} · 租户 ID ${m.tenant_id} · slug ${m.slug}`}
                          >
                            {m.tenant_name}
                            <i className="tenant-id-chip">#{m.tenant_id.slice(0, 8)}</i>
                            <em>{ROLE_LABEL[m.role] || m.role}</em>
                          </span>
                        ))
                      )}
                    </td>
                    <td className="muted">
                      {u.last_login_at ? new Date(u.last_login_at).toLocaleString() : "—"}
                    </td>
                    <td>
                      <button
                        className="btn ghost sm row-menu"
                        title="操作"
                        aria-label={`操作 ${u.display_name}`}
                        onClick={(e) => toggleMenu(e, u)}
                      >
                        <Icon d={DOTS_PATH} size={14} />
                      </button>
                      {menuFor?.user_id === u.user_id && (
                        <div
                          className="conv-menu admin-user-menu"
                          style={{ right: menuFor.right, top: menuFor.top }}
                          onClick={(e) => e.stopPropagation()}
                        >
                          <button
                            className="conv-menu-item"
                            onClick={() => {
                              setMenuFor(null);
                              openEditUser(u);
                            }}
                          >
                            编辑资料
                          </button>
                          {u.status === "active" ? (
                            <button
                              className="conv-menu-item"
                              disabled={isSelf}
                              title={isSelf ? "不能禁用自己" : ""}
                              onClick={() => {
                                setMenuFor(null);
                                setStatus(u, "disabled");
                              }}
                            >
                              禁用
                            </button>
                          ) : (
                            <button
                              className="conv-menu-item"
                              onClick={() => {
                                setMenuFor(null);
                                setStatus(u, "active");
                              }}
                            >
                              启用
                            </button>
                          )}
                          <button
                            className="conv-menu-item"
                            onClick={() => {
                              setMenuFor(null);
                              openReset(u);
                            }}
                          >
                            重置密码
                          </button>
                          <button
                            className="conv-menu-item"
                            disabled={isSelf}
                            title={isSelf ? "不能撤销自己的平台管理员" : ""}
                            onClick={() => {
                              setMenuFor(null);
                              setPlatformAdmin(u, !u.platform_admin);
                            }}
                          >
                            {u.platform_admin ? "撤销平台管理员" : "授予平台管理员"}
                          </button>
                          <button
                            className="conv-menu-item danger-text"
                            disabled={isSelf || u.status !== "active"}
                            onClick={() => {
                              setMenuFor(null);
                              confirmDeleteUser(u);
                            }}
                          >
                            停用（软删除）
                          </button>
                        </div>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
          <Pager page={safeUserPage} totalPages={totalUserPages} onChange={setUserPage} />
        </div>
      )}

      {tab === "tenants" && !loading && (
        <div className="panel flush">
          <table>
            <thead>
              <tr>
                <th>名称</th>
                <th>租户 ID</th>
                <th>Slug</th>
                <th>状态</th>
                <th>成员数</th>
                <th>创建人</th>
                <th>创建时间</th>
                <th style={{ width: 270 }}>操作</th>
              </tr>
            </thead>
            <tbody>
              {tenants.length === 0 && (
                <tr>
                  <td colSpan={8} className="empty">
                    暂无租户
                  </td>
                </tr>
              )}
              {pageTenants.map((t) => (
                <tr key={t.tenant_id}>
                  <td>{t.name}</td>
                  <td className="mono" title={`租户 ID：${t.tenant_id}`}>
                    {t.tenant_id.slice(0, 8)}
                  </td>
                  <td className="mono">{t.slug}</td>
                  <td>
                    <span
                      className={`badge ${t.status === "active" ? "verified" : t.status === "provisioning" ? "pending_review" : "reject"}`}
                    >
                      {TENANT_STATUS_LABEL[t.status] || t.status}
                    </span>
                  </td>
                  <td>{t.member_count}</td>
                  <td className="muted">{t.created_by || "—"}</td>
                  <td className="muted">{new Date(t.created_at).toLocaleString()}</td>
                  <td>
                    <button className="btn ghost sm" onClick={() => openMembers(t)}>
                      管理成员
                    </button>
                    <button
                      className="btn ghost sm"
                      onClick={() => openEditTenant(t)}
                      disabled={t.status === "deleted"}
                    >
                      编辑
                    </button>
                    {t.status === "active" ? (
                      <button
                        className="btn ghost sm"
                        onClick={() => void setTenantStatus(t, "disabled")}
                      >
                        停用
                      </button>
                    ) : t.status === "disabled" || t.status === "provisioning_failed" ? (
                      <button
                        className="btn ghost sm"
                        onClick={() => void setTenantStatus(t, "active")}
                      >
                        启用
                      </button>
                    ) : null}
                    <button
                      className="btn ghost sm danger-text"
                      disabled={t.status === "deleted"}
                      onClick={() => confirmDeleteTenant(t)}
                    >
                      删除
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          <Pager page={safeTenantPage} totalPages={totalTenantPages} onChange={setTenantPage} />
        </div>
      )}

      {/* 新建/编辑用户 */}
      {userFormOpen && (
        <div
          className="cd-mask"
          onClick={() => !userFormBusy && setUserFormOpen(false)}
          role="presentation"
        >
          <div
            className="cd-box"
            role="dialog"
            aria-modal="true"
            aria-label={editUser ? "编辑用户" : "新建用户"}
            onClick={(e) => e.stopPropagation()}
          >
            <div className="cd-title">{editUser ? "编辑用户" : "新建用户"}</div>
            <div className="cd-body pw-form">
              <label>
                邮箱
                <input value={userEmail} onChange={(e) => setUserEmail(e.target.value)} />
              </label>
              <label>
                姓名
                <input value={userName} onChange={(e) => setUserName(e.target.value)} />
              </label>
              {!editUser && (
                <>
                  <label>
                    初始密码
                    <PasswordInput
                      value={userPassword}
                      maxLength={PASSWORD_MAX}
                      autoComplete="new-password"
                      onChange={(e) => setUserPassword(e.target.value)}
                    />
                  </label>
                  <label>
                    所属租户
                    <select value={userTenantID} onChange={(e) => setUserTenantID(e.target.value)}>
                      <option value="">请选择</option>
                      {tenants
                        .filter((t) => t.status === "active")
                        .map((t) => (
                          <option key={t.tenant_id} value={t.tenant_id}>
                            {t.name}
                          </option>
                        ))}
                    </select>
                  </label>
                  <label>
                    租户角色
                    <select value={userRole} onChange={(e) => setUserRole(e.target.value)}>
                      <option value={ROLE_OWNER}>所有者</option>
                      <option value={ROLE_ADMIN}>管理员</option>
                      <option value={ROLE_ANALYST}>分析人员</option>
                      <option value={ROLE_REVIEWER}>审核人员</option>
                    </select>
                  </label>
                  <label className="check-row">
                    <input
                      type="checkbox"
                      checked={userPlatformAdmin}
                      onChange={(e) => setUserPlatformAdmin(e.target.checked)}
                    />{" "}
                    同时授予平台管理员
                  </label>
                </>
              )}
              {userFormErr && <div className="error">! {userFormErr}</div>}
            </div>
            <div className="cd-btns">
              <button
                className="btn ghost sm"
                disabled={userFormBusy}
                onClick={() => setUserFormOpen(false)}
              >
                取消
              </button>
              <button className="btn sm" disabled={userFormBusy} onClick={saveUser}>
                {userFormBusy ? "保存中…" : "保存"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 新建/编辑租户 */}
      {tenantFormOpen && (
        <div
          className="cd-mask"
          onClick={() => !tenantFormBusy && setTenantFormOpen(false)}
          role="presentation"
        >
          <div
            className="cd-box"
            role="dialog"
            aria-modal="true"
            aria-label={editTenant ? "编辑租户" : "新建租户"}
            onClick={(e) => e.stopPropagation()}
          >
            <div className="cd-title">{editTenant ? "编辑租户" : "新建租户"}</div>
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
                Slug（留空自动生成）
                <input
                  value={tenantSlug}
                  maxLength={40}
                  onChange={(e) => setTenantSlug(e.target.value.toLowerCase())}
                />
              </label>
              {!editTenant && (
                <>
                  <label>
                    所有者邮箱
                    <input value={ownerEmail} onChange={(e) => setOwnerEmail(e.target.value)} />
                  </label>
                  <div className="muted">
                    若邮箱已注册，将直接加入此租户；若是新邮箱，请继续填写姓名和初始密码。
                  </div>
                  <label>
                    新所有者姓名（已有账号可留空）
                    <input value={ownerName} onChange={(e) => setOwnerName(e.target.value)} />
                  </label>
                  <label>
                    新所有者初始密码（已有账号可留空）
                    <PasswordInput
                      value={ownerPassword}
                      maxLength={PASSWORD_MAX}
                      autoComplete="new-password"
                      onChange={(e) => setOwnerPassword(e.target.value)}
                    />
                  </label>
                </>
              )}
              {tenantFormErr && <div className="error">! {tenantFormErr}</div>}
            </div>
            <div className="cd-btns">
              <button
                className="btn ghost sm"
                disabled={tenantFormBusy}
                onClick={() => setTenantFormOpen(false)}
              >
                取消
              </button>
              <button className="btn sm" disabled={tenantFormBusy} onClick={saveTenant}>
                {tenantFormBusy ? "保存中…" : "保存"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 平台指定租户添加成员 */}
      {memberAddOpen && membersFor && (
        <div
          className="cd-mask top"
          onClick={() => !memberBusy && setMemberAddOpen(false)}
          role="presentation"
        >
          <div
            className="cd-box"
            role="dialog"
            aria-modal="true"
            aria-label="添加租户成员"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="cd-title">向 {membersFor.name} 添加成员</div>
            <div className="cd-body pw-form">
              <div className="tab-bar compact">
                <button
                  className={`tab${memberMode === "existing" ? " active" : ""}`}
                  onClick={() => setMemberMode("existing")}
                >
                  已有用户
                </button>
                <button
                  className={`tab${memberMode === "new" ? " active" : ""}`}
                  onClick={() => setMemberMode("new")}
                >
                  新建用户
                </button>
              </div>
              <label>
                邮箱
                <input value={memberEmail} onChange={(e) => setMemberEmail(e.target.value)} />
              </label>
              {memberMode === "new" && (
                <>
                  <label>
                    姓名
                    <input value={memberName} onChange={(e) => setMemberName(e.target.value)} />
                  </label>
                  <label>
                    初始密码
                    <PasswordInput
                      value={memberPassword}
                      maxLength={PASSWORD_MAX}
                      autoComplete="new-password"
                      onChange={(e) => setMemberPassword(e.target.value)}
                    />
                  </label>
                </>
              )}
              <label>
                角色
                <select value={memberRole} onChange={(e) => setMemberRole(e.target.value)}>
                  <option value={ROLE_OWNER}>所有者</option>
                  <option value={ROLE_ADMIN}>管理员</option>
                  <option value={ROLE_ANALYST}>分析人员</option>
                  <option value={ROLE_REVIEWER}>审核人员</option>
                </select>
              </label>
              {membersErr && <div className="error">! {membersErr}</div>}
            </div>
            <div className="cd-btns">
              <button
                className="btn ghost sm"
                disabled={memberBusy}
                onClick={() => setMemberAddOpen(false)}
              >
                取消
              </button>
              <button className="btn sm" disabled={memberBusy} onClick={addTenantMember}>
                {memberBusy ? "添加中…" : "添加"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 重置密码弹窗 */}
      {resetTarget && (
        <div
          className="cd-mask"
          onClick={() => !resetBusy && setResetTarget(null)}
          role="presentation"
        >
          <div
            className="cd-box"
            role="dialog"
            aria-modal="true"
            aria-label="重置密码"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="cd-title">重置密码</div>
            <div className="cd-body">
              <div style={{ color: "var(--text)", marginBottom: 10 }}>
                {resetTarget.display_name}（{resetTarget.email}）——
                新密码将立即生效，旧会话全部失效。
              </div>
              <label>
                新密码（8~16 字符，需含大小写字母、数字、符号中至少三种）
                <PasswordInput
                  value={resetPw}
                  maxLength={PASSWORD_MAX}
                  autoComplete="new-password"
                  onChange={(e) => setResetPw(e.target.value)}
                />
              </label>
              {resetErr && (
                <div className="error" style={{ marginTop: 8 }}>
                  ! {resetErr}
                </div>
              )}
            </div>
            <div className="cd-btns">
              <button
                className="btn ghost sm"
                disabled={resetBusy}
                onClick={() => setResetTarget(null)}
              >
                取消
              </button>
              <button className="btn sm" disabled={resetBusy} onClick={doReset}>
                {resetBusy ? "提交中…" : "确认重置"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 租户成员抽屉 */}
      {membersFor && (
        <div className="cd-mask" onClick={() => setMembersFor(null)} role="presentation">
          <div
            className="cd-box admin-members-box"
            role="dialog"
            aria-modal="true"
            aria-label="租户成员"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="cd-title">
              {membersFor.name} 的成员
              <span className="admin-members-count">共 {members.length} 人</span>
              <button className="btn sm" style={{ marginLeft: "auto" }} onClick={openMemberAdd}>
                + 添加成员
              </button>
            </div>
            <div className="cd-body">
              {membersErr && <div className="error">! {membersErr}</div>}
              {members.length === 0 && !membersErr && <div className="muted">该租户暂无成员</div>}
              {pageMembers.map((m) => (
                <div key={m.user_id} className="admin-member-row">
                  <div>
                    <div className="admin-member-name">{m.display_name}</div>
                    <div className="mono faint">{m.email}</div>
                  </div>
                  <select
                    className="member-role-select"
                    value={m.role}
                    onChange={(e) => void changeTenantMemberRole(m, e.target.value)}
                  >
                    <option value={ROLE_OWNER}>所有者</option>
                    <option value={ROLE_ADMIN}>管理员</option>
                    <option value={ROLE_ANALYST}>分析人员</option>
                    <option value={ROLE_REVIEWER}>审核人员</option>
                  </select>
                  <span className={`badge ${m.status === "active" ? "verified" : "reject"}`}>
                    {USER_STATUS_LABEL[m.status] || m.status}
                  </span>
                  <button
                    className="btn ghost sm danger-text"
                    onClick={() => confirmRemoveTenantMember(m)}
                  >
                    移除
                  </button>
                </div>
              ))}
            </div>
            <Pager page={safeMemberPage} totalPages={totalMemberPages} onChange={setMemberPage} />
            <div className="cd-btns">
              <button className="btn ghost sm" onClick={() => setMembersFor(null)}>
                关闭
              </button>
            </div>
          </div>
        </div>
      )}

      <ConfirmDialog
        open={!!confirmAction}
        title={confirmAction?.title || "确认操作"}
        body={confirmAction?.body || ""}
        confirmText="确认"
        onCancel={() => setConfirmAction(null)}
        onConfirm={() => {
          const action = confirmAction;
          setConfirmAction(null);
          if (!action) return;
          void action.run().catch((e) => setError(e instanceof Error ? e.message : String(e)));
        }}
      />
    </div>
  );
}
