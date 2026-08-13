import { useEffect, useState } from "react";
import { configGet, configPut } from "../api/client";
import type { ConfigResponse, ModelConfig } from "../types";

type Cat = "models" | "routing" | "about";
const CATS: { id: Cat; label: string; icon: string }[] = [
  { id: "models", label: "模型", icon: "M12 2a10 10 0 1 0 0 20 10 10 0 0 0 0-20zM2 12h20M12 2a15 15 0 0 1 0 20M12 2a15 15 0 0 0 0 20" },
  { id: "routing", label: "路由", icon: "M3 6h18M3 12h18M3 18h18M7 6v0M7 12v0M7 18v0" },
  { id: "about", label: "关于", icon: "M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" },
];

export default function Settings() {
  const [cfg, setCfg] = useState<ConfigResponse | null>(null);
  const [cat, setCat] = useState<Cat>("models");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [msg, setMsg] = useState("");

  useEffect(() => { configGet().then(setCfg).catch((e) => setError(e.message)).finally(() => setLoading(false)); }, []);

  async function save(next: ConfigResponse) {
    setSaving(true); setError(""); setMsg("");
    try { const r = await configPut(next); setCfg(r); setMsg("已保存，回写 config.json"); }
    catch (e: any) { setError(e.message); }
    finally { setSaving(false); }
  }

  function updateModel(i: number, patch: Partial<ModelConfig>) {
    if (!cfg) return;
    const models = [...cfg.models]; models[i] = { ...models[i], ...patch };
    setCfg({ ...cfg, models });
  }

  if (loading) return <div className="page"><div className="loading">加载中…</div></div>;
  if (!cfg) return <div className="page"><div className="error">! {error || "无配置"}</div></div>;

  return (
    <div className="settings">
      <h1>设置</h1>
      <div className="settings-cols">
        <aside className="settings-nav">
          {CATS.map((c) => (
            <button key={c.id} className={cat === c.id ? "active" : ""} onClick={() => setCat(c.id)}>
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round"><path d={c.icon} /></svg>
              {c.label}
            </button>
          ))}
        </aside>

        <div className="settings-content">
          {msg && <div className="panel" style={{ borderColor: "var(--green)", color: "var(--green)", padding: "10px 14px", marginBottom: 14 }}>{msg}</div>}
          {error && <div className="error">! {error}</div>}

          {cat === "models" && (
            <>
              <h2>模型配置</h2>
              <div className="s-group">
                <div className="s-gtitle">已注册模型（OpenAI 兼容网关，默认支持 function call）</div>
                {cfg.models.map((m, i) => (
                  <div className="model-card" key={i}>
                    <div className="mc-head">
                      <input className="mc-title" style={{ width: "auto", fontWeight: 600 }} value={m.name} onChange={(e) => updateModel(i, { name: e.target.value })} />
                      <button className="btn ghost sm" onClick={() => setCfg({ ...cfg, models: cfg.models.filter((_, j) => j !== i) })}>移除</button>
                    </div>
                    <div className="s-row"><div><div className="s-name">模型 ID</div><div className="s-desc">网关实际模型名</div></div>
                      <div className="s-ctrl"><input value={m.model} onChange={(e) => updateModel(i, { model: e.target.value })} /></div></div>
                    <div className="s-row"><div><div className="s-name">API 网关地址</div><div className="s-desc">OpenAI 兼容 endpoint</div></div>
                      <div className="s-ctrl"><input value={m.endpoint} onChange={(e) => updateModel(i, { endpoint: e.target.value })} /></div></div>
                    <div className="s-row"><div><div className="s-name">API Key</div><div className="s-desc">{m.api_key ? "已配置，留空不改" : "未配置"}</div></div>
                      <div className="s-ctrl"><input type="password" value={m.api_key || ""} placeholder="留空 = 不修改" onChange={(e) => updateModel(i, { api_key: e.target.value })} /></div></div>
                    <div className="s-row"><div className="s-name">Temperature</div>
                      <div className="s-ctrl"><input type="number" step="0.1" style={{ minWidth: 100 }} value={m.temperature} onChange={(e) => updateModel(i, { temperature: parseFloat(e.target.value) })} /></div></div>
                    <div className="s-row"><div className="s-name">Max Tokens / 超时(秒)</div>
                      <div className="s-ctrl row"><input type="number" style={{ minWidth: 90 }} value={m.max_tokens} onChange={(e) => updateModel(i, { max_tokens: parseInt(e.target.value) })} /><input type="number" style={{ minWidth: 90 }} value={m.timeout_sec} onChange={(e) => updateModel(i, { timeout_sec: parseInt(e.target.value) })} /></div></div>
                  </div>
                ))}
                <button className="btn ghost sm" onClick={() => { const name = prompt("新模型名称："); if (name) setCfg({ ...cfg, models: [...cfg.models, { name, endpoint: "", api_key: "", model: "", temperature: 0.3, max_tokens: 4096, timeout_sec: 60 }] }); }}>+ 添加模型</button>
              </div>
              <div style={{ marginTop: 18 }}><button className="btn primary" disabled={saving} onClick={() => save(cfg)}>{saving ? "保存中…" : "保存"}</button></div>
            </>
          )}

          {cat === "routing" && (
            <>
              <h2>路由与回退</h2>
              <div className="s-group">
                <div className="s-gtitle">默认模型</div>
                <div className="s-row"><div><div className="s-name">所有任务默认走</div><div className="s-desc">未单独配置的任务类型自动回退到此模型</div></div>
                  <div className="s-ctrl"><select value={cfg.router.default} onChange={(e) => setCfg({ ...cfg, router: { ...cfg.router, default: e.target.value } })}>
                    {cfg.models.map((m) => <option key={m.name} value={m.name}>{m.name}</option>)}
                  </select></div></div>
                <div className="s-row"><div><div className="s-name">最大重试次数</div><div className="s-desc">单模型失败后的重试（指数退避）</div></div>
                  <div className="s-ctrl"><input type="number" style={{ minWidth: 100 }} value={cfg.router.fallback.max_retries} onChange={(e) => setCfg({ ...cfg, router: { ...cfg.router, fallback: { ...cfg.router.fallback, max_retries: parseInt(e.target.value) } } })} /></div></div>
                <div className="s-row"><div><div className="s-name">退避基数（毫秒）</div><div className="s-desc">指数退避起始间隔</div></div>
                  <div className="s-ctrl"><input type="number" style={{ minWidth: 100 }} value={cfg.router.fallback.backoff_base_ms} onChange={(e) => setCfg({ ...cfg, router: { ...cfg.router, fallback: { ...cfg.router.fallback, backoff_base_ms: parseInt(e.target.value) } } })} /></div></div>
              </div>
              <div style={{ marginTop: 18 }}><button className="btn primary" disabled={saving} onClick={() => save(cfg)}>{saving ? "保存中…" : "保存"}</button></div>
            </>
          )}

          {cat === "about" && (
            <>
              <h2>关于</h2>
              <div className="s-group">
                <div className="s-row"><div className="s-name">应用</div><div className="s-ctrl muted">客户需求分析智能体 · 长亭</div></div>
                <div className="s-row"><div className="s-name">版本</div><div className="s-ctrl mono">v1.0.0</div></div>
                <div className="s-row"><div className="s-name">技术栈</div><div className="s-ctrl muted">Go 后端 · React 前端</div></div>
                <div className="s-row"><div className="s-name">架构</div><div className="s-ctrl muted">自主 Agent · Wiki 记忆 · SSE 流式</div></div>
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
