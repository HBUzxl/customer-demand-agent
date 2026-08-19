// 密码策略（与后端 internal/identity.ValidatePassword 对齐，2026-08-19）：
// 8~16 字符，且至少包含大写字母、小写字母、数字、符号中的三种。
export const PASSWORD_MIN = 8;
export const PASSWORD_MAX = 16;

// passwordError 返回违规提示；合规返回 null。注册与改密共用。
export function passwordError(pw: string): string | null {
  if (pw.length < PASSWORD_MIN || pw.length > PASSWORD_MAX) {
    return `密码长度需为 ${PASSWORD_MIN}~${PASSWORD_MAX} 字符`;
  }
  let classes = 0;
  if (/[a-z]/.test(pw)) classes++;
  if (/[A-Z]/.test(pw)) classes++;
  if (/[0-9]/.test(pw)) classes++;
  if (/[^a-zA-Z0-9]/.test(pw)) classes++;
  if (classes < 3) return "密码需包含大写字母、小写字母、数字、符号中的至少三种";
  return null;
}
