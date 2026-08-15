import { useEffect } from "react";

/**
 * ConfirmDialog —— 应用内确认弹窗（替换 window.confirm/alert）。
 * 危险操作确认态：主按钮红（danger）；Esc/点遮罩 = 取消。
 * 全手写无依赖（ADR-008）。
 */
export default function ConfirmDialog({
  open,
  title,
  body,
  confirmText = "确认",
  cancelText = "取消",
  danger = true,
  busy = false,
  onConfirm,
  onCancel,
}: {
  open: boolean;
  title: string;
  body?: string;
  confirmText?: string;
  cancelText?: string;
  danger?: boolean;
  busy?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}) {
  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onCancel();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, onCancel]);

  if (!open) return null;
  return (
    <div className="cd-mask" onClick={onCancel} role="presentation">
      <div
        className="cd-box"
        role="alertdialog"
        aria-modal="true"
        aria-label={title}
        onClick={(e) => e.stopPropagation()}
      >
        <div className="cd-title">{title}</div>
        {body && <div className="cd-body">{body}</div>}
        <div className="cd-btns">
          <button className="btn ghost sm" disabled={busy} onClick={onCancel}>
            {cancelText}
          </button>
          <button
            className={`btn sm ${danger ? "danger" : ""}`}
            disabled={busy}
            onClick={onConfirm}
          >
            {busy ? "处理中…" : confirmText}
          </button>
        </div>
      </div>
    </div>
  );
}
