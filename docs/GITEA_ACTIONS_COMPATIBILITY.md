# Gitea Actions 兼容版本说明

更新时间：`2026-04-01`

## 背景

这份文档用于回答一个经常被误解的问题：**Gitea 到底“支持哪些 Action 版本”**。

先给结论：

1. **Gitea 官方没有提供“全网所有 GitHub Action 的完整支持版本总表”。**
2. **是否可用，不能只看 `uses: xxx@vN` 的主版本号，还要同时看 3 个维度：**
   - Action 是从哪里下载的
   - Action 的运行时是 `node16`、`node20` 还是 `node24`
   - Action 是否依赖 GitHub.com / GHES（GitHub Enterprise Server，GitHub 企业版服务器）专有后端能力
3. **对当前仓库最关键的特殊项是 Artifact（制品上传/下载）动作：`upload-artifact@v4+` / `download-artifact@v4+` 目前不能按 GHES 兼容路径在 Gitea 上稳定使用，应继续使用 `v3` 线。**

本文的“支持”定义为：**在 Gitea + act_runner 场景下，有官方证据或足够强的一手证据表明该版本具备可运行条件**。这不等同于 Gitea 官方对每个第三方 Action 做了逐个认证。

## 官方边界

### 1. Action 下载来源

Gitea 官方文档明确说明：

- 相对写法如 `uses: actions/checkout@v4`，默认会从 `https://github.com/actions/checkout` 下载。
- Gitea 支持绝对 URL，例如 `uses: https://your-gitea.example.com/actions/checkout@v4`。
- `DEFAULT_ACTIONS_URL` 在新版本 Gitea 中只支持 `github` 和 `self` 两个值。

这意味着：

- **中国网络环境下，是否能稳定拉到 Action 仓库，本身就是兼容性的一部分。**
- 即使某个 Action 版本“语义上兼容”，如果 runner 到 GitHub 网络不通，实际仍然不可用。

### 2. 语法兼容不是版本兼容

Gitea 官方还明确列出了与 GitHub Actions 的差异：

- `concurrency` 当前会被忽略。
- `pre/post steps`、`services steps` 没有与 GitHub 完全一致的 UI 展示。

这类差异不是某个 Action 的版本问题，但会影响你对“工作流是否真的等价迁移”的判断。

### 3. 真正决定能不能跑的 3 个技术因素

| 因素 | 说明 | 影响 |
|---|---|---|
| 下载来源 | 相对 `uses:` 默认走 GitHub；绝对 URL 可走你自己的 Gitea 或其他 Git 服务 | 中国网络、内网环境最容易先卡在这里 |
| `runs.using` 运行时 | Action 的 `action.yml` 里会写 `node16` / `node20` / `node24` | runner 太旧时，新主版本会直接跑不起来 |
| 后端能力 | 有些 Action 依赖 GitHub / GHES 的 Artifact、Cache、API 行为 | 这类版本即使能下载下来，也可能在运行时失败 |

## 判定规则

### 1. Node 运行时分层

| 档位 | 典型 `runs.using` | 兼容性判断 |
|---|---|---|
| 旧兼容档 | `node16` | 对较老 runner 最友好，通常是保守选择 |
| 主流档 | `node20` | 当前较稳妥，适合大多数已升级的自建 runner |
| 新特性档 | `node24` | 需要更新的 runner；新主版本越来越多切到这一档 |

### 2. Artifact 特殊规则

`actions/upload-artifact` 官方 README 明确写明：

- `upload-artifact@v4+` **当前不支持 GHES**
- GHES 需使用 `v3` 或 `v3-node20`

`download-artifact` 官方说明同样明确：

- `download-artifact@v4+` **当前不支持 GHES**
- GHES 需使用 `v3`

对 Gitea 来说，这条规则非常关键，因为你在实际运行中拿到的报错就是：

```text
GHESNotSupportedError: @actions/artifact v2.0.0+, upload-artifact@v4+ and download-artifact@v4+ are not currently supported on GHES
```

所以这不是“偶发失败”，而是**版本线选错了**。

### 3. 不要把 GitHub Runner 版本号直接等同成 Gitea 版本号

很多官方 Action README 会写：

- 需要 `Actions Runner v2.308.0+`
- 需要 `Actions Runner v2.327.1+`
- 需要 `Actions Runner v2.329.0+`

这里要注意：

- 这些要求是**官方 Action 对 GitHub Actions Runner 的要求**。
- Gitea 官方文档**没有提供一份“Gitea 版本 / act_runner 版本 与 GitHub Runner 版本的一一映射表”**。

因此本文会把这些要求当成**风险提示**，而不是直接写成“Gitea 1.xx 一定支持/不支持”。

## 当前仓库 Action 兼容矩阵

下表只梳理**当前仓库已经实际使用到的 Action**，这是目前最有操作价值的一部分。

| Action | 当前仓库引用 | 已核实主版本运行时 | Gitea 建议 | 结论 |
|---|---|---|---|---|
| `actions/checkout` | `@v6` | `v3=node16`，`v4=node20`，`v6=node24` | 新 runner 可用 `v6`；保守建议 `v4` | **支持分层使用** |
| `actions/setup-go` | `@v6` | `v4=node16`，`v5=node20`，`v6=node24` | 新 runner 用 `v6`；稳妥建议 `v5` | **支持分层使用** |
| `actions/setup-node` | `@v6` | `v4=node20`，`v5=node24`，`v6=node24` | 若 runner 未确认支持 `node24`，建议退到 `v4` | **支持分层使用** |
| `actions/upload-artifact` | `@v3` | `v3=node16`，`v3-node20=node20`，`v4+=新 Artifact 后端` | **固定 `v3` / `v3-node20`** | **`v4+` 不建议在 Gitea 上使用** |
| `actions/download-artifact` | `@v3` | `v3=node16`，`v3-node20=node20`，`v4+=新 Artifact 后端` | **固定 `v3`** | **`v4+` 不建议在 Gitea 上使用** |
| `docker/setup-buildx-action` | `@v3` | `v2=node16`，`v3=node20` | runner 支持 `node20` 时可用 `v3`；保守用 `v2` | **支持分层使用** |
| `docker/build-push-action` | `@v6` | `v4=node16`，`v5=node20`，`v6=node20` | 当前 `v6` 可继续；老 runner 可回退 `v4` | **支持分层使用** |
| `docker/login-action` | `@v3` | `v2=node16`，`v3=node20` | runner 支持 `node20` 时可用 `v3`；保守用 `v2` | **支持分层使用** |
| `docker/metadata-action` | `@v5` | `v4=node16`，`v5=node20` | runner 支持 `node20` 时可用 `v5`；保守用 `v4` | **支持分层使用** |

## 各 Action 说明

### `actions/checkout`

- `v4` 的 `action.yml` 明确是 `runs.using: node20`
- `v6` 的 `action.yml` 明确是 `runs.using: node24`
- 官方 Marketplace 页面说明：`v5` 升级到 `node24`，需要较新的 runner
- 官方还说明：`v6` 在 **Docker container action 中执行带认证的 Git 命令** 时，需要更高的 runner 版本

**建议：**

- 如果你的 runner 是否支持 `node24` 还没有确认，先 pin 到 `v4`
- 如果已经确认新 runner 正常，`v6` 可以保留
- 无论选哪个版本，**中国网络环境都建议改成绝对 URL 或镜像到自建 Gitea**

### `actions/setup-go`

- `v4=node16`
- `v5=node20`
- `v6=node24`

同时这个 Action 的 `action.yml` 还暴露出一个对 Gitea 很重要的细节：

- `token` 默认值只有在 `github.server_url == 'https://github.com'` 时才自动带入
- 在 Gitea 场景下，这个默认 token 往往是空字符串

这意味着：

- 如果它需要去 GitHub 拉 Go toolchain 元数据，**可能会遇到匿名限流或访问失败**
- 中国网络建议优先配 `go-download-base-url` 或环境变量 `GO_DOWNLOAD_BASE_URL` 指向镜像

### `actions/setup-node`

- `v4=node20`
- `v5=node24`
- `v6=node24`

和 `setup-go` 一样，它在 Gitea 上也有两个高频坑：

1. 默认 `token` 在非 GitHub.com 环境下通常为空
2. 拉 Node 分发包时，默认外网源对中国网络不友好

**建议：**

- 未确认 `node24` 兼容前，优先 pin `v4`
- 配合 `mirror` / `mirror-token` 使用镜像源

### `actions/upload-artifact` / `actions/download-artifact`

这是 Gitea 上最需要单独记住的一组：

- `upload-artifact@v4+` 官方明确不支持 GHES
- `download-artifact@v4+` 官方明确不支持 GHES
- 你当前仓库已经实际复现了这个错误

**建议：**

- `upload-artifact` 固定到 `v3`，如果你明确需要 `node20`，可选 `v3-node20`
- `download-artifact` 固定到 `v3`
- 在没有新的官方兼容声明之前，不要升级到 `v4+`

### Docker 官方 Actions

当前仓库使用的是：

- `docker/setup-buildx-action@v3`
- `docker/build-push-action@v6`
- `docker/login-action@v3`
- `docker/metadata-action@v5`

从对应主版本的 `action.yml` 看：

- `setup-buildx-action@v3` 是 `node20`
- `build-push-action@v6` 是 `node20`
- `login-action@v3` 是 `node20`
- `metadata-action@v5` 是 `node20`

保守回退线分别是：

- `setup-buildx-action@v2=node16`
- `build-push-action@v4=node16`
- `login-action@v2=node16`
- `metadata-action@v4=node16`

**建议：**

- 如果你的 runner 已稳定支持 `node20`，当前 Docker 这组主版本可以继续用
- 如果你遇到的是运行时兼容问题，而不是 Docker 本身问题，可以统一回退一档到 `node16` 线

## 推荐版本策略

### 策略 A：保守兼容档

适用于：

- 自建 Gitea 时间不长
- runner 是否完整支持 `node20/node24` 未确认
- 优先要“先跑起来”

建议 pin：

```yaml
uses: actions/checkout@v4
uses: actions/setup-go@v5
uses: actions/setup-node@v4
uses: actions/upload-artifact@v3
uses: actions/download-artifact@v3
uses: docker/setup-buildx-action@v2
uses: docker/build-push-action@v4
uses: docker/login-action@v2
uses: docker/metadata-action@v4
```

### 策略 B：当前仓库建议档

适用于：

- runner 已经能稳定跑 `node20`
- 你希望尽量保持主版本较新
- 你接受 `checkout/setup-go/setup-node` 仍需额外确认 `node24`

建议 pin：

```yaml
uses: actions/checkout@v6
uses: actions/setup-go@v6
uses: actions/setup-node@v6
uses: actions/upload-artifact@v3
uses: actions/download-artifact@v3
uses: docker/setup-buildx-action@v3
uses: docker/build-push-action@v6
uses: docker/login-action@v3
uses: docker/metadata-action@v5
```

补充说明：

- 这组里真正有**已知硬限制**的是 Artifact，必须停在 `v3`
- `checkout/setup-go/setup-node` 的风险点不是 Gitea 语法，而是 `node24` 运行时与外网依赖

### 策略 C：中国网络稳定运行档

适用于：

- 你的主要问题不是 runner 版本，而是 GitHub 网络
- 你要先解决 “TLS handshake timeout / clone timeout”

除了 pin 版本，必须再做下面至少一项：

1. 给 runner 所在主机配置代理
2. 把常用 Action 镜像到自建 Gitea
3. 把 workflow 中的 `uses:` 改成绝对 URL

例如：

```yaml
uses: https://your-gitea.example.com/actions/checkout@v4
uses: https://your-gitea.example.com/actions/setup-go@v5
uses: https://your-gitea.example.com/actions/upload-artifact@v3
```

## 当前仓库最终建议

结合当前仓库已经暴露出的两个真实问题：

1. `actions/checkout` 拉 GitHub 超时
2. `upload-artifact@v4+` / `download-artifact@v4+` 在 Gitea 上报 GHES 不支持

当前最务实的建议是：

| 类别 | 建议 |
|---|---|
| Artifact 类 | **固定 `v3`**，不要再升 `v4+` |
| 拉代码/装环境类 | 若 `node24` 未验证稳定，先退到 `checkout@v4`、`setup-go@v5`、`setup-node@v4` |
| Docker 类 | runner 能跑 `node20` 就保留当前版本，否则分别回退到 `v2/v4/v2/v4` |
| 网络层 | **优先解决 GitHub 连通性**，否则版本对了也会卡在下载阶段 |
| 来源配置 | **优先绝对 URL 或镜像到自建 Gitea**，不要继续依赖默认相对 `uses:` |

## 证据来源

### Gitea 官方文档

- Gitea Actions 总览：<https://docs.gitea.com/usage/actions/>
- Gitea Actions FAQ：<https://docs.gitea.com/1.23/usage/actions/faq>
- Gitea 与 GitHub Actions 差异：<https://docs.gitea.com/1.23/usage/actions/comparison>
- Gitea Configuration Cheat Sheet：<https://docs.gitea.com/administration/config-cheat-sheet>

### 官方 Action 仓库 / 官方页面

- `actions/checkout`：<https://github.com/actions/checkout>
- `actions/setup-go`：<https://github.com/actions/setup-go>
- `actions/setup-node`：<https://github.com/actions/setup-node>
- `actions/upload-artifact`：<https://github.com/actions/upload-artifact>
- `docker/setup-buildx-action`：<https://github.com/docker/setup-buildx-action>
- `docker/build-push-action`：<https://github.com/docker/build-push-action>
- `docker/login-action`：<https://github.com/docker/login-action>
- `docker/metadata-action`：<https://github.com/docker/metadata-action>

### 本文核对时用到的关键一手证据

- `actions/checkout@v4/v6`、`actions/setup-go@v4/v5/v6`、`actions/setup-node@v4/v5/v6`、`actions/upload-artifact@v3/v3-node20`、`actions/download-artifact@v3/v3-node20` 的 `action.yml`
- `docker/setup-buildx-action@v2/v3`、`docker/build-push-action@v4/v6`、`docker/login-action@v2/v3`、`docker/metadata-action@v4/v5` 的 `action.yml`
- `actions/upload-artifact` 官方 README 中关于 GHES 不支持 `v4+` 的说明
- Gitea 官方关于 `DEFAULT_ACTIONS_URL`、绝对 URL、相对 `uses:` 默认指向 GitHub 的说明

## 维护建议

以后每次升级 Action 主版本时，按这个顺序检查：

1. 先看该版本 tag 的 `action.yml`，确认 `runs.using`
2. 再看 README / Release Notes 是否要求更高的 runner
3. 再看它是否引入了新的 GitHub 专有后端依赖
4. 最后再决定要不要在 Gitea 上升级

如果只看 `@vN` 主版本号，不看 `action.yml`，在 Gitea 上很容易出现“版本看起来是小升级，实际却把运行时从 `node20` 切到 `node24`”这种隐性兼容问题。
