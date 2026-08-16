import { createBrowserRouter, redirect } from "react-router-dom";
import { Suspense, lazy } from "react";
import AppLayout from "./layouts/AppLayout";

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
 * 路由树（每个功能一个具名路径）：
 *   /dashboard            商机面板（高价值线索 + 一键 AI 分析交接）
 *   /analyze              需求分析（新对话）
 *   /analyze/:sessionId   继续某对话（URL 可定位 / 书签 / 恢复）
 *   /history              对话列表
 *   /history/:id          回放（只读完整轨迹）
 *   /memory               记忆库列表
 *   /memory/new           新建记忆
 *   /memory/:type/:title  查看 / 编辑单条记忆
 *   /settings             设置
 *   *                     404
 */
export const router = createBrowserRouter([
  {
    path: "/",
    element: <AppLayout />,
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
      { path: "settings", element: withSuspense(Settings) },
      { path: "*", element: withSuspense(NotFound) },
    ],
  },
]);
