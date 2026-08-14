// ESLint flat config — React + TypeScript + Vite
// 标准 Vite 脚手架规则集 + typescript-eslint
import js from "@eslint/js";
import globals from "globals";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";
import tseslint from "typescript-eslint";

export default tseslint.config(
  { ignores: ["dist", "node_modules"] },
  {
    extends: [js.configs.recommended, ...tseslint.configs.recommended],
    files: ["**/*.{ts,tsx}"],
    languageOptions: {
      ecmaVersion: 2022,
      globals: globals.browser,
    },
    plugins: {
      "react-hooks": reactHooks,
      "react-refresh": reactRefresh,
    },
    rules: {
      ...reactHooks.configs.recommended.rules,
      "react-refresh/only-export-components": ["warn", { allowConstantExport: true }],
      "@typescript-eslint/no-unused-vars": [
        "warn",
        { argsIgnorePattern: "^_", varsIgnorePattern: "^_" },
      ],
      // react-hooks v7 的 set-state-in-effect 对「异步 fetch→.then setState」的标准数据加载
      // 模式误报（本仓库 7 处 useEffect(() => load(), [deps]) 均为此模式，React 文档亦认可）。
      // 放开此规则；若日后要做全量 effect 重构可重新启用。
      "react-hooks/set-state-in-effect": "off",
    },
  },
);
