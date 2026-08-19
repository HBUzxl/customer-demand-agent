import { useEffect, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import {
  ApiError,
  submitMessage,
  subscribeStream,
  cancelRun,
  truncateMessages,
  sessionRunningState,
  sessionGet,
  memoryList,
  customerCreate,
  reviewPending,
  reviewApprove,
  reviewReject,
} from "../api/client";
import { useAuth } from "../auth/AuthProvider";
import type { AnalysisResult, AgentEvent, MemoryEntry } from "../types";
import MarkdownView from "../components/MarkdownView";
import ResultCard from "../components/ResultCard";
import ToolTimeline from "../components/ToolTrace";
import AnchorNav, { type TocEntry } from "../components/TocRail";
import type { ToolTrace } from "../components/ToolTrace";
import Icon from "../components/Icon";

// Seg 是按时间顺序记录的一个执行分段（与模型执行过程一致：
// 思考→工具→中间说明→再思考→…→最终回答），避免不同轮次的思考/输出
// 被拼进同一块导致时序错乱。
type ReasoningSeg = { kind: "reasoning"; text: string };
type ToolSeg = { kind: "tool"; tc: ToolTrace };
type TextSeg = { kind: "text"; text: string };
type Seg = ReasoningSeg | ToolSeg | TextSeg;

// BodyBlock 是渲染层对 Seg 的分组：连续的「思考+工具」归并成一个可折叠的
// 过程块，text 段独立成正文块——content 是主体直接显示，思考/工具折叠（A1+B1）。
type ProcessBlock = { kind: "process"; segs: (ReasoningSeg | ToolSeg)[] };
type BodyBlock = ProcessBlock | { kind: "text"; text: string };

type CustomerMode = "unset" | "existing" | "new" | "none";
type CustomerDraft = {
  name: string;
  industry: string;
  scale: string;
  existingSecurity: string;
  notes: string;
};

// groupSegs 把 segs 按时序切分为正文块与过程块（保持交错顺序）。
function groupSegs(segs: Seg[]): BodyBlock[] {
  const blocks: BodyBlock[] = [];
  let proc: ProcessBlock | null = null;
  for (const s of segs) {
    if (s.kind === "text") {
      if (proc) {
        blocks.push(proc);
        proc = null;
      }
      if (s.text.trim()) blocks.push({ kind: "text", text: s.text });
    } else {
      if (!proc) proc = { kind: "process", segs: [] };
      proc.segs.push(s);
    }
  }
  if (proc) blocks.push(proc);
  return blocks;
}

// 最后一个 text 段：可能不是 segs 的最后一段——ask_user 等工具段可位于其后
// （模型先输出完整分析再调用询问工具）。流式/完成时它以「最终回答」身份渲染，
// 而非「中间说明」正文块，需按此查找并移出正文区。
function lastTextSeg(segs: Seg[]): TextSeg | null {
  for (let j = segs.length - 1; j >= 0; j--) {
    const s = segs[j];
    if (s.kind === "text") return s;
  }
  return null;
}

interface ChatMsg {
  id: string; // 稳定渲染 key（"m{seq}" 或 uid()）
  seq?: number; // 持久化序号（编辑重发截断用，本地新消息无）
  role: "user" | "assistant";
  text: string; // 最终回答（done.content；流式中末尾 text 分段实时充当）
  segs?: Seg[]; // 按序执行分段（思考/工具/中间说明）
  analysis?: AnalysisResult;
  streaming?: boolean;
  askUser?: {
    question: string;
    options: { label: string; value?: string; description?: string; input?: string }[];
  }; // F3 待答选项（input=选此项需用户补充的信息提示，前端先弹输入再回传）
}

const uid = () => Math.random().toString(36).slice(2);

// ANALYSIS_TOOL 与后端 agent.ToolAnalysisSubmit 对齐（ADR-013）。
const ANALYSIS_TOOL = "analysis_submit";

// ASK_USER_TOOL 与后端 agent.ToolAskUser 对齐（F3）。
const ASK_USER_TOOL = "ask_user";

// parseSubmitParams 从 analysis_submit 的参数还原结构化分析（回放用）。
function parseSubmitParams(params: string): AnalysisResult | undefined {
  try {
    const r = JSON.parse(params);
    if (r && r.demand_analysis) return r as AnalysisResult;
  } catch {
    /* 非 JSON 参数 */
  }
  return undefined;
}

// parseAskUserParams 从 ask_user 的参数还原「待答问题 + 选项」（刷新/回放恢复用）。
function parseAskUserParams(params: string):
  | {
      question: string;
      options: { label: string; value?: string; description?: string; input?: string }[];
    }
  | undefined {
  try {
    const r = JSON.parse(params);
    if (r && typeof r.question === "string" && Array.isArray(r.options)) {
      const opts = r.options.filter(
        (o: unknown) => o && typeof (o as { label?: unknown }).label === "string",
      );
      if (opts.length > 0) return { question: r.question, options: opts };
    }
  } catch {
    /* 非 JSON 参数 */
  }
  return undefined;
}

// ── 结尾问句的快捷回复（点击选择 or 直接输入）────────────────────
// 模型常以自然问句收尾（"要不要…？/有没有…？"）而未走 ask_user 工具。
// 检测到结尾问句就在回答下方渲染快捷回复卡：一键点选（chips）+ 自由输入。
const Q_TAIL =
  /(要不要|需不需要|是否需要|用不用|可不可以|可以吗|行不行|行吗|有没有|是否有|是不是|对不对|怎么样|如何|好吗|对吗|可否|吗|呢)[，,。.！!\s]*$/;

function looksLikeQuestion(text: string): boolean {
  const t = text.trim();
  if (!t) return false;
  if (/[?？]$/.test(t)) return true;
  return Q_TAIL.test(t);
}

// splitTrailingQuestion 把最终回答末尾的行动询问从正文拆出，交给消息底部的
// ReplyHelper 与选项一起渲染。否则自然语言问句在 ResultCard 之前、选项却在
// ResultCard 之后，会形成「问句 → 结构化分析 → 选项」的割裂体验。
//
// 优先识别「需要我/要不要我…」等行动邀请；否则按最后一个句末标点或换行
// 切分。只处理确实以问句结尾的回答，不改动正文中的普通疑问表达。
function splitTrailingQuestion(text: string): { body: string; question: string } {
  const t = text.trim();
  if (!looksLikeQuestion(t)) return { body: t, question: "" };

  const actionInvite = t.match(
    /(?:需要我|是否需要我|需不需要我|要不要我|要我|是否要我|可以帮你|要不要帮你|是否需要帮你)[^\n。！？!?；;]*[？?]\s*$/,
  );
  if (actionInvite?.index !== undefined) {
    return {
      body: t
        .slice(0, actionInvite.index)
        .replace(/[，,：:]\s*$/, "")
        .trim(),
      question: actionInvite[0].trim(),
    };
  }

  // 跳过结尾的问号，从后向前找上一处句子边界。
  let boundary = -1;
  for (let i = t.length - 2; i >= 0; i--) {
    if ("\n。！？!?；;".includes(t[i])) {
      boundary = i;
      break;
    }
  }
  if (boundary >= 0) {
    const question = t
      .slice(boundary + 1)
      .trim()
      .replace(/^(?:[-*+]\s+|\d+[.)、]\s*)/, "");
    if (question) {
      return {
        body: t.slice(0, boundary + 1).trim(),
        question,
      };
    }
  }
  return { body: "", question: t };
}

// quickChips 依据结尾问句给出常见的一键答语（点选即发送；输入框兜底自由作答）。
// 匹配取「全文最后一次命中的问句模式」——而不是扫整篇/切最后一小句：
// 正文较早出现的「是否有/要不要」等词不会被误判（例：开放问句
// 「下一步想干什么？」不再错误给出「有/没有」；长句无句号分隔也不受影响）。
const CHIP_PATTERNS: { re: RegExp; chips: string[] }[] = [
  { re: /要不要/g, chips: ["要", "不要"] },
  // 需不需要/是否需要/需要…吗 归并为「需要/不需要」自然答语，避免被「要不要」的「要/不要」误抢
  { re: /需不需要|是否需要|需要[^，,。.！!？?\n]{0,24}吗/g, chips: ["需要", "不需要"] },
  { re: /用不用/g, chips: ["用", "不用"] },
  { re: /可不可以|可以吗|行不行|行吗|可否|能不能|能否/g, chips: ["可以", "不行"] },
  { re: /会不会/g, chips: ["会", "不会"] },
  { re: /好不好|好吗/g, chips: ["好", "不好"] },
  { re: /有没有|是否有|是否/g, chips: ["有", "没有"] },
  { re: /是不是|对不对|对吗/g, chips: ["是", "不是"] },
  // 开放性问句（什么/如何/怎么/下一步…）：猜不出确定选项，给中性推进项，
  // 不再硬套「有/没有」这类二分选项。
  { re: /什么|如何|怎么|怎样|哪些|哪家|哪个|多少|何时|进展|下一步/g, chips: ["继续", "不用了"] },
];

function quickChips(text: string): string[] {
  let best: { idx: number; chips: string[] } | null = null;
  for (const p of CHIP_PATTERNS) {
    let last = -1;
    for (const m of text.matchAll(p.re)) {
      if (m.index !== undefined && m.index > last) last = m.index;
    }
    if (last >= 0 && (best === null || last > best.idx)) best = { idx: last, chips: p.chips };
  }
  return best ? best.chips : ["好的", "不用了", "再看看"];
}

// reconstruct 从会话历史重建消息流（新系统：assistant content 为自然语言，
// 工具调用按 message_id 精确归属到对应 assistant 消息（服务端持久化时写入），
// analysis 从该轮的 analysis_submit params 还原；不做老数据兼容——用户裁决）。
function reconstruct(
  messages: { id: number; role: string; content: string; seq: number }[],
  toolCalls: { message_id: number; tool_name: string; params: string; result: string }[],
): ChatMsg[] {
  const byMessage = new Map<number, ToolTrace[]>();
  for (const c of toolCalls) {
    const list = byMessage.get(c.message_id) || [];
    list.push({ tool: c.tool_name, params: c.params, result: c.result });
    byMessage.set(c.message_id, list);
  }
  const out: ChatMsg[] = [];
  for (const m of messages) {
    if (m.role === "user") {
      out.push({ id: `m${m.seq}`, seq: m.seq, role: "user", text: m.content });
    } else if (m.role === "assistant") {
      const msg: ChatMsg = { id: `m${m.seq}`, seq: m.seq, role: "assistant", text: m.content };
      // system 消息间的 assistant：工具按 message_id（数据库 id）归属
      const mine = byMessage.get(m.id);
      if (mine && mine.length > 0) {
        msg.segs = mine.map((tc) => ({ kind: "tool", tc }) as ToolSeg);
        for (let j = mine.length - 1; j >= 0; j--) {
          if (mine[j].tool === ANALYSIS_TOOL) {
            msg.analysis = parseSubmitParams(mine[j].params);
            break;
          }
        }
        // F3 刷新恢复：ask_user 的待答问题/选项随 tool_calls 持久化，这里还原到
        // 消息上——否则页面刷新后选项卡消失，用户无法继续点选/看到待确认项。
        for (let j = mine.length - 1; j >= 0; j--) {
          if (mine[j].tool === ASK_USER_TOOL) {
            const q = parseAskUserParams(mine[j].params);
            if (q) msg.askUser = q;
            break;
          }
        }
      }
      out.push(msg);
    }
  }
  return out;
}

const EXAMPLES = [
  {
    t: "商机 · 高价值线索",
    d: "查一下当前的 MQL 高价值线索，挑一条最值得跟的，看看详情，结合我们产品能力给个跟进建议",
  },
  { t: "电商 · CC 攻击", d: "大促网站被 CC 打慢，过等保二级" },
  { t: "政务 · 挂马篡改", d: "官网被挂马，要过等保三级" },
  { t: "金融 · API 滥用", d: "接口被恶意调用，数据泄露风险" },
];

export default function Conversation() {
  const { sessionId: routeSid } = useParams<{ sessionId: string }>();
  const navigate = useNavigate();
  const { lsKey } = useAuth();
  const [messages, setMessages] = useState<ChatMsg[]>([]);
  const [sessionCustomer, setSessionCustomer] = useState("");
  const [customerMode, setCustomerMode] = useState<CustomerMode>("unset");
  const [customerEntries, setCustomerEntries] = useState<MemoryEntry[]>([]);
  const [selectedCustomer, setSelectedCustomer] = useState("");
  const [customersLoading, setCustomersLoading] = useState(false);
  const [customerGateError, setCustomerGateError] = useState("");
  const [customerDraft, setCustomerDraft] = useState<CustomerDraft>({
    name: "",
    industry: "",
    scale: "",
    existingSecurity: "",
    notes: "",
  });
  const [branches, setBranches] = useState<string[]>([]);
  const [currentBranch, setCurrentBranch] = useState("");
  const [pendingReviews, setPendingReviews] = useState<
    { type: string; title: string; overlap_titles?: string[] }[]
  >([]);
  const [input, setInput] = useState(() => {
    // G2 草稿：进入时恢复该会话的草稿（新建会话无草稿）
    return "";
  });
  const [loading, setLoading] = useState(false); // 本视图内订阅进行中
  const [error, setError] = useState("");
  const [sessionId, setSessionId] = useState(routeSid || "");
  const [activeRun, setActiveRun] = useState<{ sid: string; rid: string } | null>(null); // F0：本视图发起/恢复订阅的 Run
  const scrollRef = useRef<HTMLDivElement>(null);
  // 右侧回答目录：扫描 assistant markdown 标题生成；active 当前可视标题
  const [toc, setToc] = useState<TocEntry[]>([]);
  const [activeToc, setActiveToc] = useState(-1);
  const tocRef = useRef<TocEntry[]>([]);
  tocRef.current = toc;
  const tocScanRaf = useRef(0);
  // 目录跳转锁定：平滑滚动期间抑制滚动重算，避免激活高亮在中间标题上串动
  const tocLockRef = useRef(false);
  const tocLockTimerRef = useRef(0);
  // 贴底跟随：true=贴底（自动滚底）；用户上滚离开底部 → false（接管，暂停自动贴底）
  const followBottomRef = useRef(true);
  const unsubRef = useRef<(() => void) | null>(null);
  const lastSeqRef = useRef<Map<string, number>>(new Map()); // SSE 游标（按会话隔离）
  const subSessionRef = useRef<string>(""); // 当前订阅的会话（防串台）
  const activeRunRef = useRef<string>(""); // 当前关注的 Run（事件过滤）
  const msgsRef = useRef<ChatMsg[]>([]);
  msgsRef.current = messages;

  useEffect(() => {
    // 路由变化第一件事：退旧订阅（防切换窗口旧会话事件写入新视图）
    if (subSessionRef.current !== routeSid) {
      unsubRef.current?.();
      unsubRef.current = null;
      activeRunRef.current = "";
      setLoading(false);
    }
    if (!routeSid) {
      stopSubscription();
      setMessages([]);
      setSessionId("");
      setSessionCustomer("");
      setCustomerMode("unset");
      setSelectedCustomer("");
      setCustomerGateError("");
      setCustomerDraft({ name: "", industry: "", scale: "", existingSecurity: "", notes: "" });
      setBranches([]);
      setCurrentBranch("");
      setActiveRun(null);
      return;
    }
    // 流式订阅进行中不做历史重载（切回自己会话）；切换到别的会话先停旧订阅
    let cancelled = false;
    (async () => {
      try {
        const d = await sessionGet(routeSid);
        if (cancelled) return;
        setMessages(reconstruct(d.messages || [], d.tool_calls || []));
        setSessionId(routeSid);
        setSessionCustomer(d.session?.customer || "");
        setBranches(d.branches || []);
        // 分支过滤视图：currentBranch 非空时按分支重拉
        if (currentBranch) {
          const fb = await sessionGet(routeSid, currentBranch);
          if (!cancelled) setMessages(reconstruct(fb.messages || [], fb.tool_calls || []));
        }
        // F0 切回恢复（竞态安全）：
        // a) running=true 且二次确认仍在跑 → 续订（带 rid 过滤）；
        // b) 查询窗口内 Run 完成（true→false）→ 重拉详情补最终答案；
        // c) running=false 但快照缺 assistant（快照先于完成拉取）→ 重拉补全。
        const running = await sessionRunningState(routeSid);
        if (cancelled) return;
        if (running.running) {
          const r2 = await sessionRunningState(routeSid);
          if (cancelled) return;
          if (r2?.running && r2?.run_id) {
            setActiveRun({ sid: routeSid, rid: r2.run_id });
            attachStream(routeSid, r2.run_id, { resume: true });
          } else {
            const d2 = await sessionGet(routeSid);
            if (cancelled) return;
            setMessages(reconstruct(d2.messages || [], d2.tool_calls || []));
          }
        } else {
          const sawUser = (d.messages || []).filter(
            (m: { role: string }) => m.role === "user",
          ).length;
          const sawAsst = (d.messages || []).filter(
            (m: { role: string }) => m.role === "assistant",
          ).length;
          if (sawUser > sawAsst) {
            const d2 = await sessionGet(routeSid);
            if (cancelled) return;
            setMessages(reconstruct(d2.messages || [], d2.tool_calls || []));
          }
        }
      } catch (e) {
        if (!cancelled) setError(e instanceof Error ? e.message : String(e));
      }
    })();
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [routeSid]);

  // 新会话先加载本租户客户目录。这里只读取 tenant runtime 的 customer 记忆，
  // 因此同租户成员共享、跨租户不可见。
  useEffect(() => {
    if (routeSid) return;
    let cancelled = false;
    setCustomersLoading(true);
    memoryList("customer", 0, 200)
      .then((d) => {
        if (cancelled) return;
        const items = (d.items || []).sort((a, b) => a.title.localeCompare(b.title, "zh-CN"));
        setCustomerEntries(items);
      })
      .catch((e) => {
        if (!cancelled)
          setCustomerGateError("客户列表加载失败：" + (e instanceof Error ? e.message : String(e)));
      })
      .finally(() => {
        if (!cancelled) setCustomersLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [routeSid]);

  // 组件卸载：只取消订阅（Run 在服务端继续，F0）
  useEffect(() => () => unsubRef.current?.(), []);

  useEffect(() => {
    // 用户上滚接管：暂停自动贴底（流式与静止态都不再动滚动位置），回底由 onScroll 恢复
    if (!followBottomRef.current) return;
    if (loading) {
      // 流式中：跟随滚底（即时跳转——smooth 动画每 chunk 重启，会与用户滚动打架）
      const el = scrollRef.current;
      if (el) el.scrollTop = el.scrollHeight;
      return;
    }
    // G8 静止态：恢复该会话的记忆位置（无记忆则滚底）
    const memo = scrollMemoRef.current.get(sessionId);
    if (memo !== undefined && messages.length < 30) {
      scrollRef.current?.scrollTo({ top: memo });
    } else {
      scrollRef.current?.scrollTo({ top: scrollRef.current.scrollHeight });
    }
  }, [messages, loading, sessionId]);

  // G2 草稿持久化：输入变化即存（key 含会话，cda:<tenant>:<user>: 前缀）；发送后清
  useEffect(() => {
    const key = lsKey(sessionId ? `draft-${sessionId}` : "draft-new");
    if (input) localStorage.setItem(key, input);
    else localStorage.removeItem(key);
  }, [input, sessionId, lsKey]);
  // 进入会话时恢复草稿
  useEffect(() => {
    const key = lsKey(sessionId ? `draft-${sessionId}` : "draft-new");
    const saved = localStorage.getItem(key);
    if (saved) setInput(saved);
    return () => {
      // 离开会话/卸载时保存草稿；空草稿不落盘（避免注销后 wipe 被复活出空 key）
      if (input) localStorage.setItem(key, input);
      else localStorage.removeItem(key);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sessionId]);

  // G8 滚动位置：离开会话前记录，回来恢复（新消息少于 3 条时才恢复，避免打断流式）
  const scrollMemoRef = useRef<Map<string, number>>(new Map());

  function patchLast(patch: (m: ChatMsg) => void) {
    setMessages((ms) => {
      if (!ms.length) return ms;
      const copy = [...ms];
      copy[copy.length - 1] = { ...copy[copy.length - 1] };
      patch(copy[copy.length - 1]);
      return copy;
    });
  }

  // 追加分段：同类尾部段合并（同轮增量），异类新起一段（保留真实时序）。
  function pushSeg(m: ChatMsg, seg: Seg) {
    const segs = m.segs ? [...m.segs] : [];
    const last = segs[segs.length - 1];
    const mergeable =
      last &&
      ((seg.kind === "reasoning" && last.kind === "reasoning") ||
        (seg.kind === "text" && last.kind === "text"));
    if (mergeable && last) {
      segs[segs.length - 1] = {
        ...last,
        text: (last as ReasoningSeg).text + (seg as ReasoningSeg).text,
      };
    } else {
      segs.push(seg);
    }
    m.segs = segs;
  }

  function handleEvent(e: AgentEvent, run: string) {
    // 多轮/切回防串台：只处理当前订阅 Run 的事件——replay 里历史 Run 的
    // 事件（含旧 done）带自身 x-run 归属，在这里被丢弃。
    if (activeRunRef.current && run && run !== activeRunRef.current) return;
    switch (e.type) {
      case "session":
        setSessionId(e.content || "");
        if (!routeSid && e.content) navigate(`/analyze/${e.content}`, { replace: true });
        break;
      case "reasoning":
        patchLast((m) => {
          if (e.text) pushSeg(m, { kind: "reasoning", text: e.text });
        });
        break;
      case "tool_call":
        patchLast((m) => {
          pushSeg(m, { kind: "tool", tc: { tool: e.tool || "", params: e.params || "" } });
        });
        break;
      case "tool_result":
        patchLast((m) => {
          if (!m.segs) return;
          const segs = m.segs.map((s) => ({ ...s }));
          // 从尾部找最近一个尚未回填结果的工具段（串行执行按序闭环）
          for (let j = segs.length - 1; j >= 0; j--) {
            const s = segs[j];
            if (s.kind === "tool" && !s.tc.result) {
              s.tc = { ...s.tc, result: e.result || "", cached: e.cached };
              m.segs = segs;
              // ResultCard 即时渲染（用户裁决：过程透明优先）：
              // analysis_submit 的结果到达时立刻解析其 params 展示卡片。
              if (e.tool === ANALYSIS_TOOL) {
                const analysis = parseSubmitParams(s.tc.params);
                if (analysis) m.analysis = analysis;
              }
              break;
            }
          }
        });
        break;
      case "content":
        // content 按时序入段：末尾 text 段实时充当主回答；
        // 后续又来工具时自动降为轨迹内的中间说明。
        patchLast((m) => {
          if (e.text) pushSeg(m, { kind: "text", text: e.text });
        });
        break;
      case "ask_user":
        patchLast((m) => {
          m.askUser = { question: e.question || "", options: e.options || [] };
        });
        break;
      case "done":
        patchLast((m) => {
          if (m.askUser) m.streaming = false; // 有待答问题时保持卡片展示
          // done.content 是最终答案（服务端已累积全程 content）。
          if (e.content) {
            m.text = e.content;
            // 最终回答已上移到下方主区（m.text）。流式期间末尾 text 段实时充当该回答，
            // 但 ask_user 等工具段可能位于其后（非最后一段），需从尾部向前找最后一个
            // text 段并移除——否则它仍会作为「中间说明」正文块再次渲染（重复回答 bug）。
            const segs = m.segs;
            if (segs && segs.length) {
              for (let j = segs.length - 1; j >= 0; j--) {
                if (segs[j].kind === "text") {
                  m.segs = segs.slice(0, j).concat(segs.slice(j + 1));
                  break;
                }
              }
            }
          }
          // analysis 以提交过的为准（tool_result 已即时渲染，此处最终定稿）。
          if (e.analysis) m.analysis = e.analysis;
          m.streaming = false;
        });
        stopSubscription();
        setActiveRun(null);
        activeRunRef.current = "";
        break;
      case "error":
        setError(e.error || "未知错误");
        patchLast((m) => {
          m.streaming = false;
        });
        stopSubscription();
        setActiveRun(null);
        activeRunRef.current = "";
        break;
    }
  }

  // ── F0 订阅模式：attach 到会话的 Run（事件 → 消息流）──
  // resume=true 时消息流里已有历史重建，事件只 patch 流式中的最后一条。
  function attachStream(sid: string, runID: string, opts?: { resume?: boolean }) {
    unsubRef.current?.();
    setLoading(true);
    subSessionRef.current = sid;
    activeRunRef.current = runID;
    if (!opts?.resume) {
      setMessages((m) => [...m, { id: uid(), role: "assistant", text: "", streaming: true }]);
    } else if (!msgsRef.current.length || !msgsRef.current[msgsRef.current.length - 1].streaming) {
      setMessages((m) => [...m, { id: uid(), role: "assistant", text: "", streaming: true }]);
    }
    // 游标按会话隔离；新 Run 用该会话已见最大 seq（上一轮 done 已记）——
    // 增量订阅天然跳过历史 Run 事件（run_id 过滤双保险见 handleEvent）
    const since = lastSeqRef.current.get(sid) || 0;
    unsubRef.current = subscribeStream(
      sid,
      since,
      handleEvent,
      (seq) => lastSeqRef.current.set(sid, seq),
      // 订阅故障（非主动退订）：结束 loading 并提示——Run 在服务端继续，
      // 切回会话可经恢复链路（running 查询+replay）重新接上。
      (err) => {
        setLoading(false);
        setError("事件流中断：" + err.message + "（任务在服务端继续，切回本会话可恢复）");
      },
    );
  }

  function stopSubscription() {
    unsubRef.current?.();
    unsubRef.current = null;
    setLoading(false);
  }

  // F1：显式停止（cancel Run + 结束本视图订阅；assistant 留中断态）。
  // 失败语义：404=Run 已结束（按停止收尾）；网络/5xx=停止失败，Run 可能
  // 仍在跑——保留订阅与运行态，仅提示，不谎报"已停止"。
  async function stopRun() {
    if (!activeRun) return;
    try {
      await cancelRun(activeRun.sid, activeRun.rid);
    } catch (err) {
      const status = err instanceof ApiError ? err.status : 0;
      if (status !== 404) {
        setError(
          "停止请求失败（任务可能仍在运行）：" + (err instanceof Error ? err.message : String(err)),
        );
        return; // 不退订、不标已停止——与真实运行态一致
      }
      // 404：Run 已自然结束，按完成收尾
    }
    stopSubscription();
    patchLast((m) => {
      m.streaming = false;
      // 中断态恒可辨：无内容时占位；已有部分内容时追加标记
      if (!m.text && !m.analysis) m.text = "（已停止）";
      else if (!/（已停止）$/.test(m.text)) m.text = m.text + "\n\n（已停止）";
    });
    setActiveRun(null);
  }

  // 编辑重发的分支续写标记（截断返回 branch_id 存这里，下一条 send 带上）
  const pendingBranchRef = useRef<string | null>(null);

  const splitCustomerValues = (value: string) =>
    value
      .split(/[,，;；\n]/)
      .map((item) => item.trim())
      .filter(Boolean);

  async function customerForFirstMessage(explicitCustomer?: string): Promise<{
    ok: boolean;
    customer?: string;
  }> {
    if (sessionId) return { ok: true };
    const explicit = explicitCustomer?.trim();
    if (explicit) return { ok: true, customer: explicit };
    if (customerMode === "unset") {
      setCustomerGateError("请先选择本次会话关联的客户，或选择“仅产品咨询”。");
      return { ok: false };
    }
    if (customerMode === "none") return { ok: true };
    if (customerMode === "existing") {
      if (!selectedCustomer) {
        setCustomerGateError("请选择一个已有客户。");
        return { ok: false };
      }
      return { ok: true, customer: selectedCustomer };
    }

    const name = customerDraft.name.trim();
    if (!name) {
      setCustomerGateError("登记新客户时，客户名称不能为空。");
      return { ok: false };
    }
    if (name.length > 120) {
      setCustomerGateError("客户名称不能超过 120 个字符。");
      return { ok: false };
    }
    if (customerEntries.some((entry) => entry.title.trim().toLowerCase() === name.toLowerCase())) {
      setCustomerGateError("该客户已存在，请切换到“已有客户”直接选择。");
      return { ok: false };
    }
    try {
      await customerCreate({
        name,
        industry: customerDraft.industry.trim(),
        scale: customerDraft.scale.trim(),
        existing_security: splitCustomerValues(customerDraft.existingSecurity),
        notes: customerDraft.notes.trim(),
      });
      setCustomerEntries((items) => [...items, { type: "customer", title: name }]);
      return { ok: true, customer: name };
    } catch (err) {
      setCustomerGateError(err instanceof Error ? err.message : String(err));
      return { ok: false };
    }
  }

  async function send(text: string, branch?: string, customer?: string) {
    const content = text.trim();
    if (!content || loading) return;
    const firstMessageCustomer = await customerForFirstMessage(customer);
    if (!firstMessageCustomer.ok) return;
    setCustomerGateError("");
    setError("");
    setMessages((m) => [...m, { id: uid(), role: "user", text: content }]);
    setInput("");
    setLoading(true);
    followBottomRef.current = true; // 新发消息 → 恢复贴底，让回复滚入视野
    try {
      const effBranch = branch || pendingBranchRef.current || currentBranch || undefined;
      pendingBranchRef.current = null;
      // customer 仅首轮生效（新会话绑定客户身份；后续轮以会话为准）
      const cust = !sessionId ? firstMessageCustomer.customer : undefined;
      const handle = await submitMessage(content, sessionId || undefined, cust, effBranch);
      setSessionId(handle.session_id);
      if (cust) setSessionCustomer(cust);
      setActiveRun({ sid: handle.session_id, rid: handle.run_id });
      if (!routeSid && handle.session_id)
        navigate(`/analyze/${handle.session_id}`, { replace: true });
      attachStream(handle.session_id, handle.run_id);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      setLoading(false);
    }
  }

  // 商机面板「一键 AI 分析」交接：进入空会话时自动发送（Dashboard 写入
  // sessionStorage，此处一次即焚——读后立即删，StrictMode 双跑安全；
  // 已有会话不自动发，避免误触发）。
  const handoffDoneRef = useRef(false);
  useEffect(() => {
    if (handoffDoneRef.current) return;
    handoffDoneRef.current = true;
    if (routeSid) return;
    const raw = sessionStorage.getItem(lsKey("handoff"));
    if (!raw) return;
    sessionStorage.removeItem(lsKey("handoff"));
    try {
      const h = JSON.parse(raw) as { text?: string; customer?: string };
      if (h.text && h.text.trim()) send(h.text, undefined, h.customer);
    } catch {
      /* 交接数据损坏则忽略，不阻塞正常对话 */
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // F1：编辑重发——截断该 user 消息及其之后（分叉出分支），原文填回输入框；
  // 分叉后的下一条消息写入该分支（checkpoint-tree：编辑重发=开分支续写）
  async function editResend(msg: ChatMsg) {
    if (loading) return;
    try {
      if (!msg.seq) {
        setMessages((m) => m.filter((x) => x.id !== msg.id));
        setInput(msg.text);
        return;
      }
      const r = await truncateMessages(sessionId, msg.seq); // 真实持久化 seq（≠数据库 id）
      pendingBranchRef.current = r.branch_id || null; // 分支续写标记
      const idx = messages.findIndex((m) => m.id === msg.id);
      if (idx >= 0) setMessages((m) => m.slice(0, idx));
      setInput(msg.text);
      setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  const onKey = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      send(input);
    }
  };
  const grow = (e: React.FormEvent<HTMLTextAreaElement>) => {
    const t = e.currentTarget;
    t.style.height = "auto";
    t.style.height = Math.min(t.scrollHeight, 160) + "px";
  };

  const composer = (
    <div className="composer">
      <textarea
        rows={1}
        value={input}
        onChange={(e) => setInput(e.target.value)}
        onKeyDown={onKey}
        onInput={grow}
        placeholder="输入消息，回车发送（Shift+回车换行）…"
        style={{ height: "auto" }}
      />
      <button
        className={`btn send${loading ? " stop" : " primary"}`}
        onClick={() => (loading ? stopRun() : send(input))}
        disabled={!loading && !input.trim()}
        aria-label={loading ? "停止" : "发送"}
        title={loading ? "停止生成" : "发送"}
      >
        {loading ? (
          <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor" aria-hidden>
            <rect x="6" y="6" width="12" height="12" rx="2" />
          </svg>
        ) : (
          <svg
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
          >
            <path d="M12 19V5M5 12l7-7 7 7" />
          </svg>
        )}
      </button>
    </div>
  );

  const isEmpty = messages.length === 0 && !loading;

  async function loadPending() {
    try {
      const d = await reviewPending();
      setPendingReviews(d.items || []);
    } catch {
      /* 静默 */
    }
  }
  async function actReview(type: string, title: string, approve: boolean) {
    try {
      if (approve) await reviewApprove(type, title);
      else await reviewReject(type, title);
    } finally {
      loadPending();
    }
  }
  useEffect(() => {
    loadPending();
    const t = setInterval(loadPending, 8000);
    return () => clearInterval(t);
  }, []);

  const custChip = sessionCustomer ? (
    <span className="cust-ro" title={sessionCustomer}>
      客户：{sessionCustomer}
    </span>
  ) : null;

  function onScroll() {
    const el = scrollRef.current;
    if (!el) return;
    if (sessionId) scrollMemoRef.current.set(sessionId, el.scrollTop);
    // 贴底判定：距底部 <24px 视为贴底（与轨迹框一致）；上滚离开 → 用户接管
    followBottomRef.current = el.scrollHeight - el.scrollTop - el.clientHeight < 24;
  }

  // ── 右侧回答目录：用户提问一级锚点 + assistant 标题子目录 ──
  // 每个用户提问 = 一级目录项（锚定气泡），该轮 assistant 标题整体下沉一级
  // 作为其子目录——提问多了按轮次成组，不扁平铺开（h1 不再与提问平级）。
  function rescanToc() {
    const root = scrollRef.current;
    if (!root) {
      setToc([]);
      return;
    }
    const rootTop = root.getBoundingClientRect().top;
    const entries: TocEntry[] = [];
    let prevWasQuestion = false;
    const blocks = Array.from(root.querySelectorAll<HTMLElement>(".chat-inner > .msg"));
    for (const block of blocks) {
      if (block.classList.contains("user")) {
        const bubble = block.querySelector<HTMLElement>(".bubble");
        if (!bubble) continue;
        const text = (bubble.textContent || "").trim();
        entries.push({
          id: `toc-q-${entries.length}`,
          level: 1,
          text: text || "提问",
          el: bubble,
          top: bubble.getBoundingClientRect().top - rootTop + root.scrollTop,
        });
        prevWasQuestion = true;
      } else {
        const heads = Array.from(
          block.querySelectorAll<HTMLElement>(
            ".markdown h1, .markdown h2, .markdown h3, .markdown h4",
          ),
        );
        for (const h of heads) {
          // 前面有提问 → 标题整体下沉一级，作为该提问的子目录（min 4 封顶）
          const base = prevWasQuestion ? 1 : 0;
          entries.push({
            id: `toc-h-${entries.length}-${(h.textContent || "").trim().slice(0, 24)}`,
            level: Math.min(Number(h.tagName.slice(1)) + base, 4),
            text: (h.textContent || "").trim(),
            el: h,
            top: h.getBoundingClientRect().top - rootTop + root.scrollTop,
          });
        }
        prevWasQuestion = false;
      }
    }
    setToc(entries);
  }
  function jumpToc(i: number) {
    const entry = tocRef.current[i];
    if (!entry?.el) return;
    followBottomRef.current = false; // 目录定位：用户接管，不再自动贴底
    // 锁定激活项直到平滑滚动结束：点击即选中目标，滚动过程不经过中间标题
    tocLockRef.current = true;
    window.clearTimeout(tocLockTimerRef.current);
    tocLockTimerRef.current = window.setTimeout(() => {
      tocLockRef.current = false;
    }, 2200); // scrollend 不可用时的兜底释放
    entry.el.scrollIntoView({ behavior: "smooth", block: "start" });
    setActiveToc(i);
    applyActiveHeading(tocRef.current, i); // 立即切换正文标题选中高亮
  }
  // 正文标题"选中"高亮：只在当前激活的标题上保持选中样式，切换时上一个复原
  function applyActiveHeading(list: TocEntry[], idx: number) {
    for (let i = 0; i < list.length; i++) {
      list[i].el.classList.toggle("toc-current", i === idx);
    }
  }

  // ── Scroll Spy（IntersectionObserver）：检测当前所在章节 ──
  // 不为滚动监听做高频 DOM 计算：给每个章节元素注册 IO，回调里按「顶线越过
  // 可视区中线」的最后一个条目判定 active（上卷时不越过中线则回退到第一条）。
  // 目录跳转锁定（tocLock）期间暂停更新，避免平滑滚动中高亮在中间标题串动。
  useEffect(() => {
    const root = scrollRef.current;
    if (!root || !toc.length) {
      setActiveToc(-1);
      return;
    }
    let latest = -1;
    const compute = () => {
      if (tocLockRef.current) return; // 跳转滚动中：保持目标高亮
      const mid = root.scrollTop + root.clientHeight * 0.5; // 可视区中线
      let idx = -1;
      for (let i = 0; i < toc.length; i++) {
        if (toc[i].top <= mid) idx = i;
        else break;
      }
      if (idx === -1 && toc.length) idx = 0;
      if (idx !== latest) {
        latest = idx;
        setActiveToc(idx);
        applyActiveHeading(toc, idx); // 正文标题选中高亮跟随切换
      }
    };
    compute(); // 目录/内容重扫后立即校准
    // 把观察区收窄到可视区中段（±45%）：章节穿过中段时触发回调，无需高频 scroll
    const io = new IntersectionObserver(compute, {
      root,
      rootMargin: "-45% 0px -45% 0px",
      threshold: [0, 0.5, 1],
    });
    toc.forEach((e) => io.observe(e.el));
    return () => io.disconnect();
  }, [toc, sessionId]);

  // 消息变化/流式结束后重新扫描标题；ResizeObserver 兜底内容高度变化
  // （轨迹折叠展开、图片加载等不触发 messages 变化但会位移下方标题）。
  useEffect(() => {
    const root = scrollRef.current;
    rescanToc();
    if (!root) return;
    const ro = new ResizeObserver(() => {
      if (tocScanRaf.current) return;
      tocScanRaf.current = requestAnimationFrame(() => {
        tocScanRaf.current = 0;
        rescanToc(); // 重扫后 toc 变化 → Scroll Spy effect 重跑校准 active
      });
    });
    ro.observe(root);
    // 目录跳转滚动自然结束（scrollend）或用户接管滚动（wheel/触摸）时解除激活锁定
    const unlock = () => {
      tocLockRef.current = false;
    };
    root.addEventListener("scrollend", unlock);
    root.addEventListener("wheel", unlock, { passive: true });
    root.addEventListener("touchstart", unlock, { passive: true });
    return () => {
      ro.disconnect();
      root.removeEventListener("scrollend", unlock);
      root.removeEventListener("wheel", unlock);
      root.removeEventListener("touchstart", unlock);
    };
  }, [messages, loading, sessionId]);

  // ── 空状态：欢迎语 + 输入框 + 示例，垂直居中 ──
  if (isEmpty) {
    return (
      <div className="empty-state">
        <div className="empty-inner">
          <h1>需求分析助手</h1>
          <p className="empty-sub">
            粘贴客户沟通原文，或直接描述场景。我会理解需求、匹配长亭产品、判断可行性——也可以直接跟我聊。
          </p>
          <section className="customer-start" aria-label="本次会话客户信息">
            <div className="customer-start-head">
              <div>
                <strong>本次会话归属</strong>
                <span>客户画像与会话会在当前租户内共享</span>
              </div>
              <em>发送前必选</em>
            </div>
            <div className="customer-mode-row" role="group" aria-label="客户归属方式">
              {(
                [
                  ["existing", "已有客户"],
                  ["new", "登记新客户"],
                  ["none", "仅产品咨询"],
                ] as const
              ).map(([mode, label]) => (
                <button
                  key={mode}
                  type="button"
                  className={customerMode === mode ? "active" : ""}
                  aria-pressed={customerMode === mode}
                  onClick={() => {
                    setCustomerMode(mode);
                    setCustomerGateError("");
                    if (mode === "existing" && !selectedCustomer && customerEntries.length > 0)
                      setSelectedCustomer(customerEntries[0].title);
                  }}
                >
                  {label}
                </button>
              ))}
            </div>
            {customerMode === "existing" && (
              <label className="customer-field customer-field-wide">
                <span>选择客户</span>
                <select
                  value={selectedCustomer}
                  disabled={customersLoading || customerEntries.length === 0}
                  onChange={(e) => setSelectedCustomer(e.target.value)}
                  aria-label="选择已有客户"
                >
                  <option value="">
                    {customersLoading
                      ? "正在加载…"
                      : customerEntries.length === 0
                        ? "暂无客户，请登记新客户"
                        : "请选择客户"}
                  </option>
                  {customerEntries.map((entry) => (
                    <option key={entry.title} value={entry.title}>
                      {entry.title}
                      {entry.summary ? ` · ${entry.summary}` : ""}
                    </option>
                  ))}
                </select>
              </label>
            )}
            {customerMode === "new" && (
              <div className="customer-form">
                <label className="customer-field">
                  <span>客户名称 *</span>
                  <input
                    value={customerDraft.name}
                    maxLength={120}
                    onChange={(e) => setCustomerDraft((d) => ({ ...d, name: e.target.value }))}
                    placeholder="例如：某某银行"
                    aria-label="新客户名称"
                  />
                </label>
                <label className="customer-field">
                  <span>行业</span>
                  <input
                    value={customerDraft.industry}
                    onChange={(e) => setCustomerDraft((d) => ({ ...d, industry: e.target.value }))}
                    placeholder="金融 / 制造 / 政企…"
                    aria-label="客户行业"
                  />
                </label>
                <label className="customer-field">
                  <span>规模</span>
                  <input
                    value={customerDraft.scale}
                    onChange={(e) => setCustomerDraft((d) => ({ ...d, scale: e.target.value }))}
                    placeholder="员工数、资产规模等"
                    aria-label="客户规模"
                  />
                </label>
                <label className="customer-field">
                  <span>已有安全建设</span>
                  <input
                    value={customerDraft.existingSecurity}
                    onChange={(e) =>
                      setCustomerDraft((d) => ({ ...d, existingSecurity: e.target.value }))
                    }
                    placeholder="多个产品用逗号分隔"
                    aria-label="客户已有安全建设"
                  />
                </label>
                <label className="customer-field customer-field-wide">
                  <span>背景与备注</span>
                  <textarea
                    rows={2}
                    value={customerDraft.notes}
                    onChange={(e) => setCustomerDraft((d) => ({ ...d, notes: e.target.value }))}
                    placeholder="关键痛点、采购偏好、历史事件等"
                    aria-label="客户背景与备注"
                  />
                </label>
              </div>
            )}
            {customerMode === "none" && (
              <p className="customer-none-note">本会话不关联客户画像，适合通用产品与方案咨询。</p>
            )}
            {customerGateError && <div className="customer-gate-error">{customerGateError}</div>}
          </section>
          <div className="composer-wrap">{composer}</div>
          <div className="examples-grid">
            {EXAMPLES.map((ex, i) => (
              <button
                key={i}
                className="ex-card"
                onClick={() => {
                  if (customerMode === "unset") setInput(ex.d);
                  void send(ex.d);
                }}
              >
                <div className="ex-t">{ex.t}</div>
                <div className="ex-d">{ex.d}</div>
              </button>
            ))}
          </div>
        </div>
      </div>
    );
  }

  // ── 对话中：消息流 + 底部输入 ──
  return (
    <div className="chat">
      <div className="chat-scroll" ref={scrollRef} onScroll={onScroll}>
        <div className="chat-inner">
          {messages.map((m) =>
            m.role === "user" ? (
              <div key={m.id} className="msg user msg-wrap">
                <div className="bubble">{m.text}</div>
                <div className="msg-actions">
                  <CopyBtn text={m.text} />
                  {!loading && (
                    <button
                      className="edit-resend"
                      onClick={() => editResend(m)}
                      title="编辑后重发"
                      aria-label="编辑后重发"
                    >
                      <svg
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        strokeWidth="2"
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        aria-hidden
                      >
                        <path d="M17 3a2.828 2.828 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5L17 3z" />
                      </svg>
                    </button>
                  )}
                </div>
              </div>
            ) : (
              <AssistantMsg key={m.id} m={m} onSend={send} {...askUserAnswered(messages, m.id)} />
            ),
          )}
          {error && <div className="error">! {error}</div>}
        </div>
      </div>
      <AnchorNav toc={toc} active={activeToc} onJump={jumpToc} />
      {pendingReviews.length > 0 && (
        <div className="review-strip">
          {pendingReviews.map((p) => (
            <div key={`${p.type}/${p.title}`} className="rs-item">
              <span className="rs-text">
                Agent 请求写入 [{p.type}]「{p.title}」
              </span>
              <button className="btn sm" onClick={() => actReview(p.type, p.title, true)}>
                批准
              </button>
              <button className="btn ghost sm" onClick={() => actReview(p.type, p.title, false)}>
                拒绝
              </button>
            </div>
          ))}
        </div>
      )}
      <div className="chat-input">
        <div className="inner">
          {composer}
          <div className="chat-hint">
            {sessionId ? `会话 ${sessionId.slice(5, 17)}` : "新会话"} · 回车发送
            {custChip}
            {branches.length > 1 &&
              branches.map((b) => (
                <button
                  key={b}
                  className={`chip ${b === currentBranch ? "active" : ""}`}
                  style={{
                    fontSize: 10,
                    padding: "1px 8px",
                    cursor: "pointer",
                    border: "1px solid var(--border)",
                    borderRadius: 8,
                    background:
                      b === currentBranch
                        ? "color-mix(in srgb, #7c6bd9 12%, transparent)"
                        : "transparent",
                    color: b === currentBranch ? "#7c6bd9" : "var(--text-faint)",
                  }}
                  onClick={() => {
                    setCurrentBranch(b === currentBranch ? "" : b);
                  }}
                  title={b === "main" ? "主线对话" : b}
                >
                  {b === "main" ? "主线" : b}
                </button>
              ))}
          </div>
        </div>
      </div>
    </div>
  );
}

function CopyBtn({ text }: { text: string }) {
  const [done, setDone] = useState(false);
  return (
    <button
      className={`msg-copy${done ? " done" : ""}`}
      title="复制"
      onClick={() => {
        navigator.clipboard.writeText(text).then(() => {
          setDone(true);
          setTimeout(() => setDone(false), 1200);
        });
      }}
    >
      {done ? <Icon name="check" size={15} /> : <Icon name="copy" size={15} />}
    </button>
  );
}

// renderProcess 渲染过程块内的思考段与工具时间线（text 段已作为正文块提出）。
function renderProcess(segs: (ReasoningSeg | ToolSeg)[], active: boolean) {
  const out: React.ReactNode[] = [];
  let i = 0;
  let k = 0;
  while (i < segs.length) {
    const s = segs[i];
    if (s.kind === "reasoning") {
      out.push(
        <div className="t-think" key={k++}>
          {s.text}
          {active && i === segs.length - 1 && <span className="cursor" />}
        </div>,
      );
      i++;
    } else {
      const group: ToolTrace[] = [];
      while (i < segs.length && segs[i].kind === "tool") {
        group.push((segs[i] as ToolSeg).tc);
        i++;
      }
      out.push(<ToolTimeline key={k++} tools={group} />);
    }
  }
  return out;
}

// ProcessFold 是可折叠的过程块（安全阶段状态 + 工具调用）。active=true 表示
// 正在流式进行中；后端不会下发模型原始思维链。
function ProcessFold({ segs, active }: { segs: (ReasoningSeg | ToolSeg)[]; active: boolean }) {
  const [open, setOpen] = useState(false);
  const boxRef = useRef<HTMLDivElement>(null);
  const stickyRef = useRef(true);
  useEffect(() => {
    const el = boxRef.current;
    if (el && stickyRef.current) el.scrollTop = el.scrollHeight;
  }, [segs, open]);
  function onBoxScroll() {
    const el = boxRef.current;
    if (!el) return;
    stickyRef.current = el.scrollHeight - el.scrollTop - el.clientHeight < 24;
  }
  const thinkCount = segs.filter((s) => s.kind === "reasoning").length;
  const toolCount = segs.filter((s) => s.kind === "tool").length;
  const expanded = open || active;
  return (
    <>
      <div
        className={`trace-toggle${active ? " thinking" : ""}`}
        onClick={() => setOpen((x) => !x)}
      >
        <span className="dot" />
        {active ? "处理中…" : "执行过程"}
        {thinkCount > 0 && ` · ${thinkCount} 个处理阶段`}
        {toolCount > 0 && ` · ${toolCount} 次工具调用`}
        <Icon name={open ? "chevron-down" : "chevron-right"} size={14} style={{ marginLeft: 2 }} />
      </div>
      {expanded && (
        <div className="trace-box" ref={boxRef} onScroll={onBoxScroll}>
          {renderProcess(segs, active)}
        </div>
      )}
    </>
  );
}

// askUserAnswered：该 assistant 消息之后是否已有用户答复（点选/输入即产生下一条 user 消息）。
// 有 → 选项只读展示（命中的那项高亮），不再可点，也不显示「或直接输入」框。
function askUserAnswered(
  msgs: ChatMsg[],
  id: string,
): { askAnswered: boolean; askAnsweredBy: string } {
  const idx = msgs.findIndex((m) => m.id === id);
  if (idx < 0) return { askAnswered: false, askAnsweredBy: "" };
  for (let j = idx + 1; j < msgs.length; j++) {
    if (msgs[j].role === "user") {
      return { askAnswered: true, askAnsweredBy: msgs[j].text };
    }
  }
  return { askAnswered: false, askAnsweredBy: "" };
}

// NEW_CUSTOMER_RE 启发兜底：模型未按规范带 input 时，识别「新建客户」类选项。
const NEW_CUSTOMER_RE = /(新建客户|新客户|创建客户|添加客户|新增客户)/;

// ReplyHelper 结尾问句的回复辅助：选项/快捷答语 chips + 自由输入。
// 点选或回车发送后即收起（防重复发送）；输入框兜底自定义回答。
// 选项带 input（或启发识别为「新建客户」）时：点击先弹内联输入框，
// 确认后以「选项文案：用户输入」回传（如"新建客户：张三"），Agent 据此画像。
function ReplyHelper({
  question,
  options,
  chips,
  onSend,
  answered = false,
  answeredBy = "",
}: {
  question?: string;
  options: { label: string; value?: string; description?: string; input?: string }[];
  chips: string[];
  onSend: (t: string) => void;
  answered?: boolean;
  answeredBy?: string;
}) {
  const [sent, setSent] = useState(false);
  const [sentText, setSentText] = useState("");
  const [val, setVal] = useState("");
  // inputReq：选中了带 input 的选项，等待用户补充信息（placeholder + 基础文案）
  const [inputReq, setInputReq] = useState<{
    label: string;
    base: string;
    placeholder: string;
  } | null>(null);
  const [inputVal, setInputVal] = useState("");
  const doSend = (t: string) => {
    const text = t.trim();
    if (sent || !text) return;
    setSent(true);
    setSentText(text);
    onSend(text);
  };
  // 选项被点击：带 input 或疑似新建客户 → 弹内联输入；否则直接回传。
  // 始终回传展示文案 label（value 是机器码，直接发会导致"小→small"之类英文泄露）
  const pickOption = (o: {
    label: string;
    value?: string;
    description?: string;
    input?: string;
  }) => {
    if (sent) return;
    if (o.input || NEW_CUSTOMER_RE.test(o.label)) {
      const label = o.label;
      const base = label;
      setInputReq({
        label,
        base,
        placeholder: o.input || "请输入客户名称",
      });
      return;
    }
    doSend(o.label);
  };
  // 内联输入确认：「选项文案：用户输入」回传（Agent 从回传解析出客户名）
  const submitInput = (e: React.FormEvent) => {
    e.preventDefault();
    const v = inputVal.trim();
    if (sent || !inputReq || !v) return;
    setSent(true);
    setSentText(`${inputReq.base}：${v}`);
    onSend(`${inputReq.base}：${v}`);
  };
  // done：本卡已回复（本组件发送过，或父层检测到后续已有用户消息）。
  // 展示只读——选项/chips 灰化不可点，命中的那项高亮（✓），不再渲染输入框。
  const done = sent || answered;
  const chosen = (label: string) => {
    const t = (sentText || answeredBy || "").trim();
    return t === label || t.startsWith(label + "：") || t.startsWith(label + ":");
  };
  if (done) {
    return (
      <div className="reply-helper read-only">
        {question && (
          <div className="ask-q">
            <Icon name="question" size={14} />
            {question}
          </div>
        )}
        {(options.length > 0 || chips.length > 0) && (
          <div className="ask-opts">
            {options.map((o, i) => (
              <span
                key={`o${i}`}
                className={`ask-opt read-only${chosen(o.label) ? " chosen" : ""}`}
                title={o.description}
              >
                {o.label}
                {chosen(o.label) && <Icon name="check" size={12} className="opt-check" />}
              </span>
            ))}
            {chips.map((c, i) => (
              <span
                key={`c${i}`}
                className={`ask-opt rh-chip read-only${chosen(c) ? " chosen" : ""}`}
              >
                {c}
                {chosen(c) && <Icon name="check" size={12} className="opt-check" />}
              </span>
            ))}
          </div>
        )}
        <div className="rh-done">
          <Icon name="check" size={12} /> 已收到回复
        </div>
      </div>
    );
  }
  // 内联输入模式：替换整个卡片为输入+取消（其余选项已不可点，一次补充即可）
  if (inputReq) {
    return (
      <div className="reply-helper">
        <div className="ask-q">
          <Icon name="pencil" size={14} />
          {inputReq.label}
        </div>
        <form className="rh-input" onSubmit={submitInput}>
          <input
            autoFocus
            value={inputVal}
            onChange={(e) => setInputVal(e.target.value)}
            placeholder={inputReq.placeholder}
            maxLength={200}
            autoComplete="off"
          />
          <button type="submit" className="btn sm primary" disabled={!inputVal.trim()}>
            确认
          </button>
          <button type="button" className="btn sm" onClick={() => setInputReq(null)}>
            取消
          </button>
        </form>
        <div className="rh-note">将以「{inputReq.base}：你的输入」发送，帮助 Agent 识别对象</div>
      </div>
    );
  }
  return (
    <div className="reply-helper">
      {question && (
        <div className="ask-q">
          <Icon name="question" size={14} />
          {question}
        </div>
      )}
      {(options.length > 0 || chips.length > 0) && (
        <div className="ask-opts">
          {options.map((o, i) => (
            <button
              key={`o${i}`}
              className="ask-opt"
              onClick={() => pickOption(o)}
              title={o.description}
            >
              {o.label}
              {o.input && <Icon name="pencil" size={11} className="ask-need" />}
            </button>
          ))}
          {chips.map((c, i) => (
            <button key={`c${i}`} className="ask-opt rh-chip" onClick={() => doSend(c)}>
              {c}
            </button>
          ))}
        </div>
      )}
      <form
        className="rh-input"
        onSubmit={(e) => {
          e.preventDefault();
          doSend(val);
        }}
      >
        <input
          value={val}
          onChange={(e) => setVal(e.target.value)}
          placeholder="或直接输入回复…"
          maxLength={500}
          autoComplete="off"
        />
        <button type="submit" className="btn sm" disabled={!val.trim()}>
          发送
        </button>
      </form>
    </div>
  );
}

function AssistantMsg({
  m,
  onSend,
  askAnswered = false,
  askAnsweredBy = "",
}: {
  m: ChatMsg;
  onSend: (t: string) => void;
  askAnswered?: boolean;
  askAnsweredBy?: string;
}) {
  const segs = m.segs || [];
  const trailingText = lastTextSeg(segs);
  // 末尾 text 段（不限于最后一段，工具段可能在其后）流式时实时充当最终回答（下方渲染）；
  // 其余段按时序交错渲染——content 作为正文块直接显示，思考/工具归并为可折叠的过程块（A1+B1+C1）。
  const bodySegs = m.streaming && trailingText ? segs.filter((s) => s !== trailingText) : segs;
  const blocks = groupSegs(bodySegs);
  // 最后一个过程块是流式进行中的那个（标题"思考中" + 光标）
  let lastProcessIdx = -1;
  for (let i = blocks.length - 1; i >= 0; i--) {
    if (blocks[i].kind === "process") {
      lastProcessIdx = i;
      break;
    }
  }
  const answerText = m.text || trailingText?.text || "";
  // ask_user 有显式问题时以工具参数为准；同时仍从正文移除末尾自然问句，避免
  // 同一句在 ResultCard 上方和底部操作区重复出现。没有显式工具时，拆出的
  // 自然问句既作为底部标题，也用于生成快捷选项。
  const trailingQuestion =
    m.askUser || !m.streaming
      ? splitTrailingQuestion(answerText)
      : { body: answerText, question: "" };
  const displayedAnswer = trailingQuestion.body;
  const replyQuestion = m.askUser?.question?.trim() || trailingQuestion.question;
  return (
    <div className="msg assistant msg-wrap" style={{ position: "relative" }}>
      <CopyBtn text={answerText} />
      <div className="avatar">需求分析助手</div>
      <div className="body">
        {blocks.map((b, i) =>
          b.kind === "process" ? (
            <ProcessFold
              key={`p${i}`}
              segs={b.segs}
              active={!!m.streaming && i === lastProcessIdx}
            />
          ) : (
            <div className="mid-content" key={`t${i}`}>
              <MarkdownView>{b.text}</MarkdownView>
            </div>
          ),
        )}
        {/* 最终回答：置于所有过程块之后；有待答问题时以问句卡收尾，不再显示"正在回应"占位 */}
        {displayedAnswer ? (
          <MarkdownView>{displayedAnswer}</MarkdownView>
        ) : m.streaming && !m.askUser ? (
          <span className="muted">
            正在回应…
            <span className="cursor" />
          </span>
        ) : null}
        {m.analysis && <ResultCard r={m.analysis} />}
        {/* 结尾问句 → 快捷回复卡：模型 ask_user 选项 + 自然问句启发 chips，均可自由输入。
            置于消息最末（答案/结果之后），与对话结尾连贯——用户无需上滚到思考/执行过程去选择 */}
        {(m.askUser || (!m.streaming && !!trailingQuestion.question)) && (
          <ReplyHelper
            question={replyQuestion}
            options={m.askUser?.options || []}
            chips={m.askUser ? [] : quickChips(trailingQuestion.question)}
            onSend={onSend}
            answered={askAnswered}
            answeredBy={askAnsweredBy}
          />
        )}
      </div>
    </div>
  );
}
