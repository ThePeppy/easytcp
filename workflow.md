# 开发维护工作流

本文档说明:改动代码之后,如何完整地提交、推送并发布一个新版本。

## 基本信息

| 项目     | 值                                                       |
| -------- | -------------------------------------------------------- |
| 模块路径 | `github.com/ThePeppy/easytcp`(`go.mod` 第一行,不要改回上游) |
| 自己的仓库 | `origin` → `git@github.com:ThePeppy/easytcp.git`         |
| 上游仓库 | `upstream` → `git@github.com:DarthPestilane/easytcp.git`  |
| 主分支   | `master`                                                 |
| 版本规则 | 语义化版本 `vMAJOR.MINOR.PATCH`,目前在 `v0.x` 阶段        |

## 发布一个新版本

按顺序执行,每一步成功再做下一步。

### 1. 检查改动

```bash
git status --short
git diff
```

确认只有预期的文件被改动。

### 2. 格式化、编译、静态检查、测试

```bash
gofmt -l .          # 没有输出才算通过
go build ./...
go vet ./...
make test           # 带 -race 和覆盖率
```

任何一步失败,先修复,不要继续。

如果改了依赖,整理 `go.mod` 和 `go.sum`:

```bash
make tidy           # 即 go mod tidy -v
```

如果改了 `Packer` 等接口,需要重新生成 mock:

```bash
make gen
```

### 3. 提交

只添加需要的文件,不要直接用 `git add -A`,避免把无关文件提交进去。

```bash
git add <文件...>
git commit -m "<type>: <简短说明>"
```

提交信息的 `type` 可选:

| type       | 用途                 |
| ---------- | -------------------- |
| `feat`     | 新功能               |
| `fix`      | 修复问题             |
| `refactor` | 重构,不改变行为     |
| `docs`     | 文档                 |
| `test`     | 测试                 |
| `chore`    | 构建、依赖等杂项     |

### 4. 推送代码

```bash
git push origin master
```

### 5. 确定新版本号

查看当前最新版本:

```bash
git ls-remote --tags origin | rg -v '\^\{\}'
```

按改动类型递增:

| 改动类型                     | 递增   | 例子                |
| ---------------------------- | ------ | ------------------- |
| 修复问题、不改变接口         | PATCH  | `v0.5.1` → `v0.5.2` |
| 新增功能,向后兼容           | MINOR  | `v0.5.1` → `v0.6.0` |
| 不兼容的接口改动             | MINOR  | `v0.x` 阶段同样升 MINOR,并在说明里写明 |

注意:

- 版本号只能往上加,已经发布的 tag 不要删除、不要移动。
- 不要推送上游的旧 tag(`v0.3.0`、`v0.3.2`、`v0.4.0`),它们对应的是旧的模块路径。
- 进入 `v2` 及以上时,模块路径必须加 `/v2` 后缀,这是 Go 的规则。

### 6. 打 tag 并推送

tag 必须打在已经推送的最新提交上。

```bash
git tag -a vX.Y.Z -m "vX.Y.Z: <一句话说明>"
git push origin vX.Y.Z
```

只推送这一个 tag,不要用 `git push --tags`,否则会把本地的上游旧 tag 一起推上去。

### 7. 验证新版本能用

在一个空的临时目录里测试:

```bash
mkdir /tmp/verify && cd /tmp/verify
go mod init verify
go get -u github.com/ThePeppy/easytcp@vX.Y.Z
```

再写一个 `main.go` import 并使用这个库,然后执行:

```bash
go mod tidy
go run .
go vet ./...
```

`go get -u` 会把所有依赖升到最新,所以这一步能提前发现"依赖升级后编译失败"的问题。

## 常见问题

### 不带版本号的 `go get` 拿到的还是旧版本

默认代理(如 `goproxy.cn`、`proxy.golang.org`)会缓存版本列表,新 tag 推送后可能有一段时间看不到。

- 先使用带版本号的命令:`go get github.com/ThePeppy/easytcp@vX.Y.Z`
- 或者临时绕过代理:`GOPROXY=direct go get github.com/ThePeppy/easytcp@latest`
- 过一段时间缓存过期,不带版本号的命令也会拿到新版本。

### `module declares its path as ... but was required as ...`

说明拿到的提交里 `go.mod` 的模块路径不是 `github.com/ThePeppy/easytcp`。

- 检查 `head -1 go.mod`。
- 合并上游更新后尤其要检查,上游的改动可能把路径带回 `DarthPestilane`。
- 已经发布的 tag 无法修复,只能修正后发布新的版本。

### `go install` 报 `is not a main package`

这是库,根目录没有 `main` 包,用 `go get` 引入,不要用 `go install`。要安装可执行程序,指向具体的 `main` 包,例如:

```bash
go install github.com/ThePeppy/easytcp/internal/examples/tcp/simple/server@vX.Y.Z
```

### 推送 tag 后 GitHub Actions 的 release 工作流

`.github/workflows/release.yml` 会在任何 tag 推送后触发 GoReleaser,使用 `release` 环境(需要手动确认),并依赖仓库密钥 `GORELEASER_TOKEN`。

- 没有配置该密钥时,这个工作流会失败。这不影响 `go get`,因为 Go 直接从 tag 取代码。
- 只想给 Go 用户发布时,可以忽略它的失败;需要 GitHub Release 页面的发布包时,再配置密钥。

## 合并上游更新

```bash
git fetch upstream
git merge upstream/master
```

合并后必须执行:

1. 检查 `go.mod` 第一行和所有 import,模块路径应是 `github.com/ThePeppy/easytcp`:
   ```bash
   rg "DarthPestilane/easytcp" --glob '*.go' --glob 'go.mod'
   ```
   有输出就批量替换回来。
2. 重新执行"发布一个新版本"的第 2 步到第 7 步。

## 发布前自检清单

- [ ] `gofmt -l .` 没有输出
- [ ] `go build ./...` 和 `go vet ./...` 通过
- [ ] `make test` 通过
- [ ] `go.mod` 第一行是 `module github.com/ThePeppy/easytcp`
- [ ] 代码已推送到 `origin/master`
- [ ] 新 tag 比已有版本都高,只推送了这一个 tag
- [ ] 在空目录中用 `@vX.Y.Z` 验证过 `go get` 和编译
