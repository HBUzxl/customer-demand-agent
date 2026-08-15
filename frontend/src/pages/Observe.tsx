import { useEffect, useRef, useState } from "react";

/**
 * 观测台（C2）：后台任务 + 日志 tail（SSE）+ LLM 调用审计 + 系统健康。
 * 「后端在干什么」的统一视图。
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
}

export default function Observe() {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [audit, setAudit] = useState<Audit[]>([]);
  const [health, setHealth] = useState<Health | null>(null);
  const [logs, setLogs] = useState<string[]>([]);
  const [live, setLive] = useState(true);
  const logRef = useRef<HTMLDivElement>(null);

  async function load() {
    try {
      const [t, a, h] = await Promise.all([
        fetch("/api/tasks").then((r) => r.json()),
        fetch("/api/console/llm-audit").then((r) => r.json()),
        fetch("/api/console/health").then((r) => r.json()),
      ]);
      setTasks(t.items || []);
      setAudit(a.items || []);
      setHealth(h);
    } catch {
      /* 后端不可达 */
    }
  }

  useEffect(() => {
    load();
    const t = setInterval(load, 5000);
    return () => clearInterval(t);
  }, []);

  useEffect(() => {
    if (!live) return;
    const ac = new AbortController();
    (async () => {
      const res = await fetch("/api/console/logs", { signal: ac.signal });
      const reader = res.body!.getReader();
      const decoder = new TextDecoder();
      let buf = "";
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        buf += decoder.decode(value, { stream: true });
        let idx: number;
        while ((idx = buf.indexOf("\n\n")) >= 0) {
          const raw = buf.slice(0, idx);
          buf = buf.slice(idx + 2);
          if (raw.startsWith("data:")) {
            setLogs((ls) => [...ls.slice(-200), raw.slice(5).trim()]);
          }
        }
      }
    })().catch(() => {});
    return () => ac.abort();
  }, [live]);

  useEffect(() => {
    logRef.current?.scrollTo({ top: logRef.current.scrollHeight });
  }, [logs]);

  return (
    <div className="page">
      <div className="page-head">
        <h2>观测台</h2>
        <span className="crumb">OBSERVE</span>
        <div className="page-actions">
          <button className={`btn sm ${live ? "" : "ghost"}`} onClick={() => setLive((x) => !x)}>
            {live ? "◉ 日志直播中" : "○ 日志已暂停"}
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
          <button
            className="btn ghost sm"
            onClick={() => fetch("/api/tasks/lint", { method: "POST" }).then(load)}
          >
            运行 Wiki Lint
          </button>
          {tasks.length === 0 && <div className="ob-empty">暂无任务</div>}
          {tasks.map((t) => (
            <div key={t.id} className="ob-task">
              <div className="ob-task-head">
                <span className={`ob-status ${t.status}`}>
                  {t.status === "running" ? "◉" : t.status === "done" ? "✓" : "✗"}
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
              <span className={`ob-audit-ok ${a.ok ? "ok" : "no"}`}>{a.ok ? "✓" : "✗"}</span>
              <span className="ob-audit-model">{a.model}</span>
              {a.fallback && <span className="ob-audit-fb">回退</span>}
              <span className="ob-audit-task">{a.task}</span>
              <span className="ob-audit-ms">{a.duration_ms}ms</span>
              <span className="ob-audit-time">{new Date(a.at).toLocaleTimeString()}</span>
              {a.error && <div className="ob-audit-err">{a.error.slice(0, 100)}</div>}
            </div>
          ))}
        </div>

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
