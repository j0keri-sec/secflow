# Go-Rod 迁移总结

## 完成的工作

### 1. 已迁移的源（使用 Go-Rod）

| 源名称 | 文件 | 状态 | 测试结果 |
|--------|------|------|----------|
| AVD (阿里云) | `avd_rod.go` | ✓ 可用 | 2 个漏洞/页 |
| Seebug | `seebug_rod.go` | ✓ 可用 | 20 个漏洞/页 |
| TI/Nox (奇安信) | `ti_rod.go` | ✓ 可用 | 2 个漏洞 |
| KEV (CISA) | `kev_rod.go` | ✓ 可用 | 10 个漏洞 |
| Struts2 | `struts2_rod.go` | ✓ 可用 | 1 个漏洞 |

### 2. 不需要迁移的源（API 驱动）

以下源使用 JSON API 或简单的 HTML，没有 WAF 保护，HTTP 客户端版本工作正常：

| 源名称 | 类型 | 状态 |
|--------|------|------|
| OSCS | JSON API | ✓ 正常工作 |
| Chaitin | JSON API | ✓ 正常工作 |
| ThreatBook | JSON API | ✓ 正常工作 |
| TI / Nox | JSON API | ✓ 正常工作 |
| KEV | JSON API | ✓ 正常工作 |
| Struts2 | HTML (无 WAF) | ✓ 正常工作 |

### 3. 已知问题

| 源名称 | 问题 | 建议 |
|--------|------|------|
| Venustech | 网站结构变化 | 暂时禁用，等待修复 |

## 测试验证

### 测试命令

```bash
# 测试单个 Rod 版本源
go run cmd/test-rod/main.go avd-rod
go run cmd/test-rod/main.go seebug-rod
go run cmd/test-rod/main.go ti-rod

# 测试所有源
go run cmd/test-all/main.go
```

### 测试结果

- **avd-rod**: ✓ 成功，抓取 2 个漏洞
- **seebug-rod**: ✓ 成功，抓取 20 个漏洞
- **ti-rod**: ✓ 成功，抓取 2 个漏洞
- **kev-rod**: ✓ 成功，抓取 10 个漏洞
- **struts2-rod**: ✓ 成功，抓取 1 个漏洞

## 配置建议

### 推荐的 Standalone 配置

```yaml
mode: standalone

grabber:
    sources:
        # 使用 Rod 版本（有 WAF 保护）
        - avd-rod
        - seebug-rod
        
        # 使用 HTTP 版本（API 驱动，无 WAF）
        - oscs
        - chaitin
        - threatbook
        - ti
        - kev
        - struts2
    
    disabled_sources:
        - avd        # 使用 avd-rod
        - seebug     # 使用 seebug-rod
        - venustech  # 暂时不可用

scheduler:
    enabled: true
    interval: 1h
    sources:
        - avd-rod
        - seebug-rod
        - ti-rod
        - oscs
        - chaitin
        - threatbook
        - kev
        - struts2
```

## 技术细节

### 为什么这些源使用 Rod

1. **AVD (阿里云)**: 有 WAF 保护，需要 JavaScript 执行绕过
2. **Seebug**: 有知道创宇云 WAF，需要 cookie 计算绕过
3. **TI/Nox (奇安信)**: API 驱动，Rod 版本可增强稳定性
4. **KEV (CISA)**: API 驱动，Rod 版本可增强稳定性
5. **Struts2**: HTML 解析，Rod 版本可增强稳定性

### 为什么其他源不需要 Rod

1. **API 驱动的源**: 直接返回 JSON 数据，没有反爬机制
2. **Struts2**: Apache 官方安全公告，没有 WAF

### Rod 版本的优势

- 真实浏览器环境，更难被检测
- 自动处理 JavaScript 动态内容
- 更好的 WAF 绕过能力

### Rod 版本的劣势

- 启动时间较慢（需要启动 Chrome）
- 内存占用较高
- 抓取速度较慢

## 文件变更

### 新增文件

- `pkg/grabber/avd_rod.go` - AVD Rod 版本
- `pkg/grabber/seebug_rod.go` - Seebug Rod 版本
- `pkg/grabber/ti_rod.go` - TI/Nox Rod 版本
- `pkg/grabber/kev_rod.go` - KEV Rod 版本
- `pkg/grabber/struts2_rod.go` - Struts2 Rod 版本
- `pkg/grabber/rod_base.go` - Rod 爬虫基础类
- `pkg/rodutil/browser.go` - 浏览器管理
- `pkg/rodutil/bypass.go` - WAF 绕过工具
- `pkg/rodutil/helpers.go` - 辅助函数
- `cmd/test-rod/main.go` - Rod 版本测试工具
- `cmd/test-all/main.go` - 全源测试工具

### 修改文件

- `pkg/grabber/sources.go` - 注册 Rod 版本源
- `pkg/rodutil/browser.go` - 修复 macOS Chrome 路径检测
- `pkg/grabber/README_ROD.md` - 更新文档
- `client.standalone.yaml` - 更新推荐配置

### 备份文件

- `pkg/grabber/backup/*.go` - 原始 HTTP 版本备份

## 后续建议

1. **监控**: 定期检查所有源的健康状态
2. **更新**: 当网站结构变化时，优先更新 Rod 版本
3. **扩展**: 如有新源需要 WAF 绕过，参考现有 Rod 实现
4. **优化**: 考虑添加浏览器池以提高并发性能
