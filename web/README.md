# Web Frontend

前端基于 React 19、TypeScript、Rsbuild 和 Tailwind CSS 4，默认通过 `VITE_API_URL` 访问后端 API；未配置时回落到 `http://localhost:8080/api/v1`。

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
