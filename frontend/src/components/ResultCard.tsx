import type { AnalysisResult } from "../types";
import MarkdownView from "./MarkdownView";

const feasLabel = (f: string) =>
  (
    ({ direct: "直接覆盖", custom: "需定制", partner: "外部整合", reject: "不建议接" }) as Record<
      string,
      string
    >
  )[f] || f;
const confClass = (c: number) => (c >= 0.85 ? "g" : c >= 0.5 ? "b" : c >= 0.3 ? "y" : "r");

/** 需求分析结果卡片（对话页/回放页共用）。 */
export default function ResultCard({ r }: { r: AnalysisResult }) {
  return (
    <div className="result">
      <div className="r-trigger">
        Agent 判定本轮为<b>真实客户需求</b>，已提交结构化分析（可展开系统提示词行查看判定上下文）
      </div>
      <div className="r-block">
        <h4>需求理解</h4>
        <MarkdownView>{r.demand_analysis}</MarkdownView>
      </div>
      <div className="r-block">
        <h4>可行性判定</h4>
        <div className="feas">
          <span className={`badge ${r.feasibility}`}>{feasLabel(r.feasibility)}</span>
          {r.feasibility_detail && (
            <span className="muted" style={{ fontSize: 13 }}>
              {r.feasibility_detail}
            </span>
          )}
        </div>
      </div>
      {r.matched_products && r.matched_products.length > 0 && (
        <div className="r-block">
          <h4>匹配产品（{r.matched_products.length}）</h4>
          {r.matched_products.map((p, i) => (
            <div key={i} className="prod">
              <div className="row">
                <span className="pn">{p.name}</span>
                <span className="pc right">{Math.round(p.confidence * 100)}%</span>
              </div>
              <div className="bar" style={{ width: 120, marginTop: 5 }}>
                <i
                  className={confClass(p.confidence)}
                  style={{ width: `${Math.round(p.confidence * 100)}%` }}
                />
              </div>
              <div className="pr">{p.reason}</div>
              {p.suggestion && <div className="ps">{p.suggestion}</div>}
            </div>
          ))}
        </div>
      )}
      {r.missing_info && r.missing_info.length > 0 && (
        <div className="r-block">
          <h4>待追问</h4>
          <ul className="missing">
            {r.missing_info.map((s, i) => (
              <li key={i}>{s}</li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}
