// 登录页（M4b）。未登录访问任何业务路由会被 ProtectedRoute 带到 /login，
// 并携带 from（state 或 ?from= 查询参数）——登录成功后跳回原路由。
import { useState } from "react";
import type { FormEvent } from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthProvider";
import PasswordInput from "../components/PasswordInput";

export default function Login() {
  const { login } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);

  const from =
    (location.state as { from?: string } | null)?.from ||
    new URLSearchParams(location.search).get("from") ||
    "/analyze";

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setErr("");
    setBusy(true);
    try {
      await login(email.trim(), password);
      navigate(from, { replace: true });
    } catch (e2) {
      setErr(e2 instanceof Error ? e2.message : String(e2));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="auth-wrap">
      <form className="auth-card" onSubmit={onSubmit}>
        <div className="auth-brand">
          <div className="logo">长</div>
          <div>
            <div className="auth-title">需求分析智能体</div>
            <div className="auth-sub">· 长亭 · 多租户工作空间</div>
          </div>
        </div>
        {err && <div className="error">! {err}</div>}
        <label className="auth-field">
          <span>邮箱</span>
          <input
            type="email"
            value={email}
            autoFocus
            autoComplete="username"
            placeholder="you@example.com"
            onChange={(e) => setEmail(e.target.value)}
            required
          />
        </label>
        <label className="auth-field">
          <span>密码</span>
          <PasswordInput
            value={password}
            autoComplete="current-password"
            onChange={(e) => setPassword(e.target.value)}
            required
          />
        </label>
        <button className="btn primary" disabled={busy} type="submit">
          {busy ? "登录中…" : "登录"}
        </button>
        <div className="auth-alt">
          没有账号？<Link to="/register">注册新工作空间</Link>
        </div>
      </form>
    </div>
  );
}
