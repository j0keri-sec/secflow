# Rodutil - Go-Rod 浏览器自动化工具包

## 简介

Rodutil 是基于 [go-rod](https://github.com/go-rod/rod) 的浏览器自动化工具包，用于 SecFlow 客户端的漏洞爬取。提供了 WAF 绕过、反爬虫对抗等功能。

## 功能特性

- **浏览器管理**: 全局浏览器实例管理，支持复用
- **Stealth 模式**: 内置反检测机制，绕过常见的 bot 检测
- **WAF 绕过**: 自动处理 Cloudflare 等 WAF 保护
- **人类行为模拟**: 鼠标移动、随机延迟、滚动模拟
- **配置灵活**: 支持代理、自定义 User-Agent、视口大小

## 安装

```bash
go get github.com/go-rod/rod
go get github.com/go-rod/stealth
```

## 快速开始

### 基础用法

```go
package main

import (
    "github.com/secflow/client/pkg/rodutil"
)

func main() {
    // 获取浏览器实例
    browser, err := rodutil.GetBrowser(nil)
    if err != nil {
        panic(err)
    }
    
    // 创建新页面（自动启用 stealth 模式）
    page, err := rodutil.NewPage(browser)
    if err != nil {
        panic(err)
    }
    defer page.Close()
    
    // 导航到页面（自动应用绕过技术）
    if err := rodutil.SafeNavigate(page, "https://example.com", nil); err != nil {
        panic(err)
    }
    
    // 执行操作...
}
```

### 配置选项

```go
config := &rodutil.BrowserConfig{
    Headless:     true,           // 无头模式
    Timeout:      30 * time.Second,
    Proxy:        "http://proxy:8080",
    WindowWidth:  1920,
    WindowHeight: 1080,
}

browser, err := rodutil.GetBrowser(config)
```

### 绕过配置

```go
bypassConfig := &rodutil.BypassConfig{
    SimulateMouse:   true,  // 模拟鼠标移动
    RandomDelays:    true,  // 随机延迟
    MinDelay:        100,   // 最小延迟(ms)
    MaxDelay:        500,   // 最大延迟(ms)
    RandomViewport:  true,  // 随机视口大小
    RotateUserAgent: true,  // 轮换 User-Agent
}

// 应用到页面
if err := rodutil.ApplyBypass(page, bypassConfig); err != nil {
    log.Warn(err)
}
```

## API 文档

### Browser 管理

- `GetBrowser(config *BrowserConfig) (*rod.Browser, error)` - 获取浏览器实例
- `NewPage(browser *rod.Browser) (*rod.Page, error)` - 创建 stealth 页面
- `CloseBrowser() error` - 关闭浏览器

### 导航

- `SafeNavigate(page *rod.Page, url string, config *BypassConfig) error` - 安全导航
- `NavigateWithContext(ctx context.Context, page *rod.Page, url string, waitFor string) error` - 带上下文的导航

### 绕过技术

- `ApplyBypass(page *rod.Page, config *BypassConfig) error` - 应用绕过配置
- `BypassCloudflare(page *rod.Page) error` - 绕过 Cloudflare
- `BypassCaptcha(page *rod.Page) error` - 检测验证码
- `StealthMode(page *rod.Page) error` - 启用 stealth 模式
- `HumanLikeBehavior(page *rod.Page) error` - 模拟人类行为

### 辅助函数

- `RandomDelay(minMs, maxMs int)` - 随机延迟
- `RotateUserAgent(page *rod.Page) error` - 轮换 User-Agent
- `RandomizeViewport(page *rod.Page) error` - 随机视口
- `SimulateHumanMouse(page *rod.Page) error` - 模拟鼠标
- `SimulateScrolling(page *rod.Page) error` - 模拟滚动

## 在 Grabber 中使用

```go
type MyCrawlerRod struct {
    log *golog.Logger
}

func (c *MyCrawlerRod) GetUpdate(ctx context.Context, pageLimit int) ([]*grabber.VulnInfo, error) {
    // 获取浏览器
    browser, err := rodutil.GetBrowser(nil)
    if err != nil {
        return nil, err
    }
    
    // 创建页面
    page, err := rodutil.NewPage(browser)
    if err != nil {
        return nil, err
    }
    defer page.Close()
    
    // 应用绕过
    if err := rodutil.ApplyBypass(page, nil); err != nil {
        c.log.Warn(err)
    }
    
    // 导航
    if err := rodutil.SafeNavigate(page, "https://target.com", nil); err != nil {
        return nil, err
    }
    
    // 提取数据...
}
```

## 注意事项

1. **Chrome 依赖**: 需要系统安装 Chrome 或 Chromium
2. **内存占用**: 浏览器实例占用较多内存，建议复用
3. **并发限制**: 单个浏览器实例的并发页面数有限
4. **超时设置**: 根据网络情况调整超时时间

## 故障排除

### Chrome 未找到

```bash
# macOS
brew install --cask google-chrome

# Linux
wget -q -O - https://dl.google.com/linux/linux_signing_key.pub | sudo apt-key add -
sudo sh -c 'echo "deb [arch=amd64] http://dl.google.com/linux/chrome/deb/ stable main" >> /etc/apt/sources.list.d/google.list'
sudo apt-get update
sudo apt-get install google-chrome-stable
```

### 内存不足

- 减少并发页面数
- 及时关闭不需要的页面
- 使用无头模式节省资源

## 参考

- [Go-Rod 文档](https://go-rod.github.io/)
- [Stealth 插件](https://github.com/go-rod/stealth)
