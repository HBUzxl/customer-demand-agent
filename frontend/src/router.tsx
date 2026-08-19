/* eslint-disable react-refresh/only-export-components -- 路由配置模块持有 lazy 组件引用，不是 Fast Refresh 叶组件 */
import { createBrowserRouter, redirect } from "react-router-dom";
import { Suspense, lazy } from "react";
import AppLayout from "./layouts/AppLayout";
import ProtectedRoute from "./auth/ProtectedRoute";
import Login from "./pages/Login";
import Register from "./pages/Register";

// 路由级懒加载（代码分割）
const Conversation = lazy(() => import("./pages/Conversation"));
const Dashboard = lazy(() => import("./pages/Dashboard"));
const History = lazy(() => import("./pages/History"));
const Replay = lazy(() => import("./pages/Replay"));
const MemoryList = lazy(() => import("./pages/MemoryList"));
const MemoryEditor = lazy(() => import("./pages/MemoryEditor"));
const MemoryDetail = lazy(() => import("./pages/MemoryDetail"));
const Observe = lazy(() => import("./pages/Observe"));
const Settings = lazy(() => import("./pages/Settings"));
const AdminUsers = lazy(() => import("./pages/AdminUsers"));
const Members = lazy(() => import("./pages/Members"));
const NotFound = lazy(() => import("./pages/NotFound"));

const Fallback = () => (
  <div className="page">
    <div className="loading">载入中…</div>
  </div>
);

// 用 Suspense 包裹懒加载页面
const withSuspense = (
  Comp: React.LazyExoticComponent<React.ComponentType<Record<string, never>>>,
) => (
  <Suspense fallback={<Fallback />}>
    <Comp />
  </Suspense>
);

/**
 * 路由树（每个功能一个具名路径，M4b 起业务路由包 ProtectedRoute）：
 *   /login /register       认证页（AppLayout 之外，无需登录）
 *   /dashboard            商机面板（高价值线索 + 一键 AI 分析交接）
 *   /analyze              需求分析（新对话）
 *   /analyze/:sessionId   继续某对话（URL 可定位 / 书签 / 恢复）
 *   /history              对话列表
 *   /history/:id          回放（只读完整轨迹）
 *   /memory               记忆库列表
 *   /memory/new           新建记忆
 *   /memory/:type/:title  查看 / 编辑单条记忆
 *   /observe              观测台（平台管理员）
 *   /admin/users          用户管理（平台管理员）
 *   /members              成员管理（租户 owner/admin）
 *   /settings             设置（模型/路由平台管理员；租户安全视图）
 *   *                     404
 */
export const router = createBrowserRouter([
  { path: "/login", element: <Login /> },
  { path: "/register", element: <Register /> },
  {
    path: "/",
    element: (
      <ProtectedRoute>
        <AppLayout />
      </ProtectedRoute>
    ),
    children: [
      { index: true, loader: () => redirect("/analyze") },
      { path: "dashboard", element: withSuspense(Dashboard) },
      { path: "analyze", element: withSuspense(Conversation) },
      { path: "analyze/:sessionId", element: withSuspense(Conversation) },
      { path: "c/:sessionId", loader: ({ params }) => redirect(`/analyze/${params.sessionId}`) },
      { path: "history", element: withSuspense(History) },
      { path: "history/:id", element: withSuspense(Replay) },
      {
        path: "memory",
        children: [
          { index: true, element: withSuspense(MemoryList) },
          { path: "new", element: withSuspense(MemoryEditor) },
          { path: ":type/:title", element: withSuspense(MemoryDetail) },
        ],
      },
      { path: "observe", element: withSuspense(Observe) },
      { path: "admin/users", element: withSuspense(AdminUsers) },
      { path: "members", element: withSuspense(Members) },
      { path: "settings", element: withSuspense(Settings) },
      { path: "*", element: withSuspense(NotFound) },
    ],
  },
]);
