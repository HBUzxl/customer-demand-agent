// 注册页（M4b）：注册即创建租户 + owner Membership 并自动登录（后端发 Cookie）。
import { useState } from "react";
import type { FormEvent } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthProvider";
import PasswordInput from "../components/PasswordInput";
import { passwordError, PASSWORD_MAX } from "../lib/password";

export default function Register() {
  const { register } = useAuth();
  const navigate = useNavigate();
  const [org, setOrg] = useState("");
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [pw, setPw] = useState("");
  const [pw2, setPw2] = useState("");
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setErr("");
    if (pw !== pw2) {
      setErr("两次输入的密码不一致");
      return;
    }
    const pwErr = passwordError(pw);
    if (pwErr) {
      setErr(pwErr);
      return;
    }
    setBusy(true);
    try {
      await register(org.trim(), name.trim(), email.trim(), pw);
      navigate("/analyze", { replace: true });
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
            <div className="auth-title">创建新工作空间</div>
            <div className="auth-sub">注册即建立独立租户 · 数据互不可见</div>
          </div>
        </div>
        {err && <div className="error">! {err}</div>}
        <label className="auth-field">
          <span>组织 / 工作空间名称</span>
          <input
            value={org}
            autoFocus
            placeholder="如：某集团安全部"
            onChange={(e) => setOrg(e.target.value)}
            required
          />
        </label>
        <label className="auth-field">
          <span>姓名</span>
          <input
            value={name}
            placeholder="如：张三"
            onChange={(e) => setName(e.target.value)}
            required
          />
        </label>
        <label className="auth-field">
          <span>邮箱</span>
          <input
            type="email"
            value={email}
            autoComplete="username"
            placeholder="you@example.com"
            onChange={(e) => setEmail(e.target.value)}
            required
          />
        </label>
        <label className="auth-field">
          <span>密码（8~16 字符，需含大小写字母、数字、符号中至少三种）</span>
          <PasswordInput
            value={pw}
            maxLength={PASSWORD_MAX}
            autoComplete="new-password"
            onChange={(e) => setPw(e.target.value)}
            required
          />
        </label>
        <label className="auth-field">
          <span>确认密码</span>
          <PasswordInput
            value={pw2}
            autoComplete="new-password"
            onChange={(e) => setPw2(e.target.value)}
            required
          />
        </label>
        <button className="btn primary" disabled={busy} type="submit">
          {busy ? "创建中…" : "创建并进入"}
        </button>
        <div className="auth-alt">
          已有账号？<Link to="/login">返回登录</Link>
        </div>
      </form>
    </div>
  );
}
