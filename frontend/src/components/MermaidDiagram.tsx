import { useEffect, useId, useRef, useState } from "react";
import mermaid from "mermaid";

// 初始化 mermaid（浅色主题，适配当前 UI）
mermaid.initialize({
  startOnLoad: false,
  securityLevel: "strict", // 输入来自不可信记忆内容，显式 pin sanitize（mermaid 历史有绕过）
  theme: "neutral",
  themeVariables: {
    primaryColor: "#EBF0FC",
    primaryBorderColor: "#2563eb",
    primaryTextColor: "#1f2328",
    lineColor: "#4b5563",
    fontFamily: "-apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif",
  },
  flowchart: { useMaxWidth: true },
});

// MermaidDiagram：把 ```mermaid 代码块渲染成 SVG 图表
export default function MermaidDiagram({ code }: { code: string }) {
  const [svg, setSvg] = useState("");
  const [err, setErr] = useState("");
  const reactId = useId();
  const idRef = useRef(`mmd-${reactId.replace(/:/g, "")}`);

  useEffect(() => {
    let cancelled = false;
    setSvg("");
    setErr("");
    mermaid
      .render(idRef.current, code)
      .then(({ svg }) => {
        if (!cancelled) setSvg(svg);
      })
      .catch((e) => {
        if (!cancelled) setErr(e?.message || String(e));
      });
    return () => {
      cancelled = true;
    };
  }, [code]);

  if (err) {
    return (
      <pre
        className="markdown"
        style={{
          background: "var(--red-soft)",
          color: "var(--red)",
          padding: "10px 12px",
          borderRadius: 6,
          fontSize: 12,
          whiteSpace: "pre-wrap",
        }}
      >
        mermaid 渲染失败：{err}
      </pre>
    );
  }
  if (!svg)
    return (
      <div className="faint" style={{ padding: "12px 0" }}>
        渲染图表中…
      </div>
    );
  return <div className="mermaid" dangerouslySetInnerHTML={{ __html: svg }} />;
}
