# SecFlow Rod 爬虫 WAF 绕过技术文档

## 概述

本文档详细说明 SecFlow 客户端使用的 go-rod 爬虫的 WAF 绕过技术。所有爬虫均采用 Rod 版本以应对日益严格的反爬虫机制。

## WAF 绕过技术架构

```
┌─────────────────────────────────────────────────────────────────┐
│                    WAF 绕过技术层次                              │
├─────────────────────────────────────────────────────────────────┤
│  Layer 1: 浏览器指纹混淆                                          │
│  ├── User-Agent 轮换 (20+ 浏览器配置文件)                          │
│  ├── Viewport 随机化 (桌面 + 移动设备分辨率)                        │
│  ├── Navigator 属性覆盖 (webdriver, plugins, languages 等)        │
│  └── WebGL/Canvas 指纹随机化                                      │
├─────────────────────────────────────────────────────────────────┤
│  Layer 2: 人体行为模拟                                            │
│  ├── 贝塞尔曲线鼠标移动                                           │
│  ├── 可变速度键盘输入                                             │
│  ├── 自然滚动行为                                                 │
│  └── 随机暂停/阅读模拟                                            │
├─────────────────────────────────────────────────────────────────┤
│  Layer 3: WAF 检测与响应                                          │
│  ├── 常见 WAF 模式识别                                            │
│  ├── Cloudflare 挑战检测                                          │
│  ├── 阿里云/腾讯云 WAF 检测                                       │
│  ├── 自动重试与降级策略                                           │
│  └── 代理轮换支持                                                 │
├─────────────────────────────────────────────────────────────────┤
│  Layer 4: 隐身模式                                                │
│  ├── go-stealth 库集成                                           │
│  ├── Chrome 自动化标记移除                                         │
│  ├── 资源拦截优化                                                │
│  └── Cookie/Storage 隔离                                          │
└─────────────────────────────────────────────────────────────────┘
```

## 核心绕过技术详解

### 1. 浏览器指纹混淆

#### 1.1 User-Agent 轮换

```go
// 支持的 UA 列表（部分）
var userAgents = []string{
    // Chrome on Windows
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
    // Chrome on macOS
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
    // Firefox on Windows
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:123.0) Gecko/20100101 Firefox/123.0",
    // Safari on iOS
    "Mozilla/5.0 (iPhone; CPU iPhone OS 17_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.3 Mobile/15E148 Safari/604.1",
    // Mobile Chrome
    "Mozilla/5.0 (Linux; Android 14; Pixel 7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Mobile Safari/537.36",
}
```

#### 1.2 Viewport 随机化

支持 15+ 种分辨率：
- 桌面: 1920x1080, 1366x768, 1440x900, 1536x864, 1280x720, 1600x900, 1280x800, 1680x1050, 2560x1440, 3840x2160
- 移动: 390x844 (iPhone 14), 414x896 (iPhone 11), 375x667 (iPhone 8), 412x915 (Pixel 7), 360x800 (Samsung)

#### 1.3 高级指纹覆盖

```javascript
// 注入到页面的 JavaScript 脚本
navigator.webdriver = undefined;           // 移除自动化标记
navigator.plugins = [1, 2, 3, 4, 5];      // 伪装插件列表
navigator.languages = ['zh-CN', 'zh', 'en']; // 真实语言设置
navigator.hardwareConcurrency = 4-8;       // CPU 核心数伪装
navigator.deviceMemory = [2, 4, 8];        // 内存伪装
navigator.platform = 'Win32|MacIntel';    // 平台伪装
navigator.maxTouchPoints = 0|1-10;         // 触控点伪装
```

#### 1.4 WebGL/Canvas 指纹混淆

```javascript
// WebGL 供应商/渲染器伪装
WebGLRenderingContext.prototype.getParameter = function(param) {
    if (param === 37445) return 'Intel Inc.';           // UNMASKED_VENDOR
    if (param === 37446) return 'Intel Iris OpenGL';    // UNMASKED_RENDERER
    return getParameter.call(this, param);
};

// Canvas 文字绘制微扰动
context.fillText = function(...args) {
    args[1] += (Math.random() - 0.5) * 0.1;  // 0.05px 随机偏移
    args[2] += (Math.random() - 0.5) * 0.1;
    return originalFillText.apply(this, args);
};
```

### 2. 人体行为模拟

#### 2.1 贝塞尔曲线鼠标移动

```go
// 生成自然的鼠标移动轨迹
type controlPoint struct { x, y float64 }

func generateBezierControlPoints(startX, startY, endX, endY float64) []controlPoint {
    // 计算垂直于直线的控制点偏移
    perpX := -dy/distance * distance * 0.3 * signX
    perpY := dx/distance * distance * 0.3 * signY
    
    // 添加随机扰动
    randX := (rand.Float64() - 0.5) * distance * 0.3
    randY := (rand.Float64() - 0.5) * distance * 0.3
    
    return []controlPoint{
        {startX, startY},
        {startX + dx*0.3 + perpX + randX, startY + dy*0.3 + perpY + randY},
        {startX + dx*0.7 + perpX*0.5 + randX*0.5, startY + dy*0.7 + perpY*0.5 + randY*0.5},
        {endX, endY},
    }
}
```

**特点**：
- 15-25 步平滑移动
- 曲线中段速度稍快，起始/结束段稍慢
- 5-20ms 随机间隔

#### 2.2 可变速度键盘输入

```go
func Fill(element *rod.Element, text string) error {
    for i, char := range text {
        element.Type(string(char))
        
        // 基础延迟: 30-150ms
        baseDelay := 30 + rand.Intn(120)
        
        // 5% 概率出现"思考/修正"停顿 (+200-500ms)
        if rand.Float64() > 0.95 {
            baseDelay += 200 + rand.Intn(300)
        }
        
        // 长文本时打字速度加快 (0.7x)
        if len(text) > 50 {
            baseDelay = int(float64(baseDelay) * 0.7)
        }
        
        // 单词边界停顿 (+50-150ms)
        if text[i:i+1] == " " || text[i:i+1] == "." {
            baseDelay += 50 + rand.Intn(100)
        }
        
        time.Sleep(time.Duration(baseDelay) * time.Millisecond)
    }
}
```

#### 2.3 自然滚动行为

```go
func SimulateScrolling(page *rod.Page) error {
    scrolls := 3-8 次随机
    scrollAmount := 100-500px 随机
    
    // 20% 概率向上滚动（模拟回看）
    if rand.Float64() > 0.8 {
        scrollAmount = -scrollAmount
    }
    
    // 平滑滚动 + 300-800ms 随机停顿
    page.Eval(`() => window.scrollBy({ top: %d, behavior: 'smooth' })`, scrollAmount)
}
```

### 3. WAF 检测与响应

#### 3.1 支持的 WAF 检测模式

| WAF 类型 | 检测关键词 | 响应策略 |
|----------|-----------|----------|
| Cloudflare | `#challenge-form`, `.cf-error-wrapper` | 等待 10-15 秒 |
| 阿里云 WAF | `.waf-block-page`, `#waf_tg_error` | 刷新 + 重试 |
| 腾讯云 WAF | `#captcha`, `.qcloud-captcha` | 标记需要人工 |
| 通用 | `403`, `Access Denied`, `被拒绝` | 重试或降级 |
| 验证码 | `.g-recaptcha`, `#nc_1_wrapper` | 标记需要人工 |

#### 3.2 自动重试机制

```go
func NavigateWithRetry(ctx context.Context, page *rod.Page, url string, config *BypassConfig) error {
    maxRetries := 3
    
    for attempt := 0; attempt < maxRetries; attempt++ {
        // 指数退避: 2s, 4s, 8s + 随机 0-5s
        if attempt > 0 {
            backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
            backoff += time.Duration(rand.Intn(5000)) * time.Millisecond
            
            // 旋转代理（如果启用）
            if config.UseProxyRotation {
                proxy := GetNextProxy()
                // 重新创建浏览器实例使用新代理
            }
            
            // 刷新指纹
            ApplyAdvancedFingerprint(page)
            time.Sleep(backoff)
        }
        
        err := SafeNavigate(page, url, config)
        if err == nil {
            return nil
        }
        
        // 只对 WAF 错误重试
        if !isWAFError(err) {
            return err
        }
    }
    
    return fmt.Errorf("max retries exceeded")
}
```

#### 3.3 代理池支持

```go
// 从环境变量加载代理列表
// PROXY_LIST=http://proxy1:port,http://proxy2:port

func LoadProxyPool(source string) error {
    proxyList := strings.Split(getEnvOrDefault("PROXY_LIST", ""), ",")
    for _, p := range proxyList {
        proxyPool = append(proxyPool, strings.TrimSpace(p))
    }
}

func GetNextProxy() string {
    // 轮询选择下一个代理
    proxy := proxyPool[currentProxy]
    currentProxy = (currentProxy + 1) % len(proxyPool)
    return proxy
}
```

### 4. 隐身模式配置

#### 4.1 Chrome 参数

```go
l.Set("--disable-blink-features", "AutomationControlled")  // 移除自动化标记
l.Set("--disable-web-security")
l.Set("--disable-features", "IsolateOrigins,site-per-process")
l.Set("--window-size", "1920,1080")
l.Set("--no-sandbox")
l.Set("--disable-setuid-sandbox")
l.Set("--disable-dev-shm-usage")
l.Set("--disable-accelerated-2d-canvas")
l.Set("--disable-gpu")
```

#### 4.2 go-stealth 集成

```go
func NewPage(browser *rod.Browser) (*rod.Page, error) {
    page, err := stealth.Page(browser)
    if err != nil {
        return nil, err
    }
    // 设置默认视口
    page.SetViewport(...)
    return page, nil
}
```

## 配置选项

### BypassConfig 结构

```go
type BypassConfig struct {
    // 鼠标模拟
    SimulateMouse bool
    
    // 随机延迟
    RandomDelays bool
    MinDelay int
    MaxDelay int
    
    // 视口随机化
    RandomViewport bool
    
    // UA 轮换
    RotateUserAgent bool
    
    // 代理轮换
    UseProxyRotation bool
    ProxyPool string
    
    // 高级指纹混淆
    AdvancedFingerprint bool
    
    // WAF 自动重试
    AutoRetryOnWAF bool
    MaxRetries int
    
    // 自定义请求头
    CustomHeaders map[string]string
}
```

### 使用示例

```go
config := &rodutil.BypassConfig{
    RotateUserAgent:       true,
    RandomViewport:        true,
    AdvancedFingerprint:   true,
    SimulateMouse:         true,
    RandomDelays:          true,
    MinDelay:              500,
    MaxDelay:              1500,
    AutoRetryOnWAF:        true,
    MaxRetries:            3,
}

// 在爬虫中使用
page, err := rodutil.NewPage(browser)
if err := rodutil.SafeNavigate(page, url, config); err != nil {
    // 处理错误
}
```

## 爬虫清单

### Rod 版本爬虫 (已启用)

| 源名称 | WAF 类型 | 绕过难度 | 备注 |
|--------|----------|----------|------|
| avd-rod | 阿里云 WAF | ⭐⭐⭐⭐ | 高危目标 |
| seebug-rod | 知道创宇 | ⭐⭐⭐⭐ | 高危目标 |
| ti-rod | 腾讯云 WAF | ⭐⭐⭐ | API 驱动 |
| kev-rod | 无 | ⭐ | 官方 API |
| struts2-rod | Apache | ⭐⭐ | 低风险 |
| chaitin-rod | 长亭 | ⭐⭐⭐ | 中等风险 |
| oscs-rod | 开源中国 | ⭐⭐ | 中等风险 |
| threatbook-rod | 微步 | ⭐⭐⭐ | 中等风险 |
| venustech-rod | 启明 | ⭐⭐⭐ | 中等风险 |
| cnvd-rod | CNVD WAF | ⭐⭐⭐⭐ | 高危目标 |
| cnnvd-rod | CNNVD WAF | ⭐⭐⭐⭐ | 高危目标 |
| nsfocus-rod | 绿盟 WAF | ⭐⭐⭐ | 中高风险 |
| qianxin-rod | 奇安信 | ⭐⭐⭐ | 中高风险 |
| antiy-rod | 安天 | ⭐⭐ | 中等风险 |
| dbappsecurity-rod | 安恒 | ⭐⭐⭐ | 中高风险 |

## 故障排查

### 常见问题

#### 1. 返回空数据

```bash
# 启用调试模式查看页面内容
go run cmd/test-rod/main.go -source avd-rod -debug

# 检查 WAF 拦截
# 查看页面源码中的 403/验证码/挑战页面
```

#### 2. 频繁被封

```bash
# 增加延迟配置
export SECFLOW_MIN_DELAY=1000
export SECFLOW_MAX_DELAY=3000

# 启用代理轮换
export PROXY_LIST=http://proxy1:8080,http://proxy2:8080
```

#### 3. 内存占用高

```go
// 确保及时关闭页面
defer page.Close()

// 限制并发数
semaphore := make(chan struct{}, 2)
```

### 调试技巧

```go
// 1. 截图保存
img, _ := page.Screenshot(true, nil)
os.WriteFile("debug.png", img, 0644)

// 2. 保存页面源码
html, _ := page.HTML()
os.WriteFile("debug.html", []byte(html), 0644)

// 3. 检查 WAF 状态
wafDetected, wafName := rodutil.DetectWAF(page)
if wafDetected {
    log.Printf("WAF detected: %s", wafName)
}
```

## 环境变量

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `SECFLOW_HEADLESS` | `true` | 是否 headless 运行 |
| `PROXY_LIST` | (空) | 代理列表，逗号分隔 |
| `SECFLOW_MIN_DELAY` | `100` | 最小延迟 (ms) |
| `SECFLOW_MAX_DELAY` | `500` | 最大延迟 (ms) |
| `SECFLOW_MAX_RETRIES` | `3` | 最大重试次数 |
| `DISPLAY` | (系统) | X11 display (非 headless 时) |

## 性能对比

| 指标 | HTTP 版本 | Rod 版本 |
|------|-----------|----------|
| 启动时间 | < 100ms | 2-5s |
| 内存占用 | 10-50MB | 100-500MB |
| 单页抓取 | 0.5-2s | 3-10s |
| WAF 绕过率 | 30-50% | 80-95% |
| CPU 使用 | 低 | 中高 |

## 更新日志

### 2024-03
- 添加高级指纹混淆 (WebGL/Canvas)
- 添加贝塞尔曲线鼠标移动
- 添加可变速度键盘输入
- 添加代理池支持
- 添加 WAF 自动重试机制
- 添加国内安全厂商 Rod 爬虫 (CNVD, CNNVD, NSFOCUS, 奇安信, 安天, 安恒)
