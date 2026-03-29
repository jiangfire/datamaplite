# Web Frontend

前端基于 React 19、TypeScript、Rsbuild 和 Tailwind CSS 4，默认通过同源路径 `/api/v1` 访问后端 API；开发环境下 `pnpm dev` 会把 `/api` 和 `/mcp` 代理到 `http://localhost:8080`。如有需要，也可以通过 `VITE_API_URL` 覆盖。

## 常用命令

```bash
pnpm install
pnpm dev
pnpm lint
pnpm test
pnpm build
```

## 开发说明

- `pnpm dev`：启动前端开发服务，实际端口以终端输出为准。
- `pnpm lint`：运行 ESLint。
- `pnpm test`：运行 Vitest 单元测试。
- `pnpm build`：构建生产包到 `web/dist/`。

## 目录结构

- `src/pages`：页面级路由
- `src/components`：复用组件
- `src/hooks`：数据获取与状态封装
- `src/services`：API 调用层
- `src/types`：共享类型定义
