import { useEffect, useState } from "react";
import type { ReactNode } from "react";
import {
  configGet,
  configPut,
  configPutBehavior,
  configTest,
  leadManagerPut,
  listModels,
  platformConsoleConfig,
} from "../api/client";
import { useAuth } from "../auth/AuthProvider";
import Icon from "../components/Icon";
import type { ConfigResponse, LeadManagerConfig, ModelConfig } from "../types";

type Cat = "models" | "routing" | "console" | "about";
const CATS: { id: Cat; label: string }[] = [
  { id: "models", label: "模型" },
  { id: "routing", label: "路由" },
  { id: "console", label: "配置中心" },
  { id: "about", label: "关于" },
];

const PROTOCOLS = [
  { id: "openai-chat", label: "OpenAI Chat" },
  { id: "openai-response", label: "OpenAI Response" },
  { id: "anthropic", label: "Anthropic" },
];

// 任务类型：路由表里每个任务可单独指定模型
const TASK_TYPES = [
  { id: "analysis", label: "主分析", desc: "需求理解 / 产品匹配 / 可行性判断（含工具调用）" },
  { id: "background", label: "后台任务", desc: "预留：摘要、压缩等独立任务" },
];

export default function Settings() {
  const { isPlatformAdmin } = useAuth();
  const [cfg, setCfg] = useState<ConfigResponse | null>(null);
  const [cat, setCat] = useState<Cat>("models");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [fetching, setFetching] = useState(false);
  const [error, setError] = useState("");
  const [msg, setMsg] = useState<ReactNode>("");
  // fetchModels 下拉弹窗：idx + 该模型返回的可选列表
  const [fetchPicker, setFetchPicker] = useState<{ idx: number; models: string[] } | null>(null);
  // 配置中心为平台管理功能（§7.5）：租户用户隐藏该 tab
  const cats = CATS.filter((c) => isPlatformAdmin || c.id !== "console");

  useEffect(() => {
    configGet()
      .then(setCfg)
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  }, []);

  async function save(next: ConfigResponse) {
    setSaving(true);
    setError("");
    setMsg("");
    try {
      const r = await configPut(next);
      setCfg(r);
      setMsg("已保存，回写 config.json");
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setSaving(false);
    }
  }

  function updateModel(i: number, patch: Partial<ModelConfig>) {
    if (!cfg) return;
    const models = [...cfg.models];
    models[i] = { ...models[i], ...patch };
    setCfg({ ...cfg, models });
  }

  // 测试单个模型（发 hi）
  async function testModel(m: ModelConfig) {
    setMsg("");
    setError("");
    setTesting(true);
    try {
      const r = await configTest({
        name: m.name,
        endpoint: m.endpoint,
        api_key: m.api_key && m.api_key !== "********" ? m.api_key : undefined,
        protocol: m.protocol,
        model: m.model,
      });
      if (r.ok)
        setMsg(
          <>
            <Icon name="check" size={12} /> 测试通过（{r.latency_ms}ms）：
            {(r.reply || "").slice(0, 80)}
          </>,
        );
      else setError(`测试失败（${r.latency_ms}ms）：${r.error}`);
    } catch (e) {
      setError("测试失败：" + (e instanceof Error ? e.message : String(e)));
    } finally {
      setTesting(false);
    }
  }

  // 自动获取模型列表——结果用 ConfirmDialog 模态下拉展示，避免原生 prompt 手抄全量列表。
  async function fetchModels(m: ModelConfig, idx: number) {
    setError("");
    setFetching(true);
    try {
      const r = await listModels({
        name: m.name,
        endpoint: m.endpoint,
        api_key: m.api_key && m.api_key !== "********" ? m.api_key : undefined,
        protocol: m.protocol,
      });
      if (r.models.length === 0) {
        setError("网关未返回模型列表");
        return;
      }
      setFetchPicker({ idx, models: r.models });
    } catch (e) {
      setError("获取模型失败：" + (e instanceof Error ? e.message : String(e)));
    } finally {
      setFetching(false);
    }
  }

  if (loading)
    return (
      <div className="page">
        <div className="loading">加载中…</div>
      </div>
    );
  if (!cfg)
    return (
      <div className="page">
        <div className="error">! {error || "无配置"}</div>
      </div>
    );

  return (
    <div className="settings">
      <h1>设置</h1>
      <div className="settings-cols">
        <aside className="settings-nav">
          {cats.map((c) => (
            <button
              key={c.id}
              className={cat === c.id ? "active" : ""}
              onClick={() => setCat(c.id)}
            >
              {c.label}
            </button>
          ))}
        </aside>

        <div className="settings-content">
          {msg && (
            <div
              className="panel s-msg"
              style={{
                borderColor: "var(--green)",
                color: "var(--green)",
                padding: "10px 14px",
                marginBottom: 14,
              }}
            >
              {msg}
            </div>
          )}
          {error && <div className="error">! {error}</div>}
          {!isPlatformAdmin && (
            <div className="panel muted" style={{ marginBottom: 14, fontSize: 12.5 }}>
              当前为只读视图：模型与路由为平台级配置，由平台管理员维护（§7.5）。
            </div>
          )}

          {/* 模型/路由编辑：平台管理员可改；租户用户 fieldset 整体禁用（只读） */}
          <fieldset className="s-edit" disabled={!isPlatformAdmin}>
            {cat === "models" && (
              <>
                <h2>模型配置</h2>
                <div className="s-group" style={{ marginTop: 12 }}>
                  {cfg.models.map((m, i) => (
                    <ModelCard
                      key={i}
                      m={m}
                      testing={testing}
                      fetching={fetching}
                      onPatch={(p) => updateModel(i, p)}
                      onRemove={() =>
                        setCfg({ ...cfg, models: cfg.models.filter((_, j) => j !== i) })
                      }
                      onTest={() => testModel(m)}
                      onFetch={() => fetchModels(m, i)}
                    />
                  ))}
                  <button
                    className="btn ghost sm"
                    onClick={() =>
                      setCfg({
                        ...cfg,
                        models: [
                          ...cfg.models,
                          {
                            name: `模型 ${cfg.models.length + 1}`,
                            endpoint: "",
                            api_key: "",
                            protocol: "openai-chat",
                            model: "",
                            temperature: 0.3,
                            max_tokens: 4096,
                          },
                        ],
                      })
                    }
                  >
                    + 添加模型
                  </button>
                </div>
                <div className="s-group">
                  <div className="s-gtitle">回退链（失败后依次尝试）</div>
                  {(cfg.router.fallback.chain ?? []).map((name, idx) => (
                    <div className="s-row" key={idx}>
                      <div className="s-name mono" style={{ fontSize: 12 }}>
                        {idx + 1}.
                      </div>
                      <div className="s-ctrl">
                        <select
                          value={name}
                          style={{ minWidth: 160 }}
                          onChange={(e) => {
                            const chain = [...(cfg.router.fallback.chain ?? [])];
                            chain[idx] = e.target.value;
                            setCfg({
                              ...cfg,
                              router: {
                                ...cfg.router,
                                fallback: { ...cfg.router.fallback, chain },
                              },
                            });
                          }}
                        >
                          {cfg.models.map((m) => (
                            <option key={m.name} value={m.name}>
                              {m.name}
                            </option>
                          ))}
                        </select>
                        <button
                          className="btn ghost sm"
                          title="移除该跳"
                          onClick={() => {
                            const chain = (cfg.router.fallback.chain ?? []).filter(
                              (_, j) => j !== idx,
                            );
                            setCfg({
                              ...cfg,
                              router: {
                                ...cfg.router,
                                fallback: { ...cfg.router.fallback, chain },
                              },
                            });
                          }}
                        >
                          <Icon name="cross" size={14} />
                        </button>
                      </div>
                    </div>
                  ))}
                  <div className="s-row">
                    <div className="s-name" />
                    <div className="s-ctrl">
                      <select
                        value=""
                        style={{ minWidth: 160 }}
                        onChange={(e) => {
                          if (!e.target.value) return;
                          const chain = [...(cfg.router.fallback.chain ?? []), e.target.value];
                          setCfg({
                            ...cfg,
                            router: { ...cfg.router, fallback: { ...cfg.router.fallback, chain } },
                          });
                        }}
                      >
                        <option value="">+ 添加回退模型…</option>
                        {cfg.models
                          .filter((m) => !(cfg.router.fallback.chain ?? []).includes(m.name))
                          .map((m) => (
                            <option key={m.name} value={m.name}>
                              {m.name}
                            </option>
                          ))}
                      </select>
                    </div>
                  </div>
                </div>
                <div style={{ marginTop: 18 }}>
                  <button className="btn primary" disabled={saving} onClick={() => save(cfg)}>
                    {saving ? "保存中…" : "保存"}
                  </button>
                </div>
              </>
            )}

            {cat === "routing" && (
              <>
                <h2>路由</h2>
                <p className="faint" style={{ fontSize: 13, marginBottom: 16 }}>
                  给不同任务类型指定模型。未指定的任务走默认模型。
                </p>
                <div className="s-group">
                  <div className="s-row">
                    <div>
                      <div className="s-name">默认模型</div>
                      <div className="s-desc">兜底：未单独指定的任务走这个</div>
                    </div>
                    <div className="s-ctrl">
                      <select
                        value={cfg.router.default}
                        onChange={(e) =>
                          setCfg({ ...cfg, router: { ...cfg.router, default: e.target.value } })
                        }
                      >
                        {cfg.models.map((m) => (
                          <option key={m.name} value={m.name}>
                            {m.name}
                          </option>
                        ))}
                      </select>
                    </div>
                  </div>

                  {TASK_TYPES.map((t) => (
                    <div className="s-row" key={t.id}>
                      <div>
                        <div className="s-name">{t.label}</div>
                        <div className="s-desc">{t.desc}</div>
                      </div>
                      <div className="s-ctrl">
                        <select
                          value={cfg.router.routes[t.id] || ""}
                          onChange={(e) =>
                            setCfg({
                              ...cfg,
                              router: {
                                ...cfg.router,
                                routes: { ...cfg.router.routes, [t.id]: e.target.value },
                              },
                            })
                          }
                        >
                          <option value="">跟随默认</option>
                          {cfg.models.map((m) => (
                            <option key={m.name} value={m.name}>
                              {m.name}
                            </option>
                          ))}
                        </select>
                      </div>
                    </div>
                  ))}
                </div>

                <div className="s-group">
                  <div className="s-gtitle">回退</div>
                  <div className="s-row">
                    <div className="s-name">最大重试次数</div>
                    <div className="s-ctrl">
                      <input
                        type="number"
                        style={{ minWidth: 100 }}
                        value={cfg.router.fallback.max_retries}
                        onChange={(e) =>
                          setCfg({
                            ...cfg,
                            router: {
                              ...cfg.router,
                              fallback: {
                                ...cfg.router.fallback,
                                max_retries: parseInt(e.target.value),
                              },
                            },
                          })
                        }
                      />
                    </div>
                  </div>
                  <div className="s-row">
                    <div className="s-name">退避基数（毫秒）</div>
                    <div className="s-ctrl">
                      <input
                        type="number"
                        style={{ minWidth: 100 }}
                        value={cfg.router.fallback.backoff_base_ms}
                        onChange={(e) =>
                          setCfg({
                            ...cfg,
                            router: {
                              ...cfg.router,
                              fallback: {
                                ...cfg.router.fallback,
                                backoff_base_ms: parseInt(e.target.value),
                              },
                            },
                          })
                        }
                      />
                    </div>
                  </div>
                </div>
                <div className="s-group">
                  <div className="s-gtitle">回退链（失败后依次尝试）</div>
                  {(cfg.router.fallback.chain ?? []).map((name, idx) => (
                    <div className="s-row" key={idx}>
                      <div className="s-name mono" style={{ fontSize: 12 }}>
                        {idx + 1}.
                      </div>
                      <div className="s-ctrl">
                        <select
                          value={name}
                          style={{ minWidth: 160 }}
                          onChange={(e) => {
                            const chain = [...(cfg.router.fallback.chain ?? [])];
                            chain[idx] = e.target.value;
                            setCfg({
                              ...cfg,
                              router: {
                                ...cfg.router,
                                fallback: { ...cfg.router.fallback, chain },
                              },
                            });
                          }}
                        >
                          {cfg.models.map((m) => (
                            <option key={m.name} value={m.name}>
                              {m.name}
                            </option>
                          ))}
                        </select>
                        <button
                          className="btn ghost sm"
                          title="移除该跳"
                          onClick={() => {
                            const chain = (cfg.router.fallback.chain ?? []).filter(
                              (_, j) => j !== idx,
                            );
                            setCfg({
                              ...cfg,
                              router: {
                                ...cfg.router,
                                fallback: { ...cfg.router.fallback, chain },
                              },
                            });
                          }}
                        >
                          <Icon name="cross" size={14} />
                        </button>
                      </div>
                    </div>
                  ))}
                  <div className="s-row">
                    <div className="s-name" />
                    <div className="s-ctrl">
                      <select
                        value=""
                        style={{ minWidth: 160 }}
                        onChange={(e) => {
                          if (!e.target.value) return;
                          const chain = [...(cfg.router.fallback.chain ?? []), e.target.value];
                          setCfg({
                            ...cfg,
                            router: { ...cfg.router, fallback: { ...cfg.router.fallback, chain } },
                          });
                        }}
                      >
                        <option value="">+ 添加回退模型…</option>
                        {cfg.models
                          .filter((m) => !(cfg.router.fallback.chain ?? []).includes(m.name))
                          .map((m) => (
                            <option key={m.name} value={m.name}>
                              {m.name}
                            </option>
                          ))}
                      </select>
                    </div>
                  </div>
                </div>
                <div style={{ marginTop: 18 }}>
                  <button className="btn primary" disabled={saving} onClick={() => save(cfg)}>
                    {saving ? "保存中…" : "保存"}
                  </button>
                </div>
              </>
            )}
          </fieldset>

          {/* 获取模型下拉（fetchPicker 状态下在顶部绘制：复用 ConfirmDialog 布局但装载 select）*/}
          {fetchPicker && (
            <div className="cd-mask" onClick={() => setFetchPicker(null)} role="presentation">
              <div className="cd-box" onClick={(e) => e.stopPropagation()} role="alertdialog">
                <div className="cd-title">选择模型 ID</div>
                <div className="cd-body">
                  <select
                    autoFocus
                    size={Math.min(fetchPicker.models.length, 10)}
                    defaultValue={cfg?.models[fetchPicker.idx]?.model || ""}
                    onChange={(e) => {
                      updateModel(fetchPicker.idx, { model: e.target.value });
                      setFetchPicker(null);
                    }}
                    style={{ width: "100%", fontFamily: "var(--mono)" }}
                  >
                    {fetchPicker.models.map((m) => (
                      <option key={m} value={m}>
                        {m}
                      </option>
                    ))}
                  </select>
                  <div style={{ marginTop: 8, fontSize: 12, color: "var(--text-faint)" }}>
                    从网关返回 {fetchPicker.models.length} 个模型。点遮罩或选 Esc 取消。
                  </div>
                </div>
                <div className="cd-btns">
                  <button className="btn ghost sm" onClick={() => setFetchPicker(null)}>
                    取消
                  </button>
                </div>
              </div>
            </div>
          )}

          {cat === "console" && <ConsoleCenter />}

          {cat === "about" && (
            <>
              <h2>关于</h2>
              <div className="s-group">
                <div className="s-row">
                  <div className="s-name">应用</div>
                  <div className="s-ctrl muted">客户需求分析智能体 · 长亭</div>
                </div>
                <div className="s-row">
                  <div className="s-name">版本</div>
                  <div className="s-ctrl mono">v1.0.0</div>
                </div>
                <div className="s-row">
                  <div className="s-name">技术栈</div>
                  <div className="s-ctrl muted">Go 后端 · React 前端</div>
                </div>
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  );
}

// 单个模型的紧凑卡片（label 在上 input 在下的网格）
function ModelCard({
  m,
  onPatch,
  onRemove,
  onTest,
  onFetch,
  testing,
  fetching,
}: {
  m: ModelConfig;
  onPatch: (p: Partial<ModelConfig>) => void;
  onRemove: () => void;
  onTest: () => void;
  onFetch: () => void;
  testing: boolean;
  fetching: boolean;
}) {
  return (
    <div className="model-card">
      <div className="mc-head">
        <input
          className="mc-name"
          value={m.name}
          placeholder="模型名称"
          onChange={(e) => onPatch({ name: e.target.value })}
        />
        <div className="row" style={{ gap: 6 }}>
          <button className="btn ghost sm" onClick={onTest} disabled={testing || fetching}>
            {testing ? "测试中…" : "测试"}
          </button>
          <button className="btn ghost sm" onClick={onRemove} disabled={testing || fetching}>
            移除
          </button>
        </div>
      </div>
      <div className="model-form">
        <div>
          <label>协议</label>
          <select value={m.protocol} onChange={(e) => onPatch({ protocol: e.target.value })}>
            {PROTOCOLS.map((p) => (
              <option key={p.id} value={p.id}>
                {p.label}
              </option>
            ))}
          </select>
        </div>
        <div>
          <label>模型 ID</label>
          <div className="mfield-row">
            <input
              value={m.model}
              placeholder="模型名"
              onChange={(e) => onPatch({ model: e.target.value })}
            />
            <button
              className="btn ghost sm"
              onClick={onFetch}
              disabled={testing || fetching}
              title="从网关自动获取模型列表"
            >
              {fetching ? "获取中…" : "获取"}
            </button>
          </div>
        </div>
        <div className="full">
          <label>网关地址</label>
          <input
            value={m.endpoint}
            placeholder="https://…"
            onChange={(e) => onPatch({ endpoint: e.target.value })}
          />
        </div>
        <div className="full">
          <label>API Key</label>
          <input
            type="password"
            value={m.api_key || ""}
            placeholder="留空 = 不改（已设置显示 ********，点进去可改）"
            onFocus={(e) => e.target.select()}
            onChange={(e) => onPatch({ api_key: e.target.value })}
          />
        </div>
        <div>
          <label>温度</label>
          <input
            type="number"
            step="0.1"
            value={m.temperature}
            onChange={(e) => onPatch({ temperature: parseFloat(e.target.value) })}
          />
        </div>
        <div>
          <label>Max Tokens</label>
          <input
            type="number"
            value={m.max_tokens}
            onChange={(e) => onPatch({ max_tokens: parseInt(e.target.value) })}
          />
        </div>
        <details className="mc-advanced">
          <summary>高级设置（可选）</summary>
          <div className="mgrid">
            <div>
              <label>最大上下文（token）</label>
              <input
                type="number"
                value={m.context_window || ""}
                placeholder="未配置"
                onChange={(e) => onPatch({ context_window: parseInt(e.target.value) || 0 })}
              />
            </div>
            <div>
              <label>超时（秒，0=全局）</label>
              <input
                type="number"
                value={m.timeout_sec || ""}
                placeholder="走全局"
                onChange={(e) => onPatch({ timeout_sec: parseInt(e.target.value) || 0 })}
              />
            </div>
            <div>
              <label>重试次数（0=全局）</label>
              <input
                type="number"
                value={m.max_retries || ""}
                placeholder="走全局"
                onChange={(e) => onPatch({ max_retries: parseInt(e.target.value) || 0 })}
              />
            </div>
            <div>
              <label>备注</label>
              <input
                value={m.remark || ""}
                placeholder="如：DeepSeek 官方-便宜"
                onChange={(e) => onPatch({ remark: e.target.value })}
              />
            </div>
            <div>
              <label>启用</label>
              <input
                type="checkbox"
                className="checkbox"
                checked={m.enabled !== false}
                onChange={(e) => onPatch({ enabled: e.target.checked })}
              />
            </div>
          </div>
        </details>
      </div>
    </div>
  );
}

// ── 配置中心（C1，只读）──────────────────────────────────────

interface ConsoleData {
  models: {
    name: string;
    endpoint: string;
    model: string;
    temperature: number;
    max_tokens: number;
    protocol: string;
    has_key: boolean;
    context_window?: number;
    timeout_sec?: number;
    max_retries?: number;
    enabled?: boolean;
    remark?: string;
  }[];
  router: {
    default: string;
    routes: Record<string, string>;
    fallback: { max_retries: number; backoff_base_ms: number; chain: string[] };
  };
  data: { data_dir: string; wiki_dir: string; history_db: string };
  behavior: { agent_max_iterations: number; llm_timeout_sec: number };
  lead_manager?: { enabled: boolean; base_url: string; has_key: boolean; active: boolean };
}

function ConsoleCenter() {
  const [cfg, setCfg] = useState<ConsoleData | null>(null);
  const [behavior, setBehavior] = useState<ConsoleData["behavior"] | null>(null);
  const [savingBehavior, setSavingBehavior] = useState(false);
  const [saveErr, setSaveErr] = useState("");
  useEffect(() => {
    platformConsoleConfig<ConsoleData>()
      .then((d) => {
        setCfg(d);
        setBehavior(d.behavior);
      })
      .catch((e) => setSaveErr(e instanceof Error ? e.message : String(e)));
  }, []);
  async function saveBehavior() {
    if (!behavior) return;
    setSavingBehavior(true);
    setSaveErr("");
    try {
      await configPutBehavior({
        llm_timeout_sec: behavior.llm_timeout_sec,
        agent_max_iterations: behavior.agent_max_iterations,
      });
    } catch (e) {
      setSaveErr(e instanceof Error ? e.message : String(e));
    } finally {
      setSavingBehavior(false);
    }
  }

  if (!cfg || !behavior)
    return <div className={saveErr ? "error-box" : "loading"}>{saveErr || "加载中…"}</div>;
  return (
    <>
      <h2>配置中心（只读）</h2>

      <h3 className="cc-h">回退链</h3>
      <div className="cc-group">
        <div>
          默认模型 <code>{cfg.router.default}</code>；重试 {cfg.router.fallback.max_retries} 次 /
          退避 {cfg.router.fallback.backoff_base_ms}ms
        </div>
        {cfg.router.fallback.chain?.length > 0 ? (
          <div className="cc-chain">
            {cfg.router.fallback.chain.map((m, i) => (
              <span key={i}>
                {i > 0 && <em>→ 失败 →</em>} {m}
              </span>
            ))}
          </div>
        ) : (
          <div className="muted">无回退链</div>
        )}
      </div>

      <h3 className="cc-h">行为参数（可编辑）</h3>
      <div className="cc-group">
        <label>
          Agent 最大迭代
          <input
            type="number"
            value={behavior.agent_max_iterations}
            onChange={(e) =>
              setBehavior({ ...behavior, agent_max_iterations: parseInt(e.target.value) || 15 })
            }
          />
        </label>
        <div className="muted" style={{ fontSize: 13 }}>
          输出风格会按当前登录用户自动加载本租户使用者画像，无需单独指定“当前销售”。
        </div>
        <label>
          LLM 全局超时（秒）
          <input
            type="number"
            value={behavior.llm_timeout_sec}
            onChange={(e) =>
              setBehavior({ ...behavior, llm_timeout_sec: parseInt(e.target.value) || 120 })
            }
          />
        </label>
      </div>
      <div className="row" style={{ gap: 10 }}>
        <button className="btn sm" disabled={savingBehavior} onClick={saveBehavior}>
          {savingBehavior ? "保存中…" : "保存（立即生效，无需重启）"}
        </button>
        {saveErr && <span style={{ color: "var(--red)", fontSize: 12 }}>{saveErr}</span>}
      </div>

      <h3 className="cc-h">数据位置（只读——迁移需停服）</h3>
      <div className="cc-group mono">
        <div>数据根：{cfg.data.data_dir}</div>
        <div>Wiki：{cfg.data.wiki_dir}</div>
        <div>历史库：{cfg.data.history_db}</div>
      </div>

      <LeadManagerGroup />
    </>
  );
}

// ── 商机平台（Lead Manager）分组（lead-manager 接入 M2）────
// 系统级 Token：管理员配一次、全员共用；保存后重启生效（同 jiying 语义）。
function LeadManagerGroup() {
  const [lm, setLm] = useState<LeadManagerConfig | null>(null);
  const [saving, setSaving] = useState(false);
  const [msg, setMsg] = useState("");
  const [err, setErr] = useState("");
  useEffect(() => {
    configGet()
      .then((c) => {
        if (c.lead_manager) setLm(c.lead_manager);
      })
      .catch((e) => setErr(e instanceof Error ? e.message : String(e)));
  }, []);
  async function save() {
    if (!lm) return;
    setSaving(true);
    setMsg("");
    setErr("");
    try {
      const r = await leadManagerPut(lm);
      if (r.lead_manager) setLm(r.lead_manager);
      setMsg("已保存并回写 config.json，重启后生效");
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setSaving(false);
    }
  }
  if (!lm) return err ? <div className="error-box">{err}</div> : null;
  return (
    <>
      <h3 className="cc-h">商机平台（Lead Manager）</h3>
      <div className="cc-group">
        <label>
          启用
          <input
            type="checkbox"
            className="checkbox"
            checked={lm.enabled}
            onChange={(e) => setLm({ ...lm, enabled: e.target.checked })}
          />
        </label>
        <label>
          平台地址
          <input
            value={lm.base_url || ""}
            placeholder="http://api.in.chaitin.net/mql"
            onChange={(e) => setLm({ ...lm, base_url: e.target.value })}
          />
        </label>
        <label>
          API Token（系统级）
          <input
            type="password"
            value={lm.api_key || ""}
            placeholder="lm_pat_…（留空 = 不改；已设置显示 ********）"
            onFocus={(e) => e.target.select()}
            onChange={(e) => setLm({ ...lm, api_key: e.target.value })}
          />
        </label>
        <details style={{ marginTop: 8 }}>
          <summary style={{ fontSize: 12, color: "var(--text-faint)", cursor: "pointer" }}>
            高级（限流/超时，一般用默认）
          </summary>
          <div className="cc-group" style={{ marginTop: 8 }}>
            <label>
              超时（秒）
              <input
                type="number"
                value={lm.timeout_sec ?? 10}
                onChange={(e) => setLm({ ...lm, timeout_sec: parseInt(e.target.value) || 10 })}
              />
            </label>
            <label>
              限流（次/分钟）
              <input
                type="number"
                value={lm.rate_per_min ?? 60}
                onChange={(e) => setLm({ ...lm, rate_per_min: parseInt(e.target.value) || 60 })}
              />
            </label>
            <label>
              突发容量
              <input
                type="number"
                value={lm.burst ?? 3}
                onChange={(e) => setLm({ ...lm, burst: parseInt(e.target.value) || 3 })}
              />
            </label>
          </div>
        </details>
        <div className="muted" style={{ fontSize: 12, lineHeight: 1.7 }}>
          系统级 Token：管理员配一次、全员共用（统计权限随 Token 创建人角色）。 需公司内网或
          VPN；仅只读查询（列表/详情/统计）。遇 HTTP 401 说明 Token 失效，
          请在商机平台重建后更新此处；怀疑泄露时在平台侧吊销。
        </div>
      </div>
      <div className="row" style={{ gap: 10, alignItems: "center" }}>
        <button className="btn sm" disabled={saving} onClick={save}>
          {saving ? "保存中…" : "保存（重启后生效）"}
        </button>
        {msg && <span style={{ color: "var(--green)", fontSize: 12 }}>{msg}</span>}
        {err && <span style={{ color: "var(--red)", fontSize: 12 }}>{err}</span>}
      </div>
    </>
  );
}
