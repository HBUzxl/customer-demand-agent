// 商机面板（Dashboard）：「看数据的面板」与「干活的 AI」闭环的数据侧。
// 面板负责发现高价值线索（stage=mql 口径，与 leads_search 一致）；
// 每条线索旁的「AI 分析」把线索上下文经 sessionStorage 交接给对话页
// （/analyze 自动发送），由 Agent 一条龙分析（leads_get → memory_search → 建议）。
import { useCallback, useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { leadsDashboard, leadsStats } from "../api/client";
import { useAuth } from "../auth/AuthProvider";
import type { LeadsDashboard, LeadsStats } from "../api/client";

const PAGE_SIZE = 20;

// ── 线索字段容错提取（平台响应字段名未定死，按候选名匹配）────

const firstStr = (o: Record<string, unknown>, keys: string[]): string => {
  for (const k of keys) {
    const v = o[k];
    if (typeof v === "string" && v.trim()) return v.trim();
    if (typeof v === "number" && Number.isFinite(v)) return String(v);
  }
  return "";
};

const firstArr = (o: Record<string, unknown>, keys: string[]): string[] => {
  for (const k of keys) {
    const v = o[k];
    if (Array.isArray(v)) {
      return v
        .map((x) =>
          typeof x === "string" || typeof x === "number"
            ? String(x)
            : x && typeof x === "object"
              ? firstStr(x as Record<string, unknown>, ["name", "title", "label"])
              : "",
        )
        .filter(Boolean);
    }
    if (typeof v === "string" && v.trim()) return [v.trim()];
  }
  return [];
};

const ID_KEYS = ["id", "lead_id", "leadId", "uuid"];
// 字段候选含商机平台（Lead Manager）实测字段名（2026-08 联调：
// company_name / assigned_to_name / contact_name / product_names / notes 等）。
const TITLE_KEYS = [
  "company_name",
  "title",
  "name",
  "customer",
  "customer_name",
  "company",
  "lead_name",
  "客户",
  "客户名称",
  "公司名称",
];
const STAGE_KEYS = ["stage", "stage_name", "stageName", "阶段"];
const STATUS_KEYS = ["status", "state", "状态"];
const OWNER_KEYS = [
  "assigned_to_name",
  "owner_name",
  "owner",
  "assignee",
  "ml_creator_name",
  "creator",
  "sales",
  "负责人",
];
const CONTACT_KEYS = ["contact_name", "contact", "联系人"];
const PHONE_KEYS = ["contact_phone", "phone", "mobile", "电话"];
const TIME_KEYS = ["created_at", "createdAt", "create_time", "created"];
const FOLLOWUP_KEYS = ["last_followup_at", "last_followup", "最近跟进"];
const SOURCE_KEYS = ["source", "external_source", "来源"];
const INDUSTRY_KEYS = ["industry", "行业"];
const AMOUNT_KEYS = ["estimated_amount", "budget", "amount", "预算", "预估金额"];
const CURRENCY_KEYS = ["currency", "币种"];
const TAG_KEYS = ["tags", "labels", "标签"];
const PROD_KEYS = ["product_names", "products", "productNames", "interest_products", "产品"];
const DESC_KEYS = ["notes", "description", "remark", "summary", "备注", "描述", "需求"];

function fmtTime(v: string): string {
  if (!v) return "";
  const n = Number(v);
  const d = Number.isFinite(n) && v.trim() !== "" ? new Date(n > 1e12 ? n : n * 1000) : new Date(v);
  return Number.isNaN(d.getTime()) ? v : d.toLocaleString("zh-CN", { hour12: false });
}

// 一键 AI 分析：组装带线索上下文的交接 prompt → 跳转对话页自动发送。
// customer 同时作为会话客户身份（身份行注入 + session_bind_customer 闭环）。
function analyzeWithAI(
  item: Record<string, unknown>,
  navigate: (to: string) => void,
  handoffKey: string,
) {
  const id = firstStr(item, ID_KEYS);
  const title = firstStr(item, TITLE_KEYS) || (id ? `线索 ${id}` : "未命名线索");
  const stage = firstStr(item, STAGE_KEYS);
  const status = firstStr(item, STATUS_KEYS);
  const owner = firstStr(item, OWNER_KEYS);
  const contact = firstStr(item, CONTACT_KEYS);
  const phone = firstStr(item, PHONE_KEYS);
  const time = firstStr(item, TIME_KEYS);
  const followup = firstStr(item, FOLLOWUP_KEYS);
  const source = firstStr(item, SOURCE_KEYS);
  const industry = firstStr(item, INDUSTRY_KEYS);
  const amount = firstStr(item, AMOUNT_KEYS);
  const currency = firstStr(item, CURRENCY_KEYS) || "CNY";
  const prods = firstArr(item, PROD_KEYS);
  const tags = firstArr(item, TAG_KEYS);
  const desc = firstStr(item, DESC_KEYS);
  const lines = [
    `- 客户：${title}`,
    id && `- 线索 ID：${id}`,
    stage && `- 阶段：${stage}${status ? `（状态 ${status}）` : ""}`,
    owner && `- 负责人：${owner}`,
    contact && `- 联系人：${contact}${phone ? `（${phone}）` : ""}`,
    source && `- 来源：${source}`,
    industry && `- 行业：${industry}`,
    amount && `- 预算/预估金额：${amount} ${currency}`,
    time && `- 创建时间：${time}`,
    followup && `- 最近跟进：${followup}`,
    prods.length > 0 && `- 涉及产品：${prods.join("、")}`,
    tags.length > 0 && `- 标签：${tags.join("、")}`,
    desc && `- 备注：${desc}`,
  ]
    .filter(Boolean)
    .join("\n");
  const ref = id ? `线索 ${id}` : "该线索";
  const text =
    `【商机面板 · 一键 AI 分析】\n` +
    `请对高价值线索「${title}」做一条龙跟进分析：先用 leads_get 读取${ref}的详情与活动记录` +
    `（with_activities=true），再结合产品知识库（memory_search）匹配适合的长亭产品，` +
    `输出：需求理解、推荐产品与置信度、可行性判断、跟进话术建议、待向客户确认的信息。\n\n` +
    `面板线索摘要：\n${lines}`;
  sessionStorage.setItem(handoffKey, JSON.stringify({ text, customer: title }));
  navigate("/analyze");
}

// 统计数据容错提取：一层对象（含一层嵌套）里的数值字段 → KPI 卡（至多 4 个）。
// 已知漏斗键（ml/mql/msb/msp，实测字段）给友好标签，未知键原样展示。
const STAT_LABELS: Record<string, string> = {
  ml: "ML 线索",
  mql: "MQL 高价值",
  msb: "MSB",
  msp: "MSP",
};
function statNumbers(data: unknown): { label: string; value: string }[] {
  if (!data || typeof data !== "object" || Array.isArray(data)) return [];
  const out: { label: string; value: string }[] = [];
  for (const [k, v] of Object.entries(data as Record<string, unknown>)) {
    if (typeof v === "number") out.push({ label: STAT_LABELS[k] || k, value: String(v) });
    else if (v && typeof v === "object" && !Array.isArray(v))
      for (const [k2, v2] of Object.entries(v as Record<string, unknown>))
        if (typeof v2 === "number") out.push({ label: k2, value: String(v2) });
    if (out.length >= 4) break;
  }
  return out;
}

export default function Dashboard() {
  const navigate = useNavigate();
  const { lsKey } = useAuth();
  const [page, setPage] = useState(1);
  const [data, setData] = useState<LeadsDashboard | null>(null);
  const [stats, setStats] = useState<LeadsStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState("");
  const [loadedAt, setLoadedAt] = useState<string>("");

  const load = useCallback(async (p: number) => {
    setLoading(true);
    setErr("");
    try {
      const d = await leadsDashboard(p, PAGE_SIZE);
      setData(d);
      setLoadedAt(new Date().toLocaleTimeString("zh-CN", { hour12: false }));
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
    // 统计 best-effort：失败不阻塞面板（列表照常渲染）
    leadsStats("summary")
      .then(setStats)
      .catch(() => setStats(null));
  }, []);

  useEffect(() => {
    load(page);
  }, [page, load]);

  const items = data?.items || [];
  const total = data?.total ?? null;
  const kpis = statNumbers(stats?.data);

  return (
    <div className="page">
      <div className="page-head">
        <h2>
          商机面板 <span className="faint">· 高价值线索（MQL）</span>
        </h2>
        <div className="page-actions">
          {loadedAt && (
            <span className="faint" style={{ fontSize: 12 }}>
              更新于 {loadedAt}
            </span>
          )}
          <button className="btn sm" onClick={() => load(page)} disabled={loading}>
            刷新
          </button>
        </div>
      </div>
      <p className="page-sub">
        面板发现高价值线索，AI 负责一条龙分析——点击线索卡上的「AI
        分析」，自动带着线索上下文进入对话。
      </p>

      {/* 载入中 */}
      {loading && !data && <div className="loading">载入中…</div>}

      {/* 未接入引导 */}
      {data && !data.enabled && (
        <div className="panel">
          <h3>商机平台未接入</h3>
          <p className="muted" style={{ marginBottom: 10 }}>
            面板数据来自商机平台（Lead
            Manager）只读接口。接入后可在这里查看高价值线索（MQL）并一键发起 AI 分析。
          </p>
          <ol className="muted" style={{ marginLeft: 20, marginBottom: 12, fontSize: 13 }}>
            <li>在「设置 → 商机平台」填写 base_url 与系统级 API Token（lm_pat_…）并启用</li>
            <li>重启后端生效（与 Agent 工具同语义，无热切换）</li>
          </ol>
          <Link className="btn primary sm" to="/settings">
            去设置
          </Link>
        </div>
      )}

      {/* 列表加载失败 */}
      {err && (
        <div className="panel">
          <div className="error" style={{ marginBottom: 10 }}>
            ! {err}
          </div>
          <button className="btn sm" onClick={() => load(page)}>
            重试
          </button>
        </div>
      )}

      {/* KPI 区：列表计数 + 平台统计（best-effort） */}
      {data?.enabled && !err && (
        <>
          <div className="dash-kpis">
            <div className="dash-kpi">
              <div className="dash-kpi-v">{total !== null ? total : "—"}</div>
              <div className="dash-kpi-l">高价值线索总数{total === null ? "（未知）" : ""}</div>
            </div>
            <div className="dash-kpi">
              <div className="dash-kpi-v">{data.count ?? 0}</div>
              <div className="dash-kpi-l">本页线索</div>
            </div>
            {kpis.map((k) => (
              <div className="dash-kpi" key={k.label} title={`统计字段：${k.label}`}>
                <div className="dash-kpi-v">{k.value}</div>
                <div className="dash-kpi-l">{k.label}</div>
              </div>
            ))}
          </div>
          {stats?.error && (
            <div className="muted" style={{ fontSize: 12.5, marginBottom: 10 }}>
              统计暂不可用：{stats.error}
            </div>
          )}

          {/* 线索卡列表 */}
          {items.length === 0 && !loading ? (
            <div className="panel muted">当前没有阶段为 mql 的高价值线索。</div>
          ) : (
            <div className="lead-list">
              {items.map((it, i) => {
                const id = firstStr(it, ID_KEYS);
                const title = firstStr(it, TITLE_KEYS) || (id ? `线索 ${id}` : "未命名线索");
                const stage = firstStr(it, STAGE_KEYS);
                const status = firstStr(it, STATUS_KEYS);
                const owner = firstStr(it, OWNER_KEYS);
                const contact = firstStr(it, CONTACT_KEYS);
                const phone = firstStr(it, PHONE_KEYS);
                const time = firstStr(it, TIME_KEYS);
                const followup = firstStr(it, FOLLOWUP_KEYS);
                const source = firstStr(it, SOURCE_KEYS);
                const industry = firstStr(it, INDUSTRY_KEYS);
                const amount = firstStr(it, AMOUNT_KEYS);
                const currency = firstStr(it, CURRENCY_KEYS) || "CNY";
                const prods = firstArr(it, PROD_KEYS);
                const tags = firstArr(it, TAG_KEYS);
                const desc = firstStr(it, DESC_KEYS);
                const chips = [...prods, ...tags].slice(0, 5);
                return (
                  <div className="lead-card" key={id || i}>
                    <div className="lead-main">
                      <div className="lead-title-row">
                        <span className="lead-title" title={title}>
                          {title}
                        </span>
                        {stage && <span className="lead-stage">{stage}</span>}
                        {status && <span className="lead-status">{status}</span>}
                      </div>
                      <div className="lead-meta">
                        {id && <span className="mono">{id.slice(0, 13)}</span>}
                        {owner && <span>负责人：{owner}</span>}
                        {contact && (
                          <span>
                            联系人：{contact}
                            {phone ? ` ${phone}` : ""}
                          </span>
                        )}
                        {source && <span>来源：{source}</span>}
                        {industry && <span>行业：{industry}</span>}
                        {amount && (
                          <span>
                            金额：{amount} {currency}
                          </span>
                        )}
                        {time && <span>创建：{fmtTime(time)}</span>}
                        {followup && <span>最近跟进：{fmtTime(followup)}</span>}
                      </div>
                      {chips.length > 0 && (
                        <div className="lead-chips">
                          {chips.map((c, j) => (
                            <span className="tag" key={j}>
                              {c}
                            </span>
                          ))}
                          {prods.length + tags.length > 5 && (
                            <span className="faint" style={{ fontSize: 11.5 }}>
                              +{prods.length + tags.length - 5}
                            </span>
                          )}
                        </div>
                      )}
                      {desc && (
                        <div className="lead-desc" title={desc}>
                          {desc.length > 140 ? desc.slice(0, 140) + "…" : desc}
                        </div>
                      )}
                    </div>
                    <button
                      className="btn primary ai-btn"
                      onClick={() => analyzeWithAI(it, navigate, lsKey("handoff"))}
                      title="带着这条线索的上下文进入对话，由 AI 一条龙分析"
                    >
                      ✨ AI 分析
                    </button>
                  </div>
                );
              })}
            </div>
          )}

          {/* 翻页（total 未知时按满页推断可翻下一页） */}
          {(page > 1 ||
            items.length === PAGE_SIZE ||
            (total !== null && page * PAGE_SIZE < total)) && (
            <div className="dash-pager">
              <button className="btn sm" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>
                上一页
              </button>
              <span className="muted" style={{ fontSize: 12.5 }}>
                第 {page} 页
                {total !== null ? ` / 共 ${Math.max(1, Math.ceil(total / PAGE_SIZE))} 页` : ""}
              </span>
              <button
                className="btn sm"
                disabled={total !== null ? page * PAGE_SIZE >= total : items.length < PAGE_SIZE}
                onClick={() => setPage((p) => p + 1)}
              >
                下一页
              </button>
            </div>
          )}
        </>
      )}
    </div>
  );
}
