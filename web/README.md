# web/ — 用户端（Vue3 + Element Plus）

用户端源码与构建产物放此目录，构建后 `dist/` 经 `go:embed` 打进后端单二进制（同源部署）。

- 开发：在此目录初始化 Vue3 + Element Plus 工程（T9 阶段落地）。
- 构建：`npm run build` 生成 `dist/`（`dist/` 保留供 go:embed 打包）。
- 本目录当前为空占位，待 T9 任务实现。
