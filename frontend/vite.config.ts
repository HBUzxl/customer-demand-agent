import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    // 开发时把 /api 代理到后端 Go 服务
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        // 关闭 changeOrigin：保留浏览器原始 Host（devbox 域名），
        // 否则后端 CSRF 同源校验（Origin vs r.Host）因 Host 被改写为 localhost:8080 而 403「跨站请求被拒绝」。
        changeOrigin: false,
      },
    },
  },
  build: {
    outDir: "dist",
  },
});
