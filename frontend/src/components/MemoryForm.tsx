import { useState } from "react";
import type { MemoryEntry } from "../types";

const TYPES = ["product", "threat", "compliance", "industry", "customer", "user"];
const typeLabel: Record<string, string> = {
  product: "产品",
  threat: "威胁",
  compliance: "合规",
  industry: "行业",
  customer: "客户",
  user: "使用者",
};

export interface MemoryValues {
  type: string;
  title: string;
  summary: string;
  content: string;
  tags: string;
  aliases: string;
}

export function toValues(e: Partial<MemoryEntry>): MemoryValues {
  return {
    type: e.type || "customer",
    title: e.title || "",
    summary: e.summary || "",
    content: e.content || "",
    tags: (e.tags || []).join(", "),
    aliases: (e.aliases || []).join(", "),
  };
}

export function fromValues(
  v: MemoryValues,
): Partial<MemoryEntry> & { type: string; title: string } {
  return {
    type: v.type,
    title: v.title.trim(),
    summary: v.summary.trim(),
    content: v.content,
    tags: v.tags
      .split(/[,，]/)
      .map((s) => s.trim())
      .filter(Boolean),
    aliases: v.aliases
      .split(/[,，]/)
      .map((s) => s.trim())
      .filter(Boolean),
    // 不传 status：人工通道一律 verified（P10——pending 是 AI 写入专属语义）
  };
}

// MemoryForm —— 新建 / 编辑共用表单
export default function MemoryForm({
  initial,
  submitting,
  onSubmit,
  onCancel,
}: {
  initial: MemoryValues;
  submitting: boolean;
  onSubmit: (v: MemoryValues) => void;
  onCancel: () => void;
}) {
  const [v, setV] = useState<MemoryValues>(initial);
  const [formErr, setFormErr] = useState("");
  const set = (patch: Partial<MemoryValues>) => setV({ ...v, ...patch });

  function submit() {
    if (!v.title.trim() || !v.content.trim()) {
      setFormErr("标题和内容不能为空");
      return;
    }
    onSubmit(v);
  }

  return (
    <div className="panel">
      <div className="row" style={{ marginBottom: 12 }}>
        <div style={{ flex: "0 0 150px" }}>
          <label>类型</label>
          <select value={v.type} onChange={(e) => set({ type: e.target.value })}>
            {TYPES.map((t) => (
              <option key={t} value={t}>
                {typeLabel[t]}
              </option>
            ))}
          </select>
        </div>
        <div className="col">
          <label>标题（唯一标识）</label>
          <input
            value={v.title}
            onChange={(e) => set({ title: e.target.value })}
            placeholder="如 雷池 / CC攻击 / 某客户"
          />
        </div>
      </div>
      <div style={{ marginBottom: 12 }}>
        <label>摘要（一句话，可选）</label>
        <input
          value={v.summary}
          onChange={(e) => set({ summary: e.target.value })}
          placeholder="简短描述"
        />
      </div>
      <div style={{ marginBottom: 12 }}>
        <label>内容（Markdown 正文）</label>
        <textarea
          rows={10}
          value={v.content}
          onChange={(e) => set({ content: e.target.value })}
          placeholder="产品能力 / 威胁描述 / 客户画像…"
        />
      </div>
      <div className="row" style={{ marginBottom: 12 }}>
        <div className="col">
          <label>标签（逗号分隔）</label>
          <input
            value={v.tags}
            onChange={(e) => set({ tags: e.target.value })}
            placeholder="WAF, 边界安全"
          />
        </div>
        <div className="col">
          <label>别名（逗号分隔）</label>
          <input
            value={v.aliases}
            onChange={(e) => set({ aliases: e.target.value })}
            placeholder="SafeLine, WAF"
          />
        </div>
      </div>
      <div className="row">
        <button className="btn" onClick={submit} disabled={submitting}>
          {submitting ? "保存中…" : "保存"}
        </button>
        <button className="btn ghost" onClick={onCancel}>
          取消
        </button>
        <span className="faint mono" style={{ fontSize: 12 }}>
          人工写入，跳过 AI 权限校验
        </span>
      </div>
    </div>
  );
}
