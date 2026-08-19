// ProtectedRoute：未登录跳 /login（保留安全原路由 from 参数），登录中显示载入。
import { Navigate, useLocation } from "react-router-dom";
import type { ReactNode } from "react";
import { useAuth } from "./AuthProvider";

export default function ProtectedRoute({ children }: { children: ReactNode }) {
  const { loading, authed } = useAuth();
  const location = useLocation();
  if (loading) {
    return (
      <div className="page">
        <div className="loading">载入中…</div>
      </div>
    );
  }
  if (!authed) {
    const from = location.pathname + location.search;
    return <Navigate to="/login" replace state={{ from }} />;
  }
  return <>{children}</>;
}
