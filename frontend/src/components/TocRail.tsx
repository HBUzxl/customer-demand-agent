// AnchorNav：右侧悬浮章节目录。默认态极简——每个章节一条短横线（当前深灰蓝、
// 其余浅灰），存在感极低；鼠标悬浮整个导航容器时，从右缘滑出目录抽屉（章节标题、
// 层级缩进、点击跳转、当前高亮），抽屉右上角图钉可固定为展开态。fixed 定位垂直
// 居中于页面右侧，不占用正文布局。滚动联动（Active 检测）由 Conversation.tsx 用
// IntersectionObserver 驱动（不监听 scroll 高频计算）。
import { useState } from "react";

export type TocEntry = {
  id: string;
  level: number; // 1..4
  text: string;
  el: HTMLElement;
  top: number; // 相对滚动容器的滚动偏移（px），用于 Active 定位
};

export default function AnchorNav({
  toc,
  active,
  onJump,
}: {
  toc: TocEntry[];
  active: number;
  onJump: (i: number) => void;
}) {
  const [pinned, setPinned] = useState(false);
  if (!toc.length) return null;
  return (
    <nav className={`anchor-nav${pinned ? " pinned" : ""}`} aria-label="章节目录">
      {/* 默认态：每条目一条短横线，弱化存在感（仅不显示目录时显示） */}
      <div className="anchor-lines">
        {toc.map((e, i) => (
          <button
            key={e.id}
            className={`anchor-item${i === active ? " active" : ""}`}
            onClick={() => onJump(i)}
            title={e.text}
            aria-label={`定位到：${e.text}`}
          >
            <span className="anchor-line" />
          </button>
        ))}
      </div>
      {/* 悬浮/固定态：从右缘滑出的目录抽屉（标题 + 层级缩进 + 当前高亮 + 图钉固定） */}
      <div className="anchor-panel">
        <button
          className={`anchor-pin${pinned ? " pinned" : ""}`}
          aria-pressed={pinned}
          aria-label={pinned ? "取消固定目录" : "固定目录"}
          title={pinned ? "取消固定目录" : "固定目录"}
          onClick={() => setPinned((p) => !p)}
        >
          <svg
            viewBox="0 0 24 24"
            fill={pinned ? "currentColor" : "none"}
            stroke="currentColor"
            strokeWidth="1.7"
            strokeLinecap="round"
            strokeLinejoin="round"
          >
            <path d="M16 9V4h1c.55 0 1-.45 1-1s-.45-1-1-1H7c-.55 0-1 .45-1 1s.45 1 1 1h1v5c0 1.66-1.34 3-3 3v2h5.97v7l1 1 1-1v-7H19v-2c-1.66 0-3-1.34-3-3z" />
          </svg>
        </button>
        <div className="anchor-items">
          {toc.map((e, i) => (
            <button
              key={e.id}
              className={`anchor-navitem lv${e.level}${i === active ? " active" : ""}`}
              style={e.level === 1 && i > 0 ? { marginTop: 12 } : undefined}
              onClick={() => onJump(i)}
              title={e.text}
            >
              <span className="anchor-navitem-text">{e.text || "（无标题）"}</span>
            </button>
          ))}
        </div>
      </div>
    </nav>
  );
}
