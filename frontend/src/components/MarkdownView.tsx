import ReactMarkdown, { type Components } from "react-markdown";
import remarkGfm from "remark-gfm";
import rehypeRaw from "rehype-raw";
import rehypeSanitize from "rehype-sanitize";
import MermaidDiagram from "./MermaidDiagram";

// markdown 自定义组件：mermaid 代码块渲染成图表，图片断链隐藏
// 安全基线：rehype-sanitize 必须在 rehype-raw 之后（勿回退）。
const mdComponents: Components = {
  code({ className, children }) {
    const match = /language-(\w+)/.exec(className || "");
    if (match && match[1] === "mermaid") {
      return <MermaidDiagram code={String(children).replace(/\n$/, "")} />;
    }
    return <code className={className}>{children}</code>;
  },
  img({ node: _node, src, alt, ...rest }) {
    return (
      <img
        {...rest}
        src={src}
        alt={alt || ""}
        loading="lazy"
        onError={(e) => {
          e.currentTarget.style.display = "none";
        }}
      />
    );
  },
};

// MarkdownView 统一的 markdown 渲染组件（对话气泡 / 记忆详情共用）。
export default function MarkdownView({ children }: { children: string }) {
  return (
    <div className="markdown">
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        rehypePlugins={[rehypeRaw, rehypeSanitize]}
        components={mdComponents}
      >
        {children}
      </ReactMarkdown>
    </div>
  );
}
