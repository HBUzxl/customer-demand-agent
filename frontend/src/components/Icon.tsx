import type { CSSProperties } from "react";

/**
 * 全站共享 SVG 图标（stroke 风格：24×24 viewBox、fill none、stroke currentColor、
 * 圆头圆角、strokeWidth 1.6）。命名图标用 <Icon name="check" />，任意路径用 <Icon d="…" />。
 */
const ICONS: Record<string, string> = {
  copy: "M8 8h10a1 1 0 0 1 1 1v10a1 1 0 0 1-1 1H8a1 1 0 0 1-1-1V9a1 1 0 0 1 1-1zM6 16H5a1 1 0 0 1-1-1V5a1 1 0 0 1 1-1h10a1 1 0 0 1 1 1v1",
  check: "M20 6L9 17l-5-5",
  cross: "M18 6L6 18M6 6l12 12",
  question: "M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3M12 17h.01",
  pencil: "M12 20h9M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4z",
  "chevron-down": "M6 9l6 6 6-6",
  "chevron-right": "M9 6l6 6-6 6",
  "chevron-up": "M18 15l-6-6-6 6",
  "dots-v":
    "M12 5a1 1 0 1 0 0 2 1 1 0 0 0 0-2zm0 6a1 1 0 1 0 0 2 1 1 0 0 0 0-2zm0 6a1 1 0 1 0 0 2 1 1 0 0 0 0-2z",
  plus: "M12 5v14M5 12h14",
};

interface IconProps {
  /** 命名图标名（见 ICONS）；与 d 二选一，name 优先 */
  name?: keyof typeof ICONS;
  /** 裸路径（兼容原 Ico 用法） */
  d?: string;
  size?: number;
  strokeWidth?: number;
  className?: string;
  style?: CSSProperties;
}

export default function Icon({
  name,
  d,
  size = 18,
  strokeWidth = 1.6,
  className,
  style,
}: IconProps) {
  const path = name ? ICONS[name] : d;
  if (!path) return null;
  return (
    <svg
      className={`icon${className ? ` ${className}` : ""}`}
      style={style}
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={strokeWidth}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d={path} />
    </svg>
  );
}
