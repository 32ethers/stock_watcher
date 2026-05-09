# stock_watcher — 自托管金融信息看板

一个用 Go 语言编写的轻量级自托管金融数据看板，支持 A 股、美股指数、商品期货、加密货币、Binance 理财及以太坊 Gas 价格的实时查看。

## 功能

| 卡片 | 数据源 | 内容 |
|---|---|---|
| 大盘指数 | Tencent Finance + Yahoo Finance | 上证、深证、创业板、恒生、道指、纳指、标普 500 |
| 我的持仓 | Tencent Finance | 用户配置的 A 股/ETF 实时行情 |
| 商品期货 | Yahoo Finance | 黄金、白银、布油、WTI 原油、天然气、铜 |
| 加密货币 | Binance API v3 | 用户配置的交易对 24h 行情 |
| Binance 理财 | Binance Earn API | 灵活理财产品的实时 APR |
| Ethereum Gas | Etherscan Gas Tracker | Safe / Propose / Fast Gas 价格 (GWei) |

## 快速开始

### 编译

```bash
git clone <repo-url> && cd stock_watcher
go build -o stock-watcher .
```

### 配置

```bash
cp config.sample.yaml config.yaml
vim config.yaml
```

- `port` — HTTP 监听端口（默认 8000）
- `stocks_cn` — 持仓股票/ETF 代码（如 `600519`, `510300`）
- `crypto` — 加密货币交易对（需用 Binance 符号，如 `BTCUSDT`）
- `earn` — Binance 理财资产（如 `BTC`, `ETH`, `USDT`）
- `etherscan_api_key` — Etherscan API 密钥（[免费申请](https://etherscan.io/myapikey)）

### 运行

```bash
./stock-watcher
```

打开浏览器访问 `http://localhost:8000`。

## 部署

项目提供了 `sync.sh` 脚本，可交叉编译为 `linux/amd64` 并通过 rsync 推送到远程服务器。修改脚本中的目标地址后执行：

```bash
./sync.sh
```

建议在生产环境中使用 nginx/Caddy 反向代理并配置 HTTPS。

## 技术栈

- **后端**: Go 标准库 + `yaml.v3` + `golang.org/x/text`（GBK 解码）
- **前端**: 纯 HTML/CSS/JavaScript（无框架），通过 `fetch()` 调用后端 REST API
- **构建产物**: 单文件静态二进制，无需运行时依赖
