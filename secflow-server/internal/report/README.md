# Security Report Generator

安全报告生成模块，支持导出为 Markdown、HTML 和 PDF 格式，集成 AI 智能摘要。

## 技术栈

- **Markdown 解析**: [goldmark](github.com/yuin/goldmark) - 纯 Go 实现的 Markdown 解析器
- **AI 摘要**: **Minimax API** (abab6.5s / abab6.5g 模型)
- **GFM 扩展**: 支持 GitHub Flavored Markdown（表格、任务列表等）

## 功能特性

- ✅ **Markdown 导出** - 纯文本格式，适合存档
- ✅ **HTML 导出** - 精美样式，中文友好
- ✅ **多数据源选择** - NVD、CNVD、CNNVD、先知、奇安信、安全客
- ✅ **AI 智能摘要** - Minimax API 支持

## 报告结构

```
安全周报
├── 🤖 AI 智能摘要（可选）
│   ├── 概况
│   ├── 重点关注
│   ├── 安全建议
│   └── 趋势分析
├── 0x00 本周安全概况
│   ├── 漏洞统计汇总
│   └── 来源分布
├── 漏洞列表 (TOP 10)
├── 0x01 重要漏洞列表
├── 本周安全事件
└── 数据来源
```

## 使用方法

### 基本使用

```go
import "github.com/secflow/server/internal/report"

// 创建生成器（无 AI）
gen := report.NewGenerator(vulnRepo, articleRepo)

// 创建生成器（带 Minimax AI）
gen := report.NewGeneratorWithAI(
    vulnRepo, 
    articleRepo,
    "your-minimax-api-key",      // API Key
    "your-group-id",             // Group ID
    report.AIModelGPT4,          // 模型类型
)
```

### 生成报告

```go
// 报告配置
config := &report.ReportConfig{
    Title:      "第45期安全周报",
    ReportType: report.ReportTypeWeekly,
    DateFrom:   time.Now().AddDate(0, 0, -7),
    DateTo:     time.Now(),
    Sources:    []report.DataSource{report.DataSourceNVD, report.DataSourceCNVD},
    AIModel:   report.AIModelGPT4, // 启用 AI 摘要
}

// 生成报告
data, err := gen.GenerateReport(ctx, config)

// 导出
md, filename, _ := gen.ExportToFile(data, report.FormatMarkdown)
html, _, _ := gen.ExportToFile(data, report.FormatHTML)
```

## AI 摘要配置

### Minimax API

```go
// 启用 Minimax AI 摘要
gen := report.NewGeneratorWithAI(
    vulnRepo,
    articleRepo,
    os.Getenv("MINIMAX_API_KEY"),   // API Key
    os.Getenv("MINIMAX_GROUP_ID"),   // Group ID
    report.AIModelGPT4,
)
```

### 环境变量

```bash
# Minimax API 配置
export MINIMAX_API_KEY="your-api-key"
export MINIMAX_GROUP_ID="your-group-id"
```

## 数据来源

| 来源 | ID | 说明 |
|------|-----|------|
| NVD | `nvd` | 美国国家漏洞数据库 |
| CNVD | `cnvd` | 中国国家漏洞库 |
| CNNVD | `cnnvd` | 中国国家信息安全漏洞库 |
| 先知社区 | `xianzhi` | 阿里云先知社区 |
| 奇安信 | `qianxin` | 奇安信威胁情报 |
| 安全客 | `anquanke` | 安全客资讯 |

## 定时任务

```go
scheduler := report.NewScheduler(gen, reportRepo, &report.SchedulerConfig{
    WeeklyEnabled:  true,
    WeeklyCron:     "0 9 * * 1",  // 每周一 9:00
    MonthlyEnabled: true,
    MonthlyCron:    "0 9 1 * *",   // 每月 1 日 9:00
    DefaultSources: []report.DataSource{report.DataSourceNVD, report.DataSourceCNVD},
    AIModel:       report.AIModelGPT4,
})

scheduler.Start()
```

## API Handler 集成

```go
// main.go
reportGen := report.NewGeneratorWithAI(
    vulnRepo, 
    articleRepo,
    cfg.Minimax.APIKey,
    cfg.Minimax.GroupID,
    report.AIModelGPT4,
)
reportH := handler.NewReportHandler(reportGen, vulnRepo, reportRepo)
```

## 后续优化

- [ ] PDF 导出（前端 html2pdf.js）
- [ ] Claude/Gemini API 支持
- [ ] 定时任务 Web UI 配置
- [ ] 报告模板自定义
