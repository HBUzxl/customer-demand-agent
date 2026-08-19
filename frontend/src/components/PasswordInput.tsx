// 密码输入框 + 明文切换按钮（右侧眼睛图标）。点按在 password/text 间切换，
// 用于登录 / 注册 / 改密，避免输错长密码需反复重输。
import { useState } from "react";
import type { InputHTMLAttributes } from "react";

type Props = Omit<InputHTMLAttributes<HTMLInputElement>, "type" | "value" | "onChange"> & {
  value: string;
  onChange: (e: React.ChangeEvent<HTMLInputElement>) => void;
};

export default function PasswordInput({ value, onChange, className, ...rest }: Props) {
  const [show, setShow] = useState(false);
  return (
    <span className="pw-input">
      <input
        type={show ? "text" : "password"}
        className={className}
        value={value}
        onChange={onChange}
        {...rest}
      />
      <button
        type="button"
        className={`pw-toggle${show ? " on" : ""}`}
        onClick={() => setShow((s) => !s)}
        aria-label={show ? "隐藏密码" : "显示密码"}
        title={show ? "隐藏密码" : "显示密码"}
        aria-pressed={show}
      >
        <svg
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.7"
          strokeLinecap="round"
          strokeLinejoin="round"
          aria-hidden="true"
        >
          {show ? (
            // 显示态：睁眼
            <>
              <path d="M1 12s4-7 11-7 11 7 11 7-4 7-11 7S1 12 1 12z" />
              <circle cx="12" cy="12" r="3" />
            </>
          ) : (
            // 隐藏态：闭眼（眼睛 + 斜杠）
            <>
              <path d="M1 12s4-7 11-7 11 7 11 7-4 7-11 7S1 12 1 12z" />
              <circle cx="12" cy="12" r="3" />
              <line x1="4" y1="20" x2="20" y2="4" />
            </>
          )}
        </svg>
      </button>
    </span>
  );
}
