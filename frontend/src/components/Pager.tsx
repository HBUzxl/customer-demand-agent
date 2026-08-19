/**
 * Pager —— 通用客户端分页控件（列表 slice 后按页展示）。
 * 单页时自动隐藏。页码从 0 开始。
 */
export default function Pager({
  page,
  totalPages,
  onChange,
}: {
  page: number;
  totalPages: number;
  onChange: (page: number) => void;
}) {
  if (totalPages <= 1) return null;
  return (
    <div className="pager">
      <button className="btn ghost sm" disabled={page <= 0} onClick={() => onChange(page - 1)}>
        ‹ 上一页
      </button>
      <span className="pager-info">
        第 {page + 1} / {totalPages} 页
      </span>
      <button
        className="btn ghost sm"
        disabled={page >= totalPages - 1}
        onClick={() => onChange(page + 1)}
      >
        下一页 ›
      </button>
    </div>
  );
}
