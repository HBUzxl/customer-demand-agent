import { useCallback, useEffect, useRef, useState } from "react";
import {
  observeHealth,
  observeLLMAudit,
  observeTasks,
  subscribePlatformLogs,
  taskLint,
} from "../api/client";
import { useAuth } from "../auth/AuthProvider";
import Icon from "../components/Icon";

/**
 * 观测台（C2）：后台任务 + 日志 tail（SSE）+ LLM 调用审计 + 系统健康。
 * 「后端在干什么」的统一视图。多租户：观测台为平台功能（§7.5）——LLM 审计与
 * 原始日志走 /api/platform/**，仅平台管理员可见；租户用户由导航与页面双重隐藏。
 */

interface Task {
  id: string;
  type: string;
  detail: string;
  status: string;
  result: string;
  created_at: string;
  done_at?: string;
}

interface Audit {
  task: string;
  model: string;
  fallback: boolean;
  duration_ms: number;
  ok: boolean;
  error?: string;
  at: string;
}

interface Health {
  status: string;
  tasks: { id: string; type: string; status: string }[];
  taskCount?: number;
}

export default function Observe() {
  const { isPlatformAdmin } = useAuth();
  const [tasks, setTasks] = useState<Task[]>([]);
  const [audit, setAudit] = useState<Audit[]>([]);
  const [health, setHealth] = useState<Health | null>(null);
  const [logs, setLogs] = useState<string[]>([]);
  const [live, setLive] = useState(true);
  const logRef = useRef<HTMLDivElement>(null);

  const load = useCallback(async () => {
    if (!isPlatformAdmin) return; // 平台功能：租户用户不拉取平台数据（避免无谓 403）
    try {
      const [t, a, h] = await Promise.all([
        observeTasks<{ items?: Task[] }>(),
        observeLLMAudit<{ items?: Audit[] }>(),
        observeHealth<Health>(),
      ]);
      setTasks(t.items || []);
      setAudit(a.items || []);
      setHealth(h);
    } catch {
      /* 后端不可达 */
    }
  }, [isPlatformAdmin]);

  useEffect(() => {
    if (!isPlatformAdmin) return; // 平台功能：租户用户不拉取平台数据
    void load();
    const t = setInterval(load, 5000);
    return () => clearInterval(t);
  }, [isPlatformAdmin, load]);

  useEffect(() => {
    if (!live) return;
    if (!isPlatformAdmin) return; // 平台功能：租户用户不订阅平台日志
    return subscribePlatformLogs((line) => setLogs((ls) => [...ls.slice(-200), line]));
  }, [live, isPlatformAdmin]);

  useEffect(() => {
    logRef.current?.scrollTo({ top: logRef.current.scrollHeight });
  }, [logs]);

  // 平台功能：非平台管理员直接访问也拒绝展示（导航已隐藏，双保险）
  if (!isPlatformAdmin) {
    return (
      <div className="page">
        <div className="panel">
          <h3>观测台</h3>
          <p className="muted">观测台为平台管理功能（§7.5），仅平台管理员可查看。</p>
        </div>
      </div>
    );
  }

  return (
    <div className="page">
      <div className="page-head">
        <h2>观测台</h2>
        <span className="crumb">OBSERVE</span>
        <div className="page-actions">
          <button className={`btn sm ${live ? "" : "ghost"}`} onClick={() => setLive((x) => !x)}>
            <i className={`run-dot${live ? "" : " off"}`} />
            {live ? "日志直播中" : "日志已暂停"}
          </button>
        </div>
      </div>
      <div className="page-sub">
        后台任务 · 系统日志 · LLM 调用审计 —— 后端在干什么，这里全看得见。
      </div>

      <div className="ob-grid">
        {/* 后台任务 */}
        <div className="ob-panel">
          <h3>后台任务</h3>
          <button className="btn ghost sm" onClick={() => taskLint().then(load)}>
            运行 Wiki Lint
          </button>
          {tasks.length === 0 && <div className="ob-empty">暂无任务</div>}
          {tasks.map((t) => (
            <div key={t.id} className="ob-task">
              <div className="ob-task-head">
                <span className={`ob-status ${t.status}`}>
                  {t.status === "running" ? (
                    <i className="run-dot" />
                  ) : t.status === "done" ? (
                    <Icon name="check" size={12} />
                  ) : (
                    <Icon name="cross" size={12} />
                  )}
                </span>
                <span className="ob-task-type">{t.type}</span>
                <span className="ob-task-detail">{t.detail}</span>
                <span className="ob-task-time">{new Date(t.created_at).toLocaleTimeString()}</span>
              </div>
              {t.result && <div className="ob-task-result">{t.result}</div>}
            </div>
          ))}
        </div>

        {/* LLM 审计 */}
        <div className="ob-panel">
          <h3>LLM 调用审计</h3>
          {audit.length === 0 && <div className="ob-empty">暂无调用</div>}
          {audit.slice(0, 15).map((a, i) => (
            <div key={i} className="ob-audit">
              <span className={`ob-audit-ok ${a.ok ? "ok" : "no"}`}>
                <Icon name={a.ok ? "check" : "cross"} size={12} />
              </span>
              <span className="ob-audit-model">{a.model}</span>
              {a.fallback && <span className="ob-audit-fb">回退</span>}
              <span className="ob-audit-task">{a.task}</span>
              <span className="ob-audit-ms">{a.duration_ms}ms</span>
              <span className="ob-audit-time">{new Date(a.at).toLocaleTimeString()}</span>
              {a.error && <div className="ob-audit-err">{a.error.slice(0, 100)}</div>}
            </div>
          ))}
        </div>

        {/* 系统健康（C2） */}
        {health && (
          <div className="ob-panel">
            <h3>系统健康</h3>
            <div className="cc-stats">
              <div className="cc-stat">
                <div className="cc-num">{health.tasks.length}</div>
                <div className="cc-label">后台任务（最近 5）</div>
              </div>
              <div className="cc-stat">
                <div className="cc-num">
                  {health.tasks.filter((t) => t.status === "running").length}
                </div>
                <div className="cc-label">运行中</div>
              </div>
              <div className="cc-stat">
                <div className="cc-num">
                  {health.status === "ok" ? <Icon name="check" size={13} /> : health.status}
                </div>
                <div className="cc-label">服务状态</div>
              </div>
            </div>
          </div>
        )}

        {/* 系统日志 */}
        <div className="ob-panel ob-logs">
          <h3>系统日志（tail 100 · 直播）</h3>
          <div className="ob-logview" ref={logRef}>
            {logs.length === 0 && <div className="ob-empty">等待日志…</div>}
            {logs.map((l, i) => (
              <div key={i} className="ob-logline">
                {l}
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
