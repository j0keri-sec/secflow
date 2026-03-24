# 爬虫源配置参考

本文档列出所有支持的漏洞情报源和安全资讯源。

## 漏洞情报源

### 国际源

#### 1. NVD (National Vulnerability Database)

**源标识**: `nvd`

**描述**: 美国国家漏洞数据库，漏洞 CVE 信息的权威来源。

**API 端点**: `https://services.nvd.nist.gov/rest/json/cves/2.0`

**配置参数**:

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `page_limit` | int | 10 | 爬取页数 |
| `year` | int | - | 筛选特定年份 |
| `severity` | string | - | cvssV3_severity: HIGH/CRITICAL |

**示例**:

```bash
curl -X POST http://localhost:8080/api/v1/tasks/vuln-crawl \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"source":"nvd","page_limit":10,"priority":50}'
```

**特点**:
- ✅ 数据权威，更新及时
- ✅ API 稳定，支持批量查询
- ⚠️ 部分数据需要 rate limiting

---

#### 2. CVE.org

**源标识**: `cve`

**描述**: MITRE CVE 官方列表。

**爬取方式**: HTTP API

**配置参数**:

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `page_limit` | int | 10 | 爬取页数 |

---

#### 3. AVD (Alibaba Vuln Database)

**源标识**: `avd`

**描述**: 阿里云漏洞数据库。

**爬取方式**: go-rod 浏览器爬取

**配置参数**:

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `page_limit` | int | 10 | 爬取页数 |
| `category` | string | - | 漏洞分类 |

**难度等级**: ⭐⭐ (较易爬取)

---

#### 4. Exploit-DB

**源标识**: `exploitdb`

**描述**: Offensive Security 漏洞利用数据库。

**爬取方式**: HTTP 爬取

**特点**:
- ✅ 提供漏洞利用代码
- ✅ 数据更新快

---

#### 5. GitHub Advisory Database

**源标识**: `github`

**描述**: GitHub 漏洞咨询数据库。

**爬取方式**: GraphQL API

**认证**: 需要 GitHub Token

**配置**:

```yaml
github:
  token: "ghp_xxxx"  # GitHub Personal Access Token
  per_page: 100
```

---

#### 6. OSV (Open Source Vulnerabilities)

**源标识**: `osv`

**描述**: Google 开源漏洞数据库。

**API 端点**: `https://api.osv.dev/v1/query`

**特点**:
- ✅ 覆盖多种生态系统
- ✅ 支持包名直接查询

---

#### 7. Vulners

**源标识**: `vulners`

**描述**: 综合漏洞情报平台。

**API 端点**: `https://vulners.com/api/v3`

**特点**:
- ✅ 整合多个数据源
- ✅ 提供 Linux 漏洞情报

---

#### 8. CISA KEV

**源标识**: `cisa`

**描述**: CISA 已知被利用漏洞目录。

**URL**: `https://www.cisa.gov/known-exploited-vulnerabilities-catalog`

**特点**:
- ✅ 已知的在野利用漏洞
- ✅ 权威机构发布

---

### 国内源

#### 9. 奇安信威胁情报中心

**源标识**: `qianxin`

**描述**: 奇安信威胁情报中心。

**爬取方式**: go-rod 浏览器爬取

**子类型**:

| 类型标识 | 名称 | URL 模式 |
|----------|------|----------|
| `qianxin-vuln` | 漏洞情报 | `ti.qianxin.com/vulnerability` |
| `qianxin-weekly` | 安全周报 | `ti.qianxin.com/vulnerability/notice-detail/{id}?type=hot-week` |

**配置参数**:

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `article_type` | string | `hot-week` | 文章类型 |
| `page_limit` | int | 5 | 爬取页数 |

**难度等级**: ⭐⭐⭐⭐ (需要处理验证码)

---

#### 10. 绿盟科技

**源标识**: `nsfocus`

**描述**: 绿盟科技漏洞库。

**爬取方式**: go-rod 浏览器爬取

**URL**: `www.nsfocus.net/vulndb/`

**难度等级**: ⭐⭐⭐ (有基础防护)

---

#### 11. 启明星辰

**源标识**: `venustech`

**描述**: 启明星辰漏洞响应平台。

**爬取方式**: go-rod 浏览器爬取

**URL**: `www.venustech.com.cn`

**难度等级**: ⭐⭐⭐⭐ (有 WAF 拦截)

---

#### 12. 深信服

**源标识**: `sangfor`

**描述**: 深信服安全响应中心。

**爬取方式**: HTTP + go-rod

**URL**: `www.sangfor.com`

---

#### 13. 安恒信息

**源标识**: `dbappsecurity`

**描述**: 安恒信息威胁情报。

**爬取方式**: HTTP 爬取

---

#### 14. 天融信

**源标识**: `topsec`

**描述**: 天融信阿尔法实验室。

**爬取方式**: go-rod 浏览器爬取

**URL**: `topic.topsec.com.cn`

---

#### 15. 知道创宇

**源标识**: `knownsec`

**描述**: 知道创宇 Seebug 漏洞库。

**爬取方式**: HTTP API

**URL**: `www.seebug.org`

---

## 安全资讯源

### 1. 奇安信安全周报

**源标识**: `qianxin-weekly`

**描述**: 奇安信每周安全资讯汇总。

**爬取方式**: go-rod 浏览器爬取

**URL**: `ti.qianxin.com/vulnerability/notice-detail/{id}?type=hot-week`

**内容字段**:
- 标题
- 发布时间
- 摘要
- 完整正文
- 标签

---

### 2. 安全客

**源标识**: `anquanke`

**描述**: 网络安全资讯平台。

**爬取方式**: HTTP 爬取

**URL**: `www.anquanke.com`

---

### 3. 先知社区

**源标识**: `xianzhi`

**描述**: 阿里云先知社区文章。

**爬取方式**: HTTP 爬取

**URL**: `xz.aliyun.com`

---

### 4. FreeBuf

**源标识**: `freebuf`

**描述**: 网络安全门户。

**爬取方式**: HTTP 爬取

**URL**: `www.freebuf.com`

---

### 5. T00ls

**源标识**: `t00ls`

**描述**: 网络安全技术社区。

**爬取方式**: go-rod (需要登录)

**URL**: `www.t00ls.com`

---

## 爬虫源配置结构

### sources.yaml

```yaml
sources:
  # ============ 漏洞情报源 ============
  vuln:
    - id: nvd
      name: "NVD"
      type: api
      enabled: true
      priority: 100
      config:
        base_url: "https://services.nvd.nist.gov/rest/json/cves/2.0"
        api_key: ""  # 可选，用于提升 rate limit
        timeout: 30s
        rate_limit: 50  # requests per 30 seconds (with API key)

    - id: cve
      name: "CVE.org"
      type: api
      enabled: true
      priority: 90
      config:
        base_url: "https://www.cve.org/VulnerabilitiesCurrentSearch"

    - id: avd
      name: "阿里云漏洞库"
      type: rod
      enabled: true
      priority: 80
      config:
        url: "https://avd.aliyun.com"
        selectors:
          list: ".vuln-list .item"
          title: ".title"
          cve_id: ".cve-id"
          severity: ".severity"
          description: ".desc"

    - id: qianxin
      name: "奇安信威胁情报"
      type: rod
      enabled: true
      priority: 70
      config:
        url: "https://ti.qianxin.com/vulnerability"
        selectors:
          list: ".vuln-item"
        requires_stealth: true

    - id: venustech
      name: "启明星辰"
      type: rod
      enabled: false  # 有 WAF，建议暂时禁用
      priority: 60
      config:
        url: "https://www.venustech.com.cn"
        requires_proxy: true
        requires_stealth: true

    # ... 更多源配置

  # ============ 安全资讯源 ============
  article:
    - id: qianxin-weekly
      name: "奇安信安全周报"
      type: rod
      enabled: true
      priority: 80
      config:
        url: "https://ti.qianxin.com/vulnerability/notice-list?type=hot-week"
        selectors:
          list: ".notice-list .notice-item"
          title: ".title"
          summary: ".summary"
          content: ".article-content"
        article_types:
          - hot-week    # 热点周报
          - hot-day     # 热点日报
          - security    # 安全公告

    - id: anquanke
      name: "安全客"
      type: http
      enabled: true
      priority: 60
      config:
        url: "https://www.anquanke.com"
        selectors:
          list: ".article-list .item"
```

## 爬取器实现模板

### HTTP 爬取器模板

```go
// pkg/vulngrabber/http_example.go
package vulngrabber

import (
    "context"
    "fmt"
    "net/http"
    "time"

    "github.com/chromedp/chromedp"
)

// HTTPGrabber HTTP 爬取器示例
type HTTPGrabber struct {
    baseURL   string
    client    *http.Client
    selectors map[string]string
}

// NewHTTPGrabber 创建 HTTP 爬取器
func NewHTTPGrabber(baseURL string, selectors map[string]string) *HTTPGrabber {
    return &HTTPGrabber{
        baseURL: baseURL,
        client: &http.Client{
            Timeout: 30 * time.Second,
        },
        selectors: selectors,
    }
}

// Fetch 爬取漏洞列表
func (g *HTTPGrabber) Fetch(ctx context.Context, page int) ([]*Vuln, error) {
    url := fmt.Sprintf("%s?page=%d", g.baseURL, page)

    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, err
    }

    resp, err := g.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    // 解析 HTML 提取漏洞数据
    // ...

    return vulns, nil
}
```

### Rod 爬取器模板

```go
// pkg/vulngrabber/rod_example.go
package vulngrabber

import (
    "context"
    "fmt"

    "github.com/chromedp/chromedp"
)

// RodGrabber Rod 爬取器示例
type RodGrabber struct {
    baseURL   string
    selectors map[string]string
}

// NewRodGrabber 创建 Rod 爬取器
func NewRodGrabber(baseURL string, selectors map[string]string) *RodGrabber {
    return &RodGrabber{
        baseURL:   baseURL,
        selectors: selectors,
    }
}

// Fetch 爬取漏洞列表
func (g *RodGrabber) Fetch(ctx context.Context, page int) ([]*Vuln, error) {
    url := fmt.Sprintf("%s?page=%d", g.baseURL, page)

    var vulns []*Vuln

    err := chromedp.Run(ctx,
        chromedp.Navigate(url),
        chromedp.WaitVisible(g.selectors["list"]),

        // 等待列表加载
        chromedp.Sleep(2 * time.Second),

        // 提取数据
        chromedp.EvaluateAsDevTools(`
            const items = document.querySelectorAll('.vuln-item');
            Array.from(items).map(item => ({
                title: item.querySelector('.title')?.textContent,
                cve: item.querySelector('.cve-id')?.textContent,
                severity: item.querySelector('.severity')?.textContent,
            }))
        `, &vulns),
    )

    return vulns, err
}
```

## 源优先级配置

```yaml
# 优先级建议
sources:
  # 高优先级：权威数据源
  - id: nvd
    priority: 100
    reliability: 10  # 可靠性评分

  - id: cve
    priority: 95
    reliability: 10

  # 中优先级：常用数据源
  - id: avd
    priority: 80
    reliability: 8

  - id: qianxin
    priority: 75
    reliability: 8

  # 低优先级：补充数据源
  - id: venustech
    priority: 60
    reliability: 7
```

## 爬取策略配置

```yaml
crawl_strategy:
  # 失败处理
  on_failure:
    retry_times: 3
    retry_delay: 10s
    backoff: exponential  # exponential / linear

  # 反爬虫策略
  anti_bot:
    user_agent_rotation: true
    user_agents:
      - "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
      - "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"
      - "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36"
    proxy_rotation: true
    request_delay:
      min: 1000ms
      max: 3000ms

  # 浏览器指纹
  fingerprint:
    stealth: true
    disable_webdriver: true
    randomize_viewport: true
```
