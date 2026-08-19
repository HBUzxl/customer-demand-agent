// AuthProvider：多租户前端身份状态（M4b）。
// 挂载即调 GET /api/auth/me 恢复登录态；维护 loading/authed/user；
// 401 统一处理由 api/client 完成（清本地敏感态 + 跳登录，保留原路由）。
// login/register 成功后写入 CSRF；logout 清本地 cda:* 键。
import { createContext, useCallback, useContext, useEffect, useState } from "react";
import type { ReactNode } from "react";
import {
  authMe,
  authLogin,
  authLogout,
  authRegister,
  authSwitchTenant,
  authChangePassword,
  setCSRFToken,
  resetUnauthFlag,
  clearLocalKeys,
} from "../api/client";
import { ROLE_PLATFORM_ADMIN } from "../types";
import type { AuthUser } from "../types";

interface AuthContextValue {
  loading: boolean;
  authed: boolean;
  user: AuthUser | null;
  isPlatformAdmin: boolean;
  // 本地键前缀：cda:<tenant_id>:<user_id>:（退出清理，租户/用户隔离）
  lsKey: (base: string) => string;
  login: (email: string, password: string) => Promise<void>;
  register: (
    orgName: string,
    displayName: string,
    email: string,
    password: string,
  ) => Promise<void>;
  logout: () => Promise<void>;
  changePassword: (oldPassword: string, newPassword: string) => Promise<void>;
  switchTenant: (tenantId: string) => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<AuthUser | null>(null);
  const [loading, setLoading] = useState(true);

  // 挂载恢复登录态：401（未登录/过期）→ authed=false；后端不可达同样视为未登录。
  useEffect(() => {
    let alive = true;
    authMe()
      .then((u) => {
        if (!alive) return;
        setUser(u);
        setCSRFToken(u.csrf);
        resetUnauthFlag();
      })
      .catch(() => {
        /* 未登录 */
      })
      .finally(() => {
        if (alive) setLoading(false);
      });
    return () => {
      alive = false;
    };
  }, []);

  async function login(email: string, password: string) {
    const u = await authLogin(email, password);
    setUser(u);
    setCSRFToken(u.csrf);
    resetUnauthFlag();
  }

  async function register(orgName: string, displayName: string, email: string, password: string) {
    const u = await authRegister(orgName, displayName, email, password);
    setUser(u);
    setCSRFToken(u.csrf);
    resetUnauthFlag();
  }

  async function logout() {
    try {
      await authLogout();
    } catch {
      /* 幂等：本地态照清 */
    }
    setUser(null);
    setCSRFToken("");
    clearLocalKeys();
    resetUnauthFlag();
  }

  async function changePassword(oldPassword: string, newPassword: string) {
    await authChangePassword(oldPassword, newPassword);
    // 改密会话仍有效；重新拉取 me 确认登录态并恢复当前会话的稳定 CSRF。
    const u = await authMe();
    setUser(u);
    setCSRFToken(u.csrf);
  }

  async function switchTenant(tenantId: string) {
    if (!user || tenantId === user.tenant_id) return;
    const u = await authSwitchTenant(tenantId);
    setUser(u);
    setCSRFToken(u.csrf);
    resetUnauthFlag();
    // 终止旧租户的 SSE/轮询与页面内状态；新页面只会使用新 TenantScope 和本租户本地键。
    window.location.assign("/analyze");
  }

  const isPlatformAdmin = !!user?.roles.includes(ROLE_PLATFORM_ADMIN);
  // 稳定引用：user 不变则函数不变（避免下游 effect 因每次渲染新函数而重复触发）
  const lsKey = useCallback(
    (base: string) => (user ? `cda:${user.tenant_id}:${user.user_id}:${base}` : `cda:${base}`),
    [user],
  );

  const value: AuthContextValue = {
    loading,
    authed: !!user,
    user,
    isPlatformAdmin,
    lsKey,
    login,
    register,
    logout,
    changePassword,
    switchTenant,
  };
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

// Context provider 与 hook 必须共享同一个模块级 AuthContext。
// eslint-disable-next-line react-refresh/only-export-components
export function useAuth(): AuthContextValue {
  const v = useContext(AuthContext);
  if (!v) throw new Error("useAuth 必须在 <AuthProvider> 内使用");
  return v;
}
