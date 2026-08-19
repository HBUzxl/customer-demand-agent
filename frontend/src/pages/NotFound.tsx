import { Link } from "react-router-dom";
import Icon from "../components/Icon";

export default function NotFound() {
  return (
    <div className="page">
      <div className="panel" style={{ textAlign: "center", padding: "60px 20px" }}>
        <div
          className="mono"
          style={{ fontSize: 13, color: "var(--amber)", letterSpacing: "0.1em" }}
        >
          ERROR · 404
        </div>
        <h2 style={{ fontSize: 22, margin: "12px 0 6px" }}>路由不存在</h2>
        <div className="faint mono" style={{ fontSize: 13, marginBottom: 20 }}>
          这条路径不在路由表里。
        </div>
        <Link to="/analyze" className="btn">
          <Icon name="chevron-right" size={14} /> 回到需求分析终端
        </Link>
      </div>
    </div>
  );
}
