package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Port     int           `yaml:"port"`
	StocksCN []StockConfig `yaml:"stocks_cn"`
	Crypto   []CryptoConf  `yaml:"crypto"`
}

type StockConfig struct {
	Code string `yaml:"code"`
}

type CryptoConf struct {
	Name   string `yaml:"name"`
	Symbol string `yaml:"symbol"`
}

const indexHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>金融信息看板</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, "PingFang SC", "Microsoft YaHei", sans-serif;
            background: #0d1117; color: #c9d1d9; min-height: 100vh;
        }
        .header {
            text-align: center; padding: 20px 16px 12px; border-bottom: 1px solid #21262d;
        }
        .header h1 { font-size: 24px; font-weight: 600; color: #f0f6fc; margin-bottom: 4px; }
        .header .updated { font-size: 13px; color: #8b949e; }
        .grid {
            display: grid; grid-template-columns: 1fr 1fr; gap: 16px;
            max-width: 1200px; margin: 16px auto; padding: 0 16px;
        }
        .card {
            background: #161b22; border: 1px solid #21262d; border-radius: 8px; overflow: hidden;
        }
        .card-header {
            padding: 12px 16px; border-bottom: 1px solid #21262d;
            font-size: 15px; font-weight: 600; color: #f0f6fc;
            display: flex; align-items: center; gap: 8px; justify-content: space-between;
        }
        .card-header .left { display: flex; align-items: center; gap: 8px; }
        .card-header .icon { font-size: 18px; }
        .card-header .time { font-size: 11px; color: #8b949e; font-weight: 400; }
        .card-body { padding: 0; min-height: 60px; position: relative; }
        .loading {
            padding: 24px 16px; text-align: center; color: #484f58; font-size: 13px;
        }
        .loading-dots::after { content: ''; animation: dots 1.5s steps(4, end) infinite; }
        @keyframes dots { 0%{content:''} 25%{content:'.'} 50%{content:'..'} 75%{content:'...'} }
        .error-msg { padding: 16px; text-align: center; color: #f85149; font-size: 13px; }
        table { width: 100%; border-collapse: collapse; }
        th {
            text-align: left; padding: 8px 16px; font-size: 12px; font-weight: 500;
            color: #8b949e; text-transform: uppercase; letter-spacing: 0.5px; background: #0d1117;
        }
        td { padding: 10px 16px; font-size: 14px; border-top: 1px solid #21262d; }
        tr:hover { background: #1c2128; }
        .name-cell { display: flex; flex-direction: column; }
        .name-cell .main-name { font-weight: 500; color: #f0f6fc; }
        .name-cell .sub-code { font-size: 12px; color: #8b949e; }
        .price { font-weight: 500; font-variant-numeric: tabular-nums; color: #f0f6fc; }
        .up { color: #f85149; }
        .down { color: #3fb950; }
        .flat { color: #8b949e; }
        .badge {
            display: inline-block; padding: 2px 6px; border-radius: 4px;
            font-size: 12px; font-weight: 500; font-variant-numeric: tabular-nums;
        }
        .badge-up { background: rgba(248, 81, 73, 0.15); color: #f85149; }
        .badge-down { background: rgba(63, 185, 80, 0.15); color: #3fb950; }
        .badge-flat { background: rgba(139, 148, 158, 0.15); color: #8b949e; }
        .empty-state { padding: 24px 16px; text-align: center; color: #8b949e; font-size: 14px; }
        .footer { text-align: center; padding: 16px; color: #484f58; font-size: 12px; }
        a { color: #58a6ff; text-decoration: none; }
        a:hover { text-decoration: underline; }
        @media (max-width: 768px) {
            .grid { grid-template-columns: 1fr; }
            .header h1 { font-size: 20px; }
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>金融信息看板</h1>
        <div class="updated" id="globalTime">加载中...</div>
    </div>
    <div class="grid">
        <div class="card">
            <div class="card-header">
                <span class="left"><span class="icon">&#x1F4C8;</span> 大盘指数</span>
                <span class="time" id="indices-time"></span>
            </div>
            <div class="card-body" id="indices-body">
                <div class="loading">加载中<span class="loading-dots"></span></div>
            </div>
        </div>
        <div class="card">
            <div class="card-header">
                <span class="left"><span class="icon">&#x1F4B0;</span> 我的持仓</span>
                <span class="time" id="stocks-time"></span>
            </div>
            <div class="card-body" id="stocks-body">
                <div class="loading">加载中<span class="loading-dots"></span></div>
            </div>
        </div>
        <div class="card">
            <div class="card-header">
                <span class="left"><span class="icon">&#x26FD;</span> 商品期货</span>
                <span class="time" id="commodities-time"></span>
            </div>
            <div class="card-body" id="commodities-body">
                <div class="loading">加载中<span class="loading-dots"></span></div>
            </div>
        </div>
        <div class="card">
            <div class="card-header">
                <span class="left"><span class="icon">&#x1F535;</span> 加密货币</span>
                <span class="time" id="crypto-time"></span>
            </div>
            <div class="card-body" id="crypto-body">
                <div class="loading">加载中<span class="loading-dots"></span></div>
            </div>
        </div>
    </div>
    <div class="footer">
        按 F5 刷新数据 &middot; 数据来源：腾讯财经 / Yahoo Finance / Binance
    </div>
<script>
function badgeClass(pct) { return pct > 0 ? 'up' : pct < 0 ? 'down' : 'flat'; }
function fmtNum(n, d) { return n.toLocaleString('en-US', {minimumFractionDigits: d, maximumFractionDigits: d}); }
function fmtChange(v) { return (v >= 0 ? '+' : '') + v.toFixed(2); }
function stockUrl(c) { return 'https://quote.eastmoney.com/' + (/^(6|51|58)/.test(c) ? 'sh' : 'sz') + c + '.html'; }

function renderTable(bodyId, data, cols) {
    var el = document.getElementById(bodyId);
    if (!data || data.length === 0) { el.innerHTML = '<div class="empty-state">暂无数据</div>'; return; }
    var h = '<table><thead><tr>';
    for (var i = 0; i < cols.length; i++) h += '<th>' + cols[i].label + '</th>';
    h += '</tr></thead><tbody>';
    for (var i = 0; i < data.length; i++) {
        var d = data[i]; h += '<tr>';
        for (var j = 0; j < cols.length; j++) h += '<td>' + cols[j].fn(d) + '</td>';
        h += '</tr>';
    }
    h += '</tbody></table>';
    el.innerHTML = h;
}

function nameCell(name, sub, url) {
    var inner = '<span class="name-cell"><span class="main-name">' + name + '</span>';
    if (sub) inner += '<span class="sub-code">' + sub + '</span>';
    inner += '</span>';
    return url ? '<a href="' + url + '" target="_blank">' + inner + '</a>' : inner;
}

function badge(pct) {
    var cls = badgeClass(pct);
    return '<span class="badge badge-' + cls + '">' + fmtChange(pct) + '%</span>';
}

function cls(pct) { return badgeClass(pct); }

function renderIndices(data) {
    renderTable('indices-body', data, [
        {label:'指数', fn: function(d){return nameCell(d.name, null, d.url)}},
        {label:'最新', fn: function(d){return '<span class="price">' + fmtNum(d.value, 2) + '</span>'}},
        {label:'涨跌额', fn: function(d){return '<span class="' + cls(d.change_pct) + '">' + fmtChange(d.change_amt) + '</span>'}},
        {label:'涨跌幅', fn: function(d){return badge(d.change_pct)}},
    ]);
}

function renderStocks(data) {
    renderTable('stocks-body', data, [
        {label:'名称', fn: function(d){return nameCell(d.name, d.code, stockUrl(d.code))}},
        {label:'最新价', fn: function(d){return '<span class="price">' + d.price.toFixed(2) + '</span>'}},
        {label:'涨跌额', fn: function(d){return '<span class="' + cls(d.change_pct) + '">' + fmtChange(d.change_amt) + '</span>'}},
        {label:'涨跌幅', fn: function(d){return badge(d.change_pct)}},
    ]);
}

function renderCommodities(data) {
    renderTable('commodities-body', data, [
        {label:'品种', fn: function(d){return nameCell(d.name, null, d.url)}},
        {label:'最新价', fn: function(d){return '<span class="price">' + fmtNum(d.price, 2) + '</span>'}},
        {label:'涨跌额', fn: function(d){return '<span class="' + cls(d.change_pct) + '">' + fmtChange(d.change_amt) + '</span>'}},
        {label:'涨跌幅', fn: function(d){return badge(d.change_pct)}},
    ]);
}

function renderCrypto(data) {
    renderTable('crypto-body', data, [
        {label:'名称', fn: function(d){return nameCell(d.name, d.symbol, 'https://www.binance.com/zh-CN/trade/' + d.symbol)}},
        {label:'价格 (USD)', fn: function(d){return '<span class="price">$' + fmtNum(d.price_usd, 2) + '</span>'}},
        {label:'涨跌幅', fn: function(d){return badge(d.change_pct_24h)}},
    ]);
}

function showError(id, msg) { document.getElementById(id).innerHTML = '<div class="error-msg">' + msg + '</div>'; }
function nowStr() { return new Date().toLocaleString('zh-CN', {hour12: false}); }

function fetchBlock(name, render, bodyId, timeId) {
    fetch('/api/' + name).then(function(r){return r.json()}).then(function(j){
        if (j.error) showError(bodyId, j.error); else render(j.data);
        if (timeId) document.getElementById(timeId).textContent = nowStr();
    }).catch(function(e){ showError(bodyId, '请求失败: ' + e.message); });
}

fetchBlock('indices', renderIndices, 'indices-body', 'indices-time');
fetchBlock('stocks', renderStocks, 'stocks-body', 'stocks-time');
fetchBlock('commodities', renderCommodities, 'commodities-body', 'commodities-time');
fetchBlock('crypto', renderCrypto, 'crypto-body', 'crypto-time');

document.getElementById('globalTime').textContent = '更新时间：' + nowStr();
Promise.all([
    fetch('/api/indices').then(function(){document.getElementById('indices-time').textContent = nowStr()}),
]).then(function(){ document.getElementById('globalTime').textContent = '更新时间：' + nowStr(); });
</script>
</body>
</html>
`
type StockInfo struct {
	Code      string  `json:"code"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	ChangePct float64 `json:"change_pct"`
	ChangeAmt float64 `json:"change_amt"`
	Volume    float64 `json:"volume"`
	Turnover  float64 `json:"turnover"`
}

type IndexInfo struct {
	Name      string  `json:"name"`
	Value     float64 `json:"value"`
	ChangePct float64 `json:"change_pct"`
	ChangeAmt float64 `json:"change_amt"`
	URL       string  `json:"url"`
}

type CommodityInfo struct {
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	ChangePct float64 `json:"change_pct"`
	ChangeAmt float64 `json:"change_amt"`
	URL       string  `json:"url"`
}

type CryptoInfo struct {
	Symbol       string  `json:"symbol"`
	Name         string  `json:"name"`
	PriceUSD     float64 `json:"price_usd"`
	ChangePct24h float64 `json:"change_pct_24h"`
}

type APIResponse struct {
	Data  interface{} `json:"data"`
	Error string      `json:"error"`
}

var (
	CNIndexSymbols = []struct {
		Key, Name, URL string
	}{
		{"sh000001", "上证指数", "https://cn.investing.com/indices/shanghai-composite"},
		{"sz399001", "深证成指", "https://cn.investing.com/indices/szse-component"},
		{"sz399006", "创业板指", "https://cn.investing.com/indices/chinext"},
	}

	GlobalIndexSymbols = []struct {
		Key, Name, URL string
	}{
		{"hkHSI", "恒生指数", "https://cn.investing.com/indices/hang-sen-40"},
		{"us.DJI", "道琼斯", "https://cn.investing.com/indices/us-30"},
		{"usNDX", "纳斯达克100", "https://cn.investing.com/indices/nq-100"},
	}

	YahooIndices = map[string]struct {
		Name, URL string
	}{
		"^GSPC": {"标普500", "https://cn.investing.com/indices/us-spx-500"},
	}

	CommodityTickers = map[string]struct {
		Name, URL string
	}{
		"GC=F":  {"黄金", "https://cn.investing.com/commodities/gold"},
		"SI=F":  {"白银", "https://cn.investing.com/commodities/silver"},
		"BZ=F":  {"布伦特原油", "https://cn.investing.com/commodities/brent-oil"},
		"CL=F":  {"WTI原油", "https://cn.investing.com/commodities/crude-oil"},
		"NG=F":  {"天然气", "https://cn.investing.com/commodities/natural-gas"},
		"HG=F":  {"铜", "https://cn.investing.com/commodities/copper"},
	}

	tencentRe = regexp.MustCompile(`v_s_([\w.]+)=["'](.+)["']`)

	httpClient = &http.Client{Timeout: 15 * time.Second}

	yahooHeaders = map[string]string{
		"User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
	}
)

func safeFloat(s string) float64 {
	s = strings.NewReplacer(",", "", "(", "", ")", "", "%", "").Replace(s)
	s = strings.TrimSpace(s)
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}

func ensureUTF8(body []byte) string {
	if utf8.Valid(body) {
		return string(body)
	}
	reader := transform.NewReader(strings.NewReader(string(body)), simplifiedchinese.GBK.NewDecoder())
	decoded, _ := io.ReadAll(reader)
	return string(decoded)
}

func fetchURL(url string) (string, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return ensureUTF8(body), nil
}

func parseTencent(text string) map[string][]string {
	result := map[string][]string{}
	for _, line := range strings.Split(text, ";") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		m := tencentRe.FindStringSubmatch(line)
		if len(m) >= 3 {
			result[m[1]] = strings.Split(m[2], "~")
		}
	}
	return result
}

func codeToTencent(code string) string {
	if strings.HasPrefix(code, "6") || strings.HasPrefix(code, "51") || strings.HasPrefix(code, "58") {
		return "sh" + code
	}
	return "sz" + code
}

func fetchYahooTicker(ticker string) (map[string]float64, error) {
	url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=2d", ticker)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range yahooHeaders {
		req.Header.Set(k, v)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var data struct {
		Chart struct {
			Result []struct {
				Meta struct {
					RegularMarketPrice  float64 `json:"regularMarketPrice"`
					ChartPreviousClose  float64 `json:"chartPreviousClose"`
				} `json:"meta"`
			} `json:"result"`
		} `json:"chart"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	if len(data.Chart.Result) == 0 {
		return nil, fmt.Errorf("no data")
	}
	meta := data.Chart.Result[0].Meta
	price := meta.RegularMarketPrice
	prev := meta.ChartPreviousClose
	changePct := 0.0
	changeAmt := 0.0
	if prev != 0 {
		changePct = (price - prev) / prev * 100
		changeAmt = price - prev
	}
	return map[string]float64{
		"price":      price,
		"prev":       prev,
		"change_pct": math.Round(changePct*100) / 100,
		"change_amt": math.Round(changeAmt*100) / 100,
	}, nil
}

func handleStocks(cfg *Config) APIResponse {
	codes := make([]string, len(cfg.StocksCN))
	for i, s := range cfg.StocksCN {
		codes[i] = codeToTencent(s.Code)
	}
	syms := make([]string, len(codes))
	for i, c := range codes {
		syms[i] = "s_" + c
	}
	url := fmt.Sprintf("http://qt.gtimg.cn/r=0.%d&q=%s", time.Now().UnixMilli(), strings.Join(syms, ","))
	text, err := fetchURL(url)
	if err != nil {
		return APIResponse{Error: fmt.Sprintf("A股/ETF数据获取失败: %v", err)}
	}
	parsed := parseTencent(text)
	var results []StockInfo
	for _, sc := range cfg.StocksCN {
		sym := codeToTencent(sc.Code)
		fields, ok := parsed[sym]
		if !ok || len(fields) < 10 {
			continue
		}
		results = append(results, StockInfo{
			Code:      sc.Code,
			Name:      fields[1],
			Price:     safeFloat(fields[3]),
			ChangePct: safeFloat(fields[5]),
			ChangeAmt: safeFloat(fields[4]),
			Volume:    safeFloat(fields[6]),
			Turnover:  safeFloat(fields[7]),
		})
	}
	return APIResponse{Data: results}
}

func handleIndices() APIResponse {
	var allSyms []struct {
		Key, Name, URL string
	}
	allSyms = append(allSyms, CNIndexSymbols...)
	allSyms = append(allSyms, GlobalIndexSymbols...)

	syms := make([]string, len(allSyms))
	for i, s := range allSyms {
		syms[i] = "s_" + s.Key
	}
	url := fmt.Sprintf("http://qt.gtimg.cn/r=0.%d&q=%s", time.Now().UnixMilli(), strings.Join(syms, ","))
	text, err := fetchURL(url)
	if err != nil {
		return APIResponse{Error: fmt.Sprintf("指数获取失败: %v", err)}
	}
	parsed := parseTencent(text)
	var results []IndexInfo
	for _, s := range allSyms {
		fields, ok := parsed[s.Key]
		if !ok || len(fields) < 8 {
			continue
		}
		results = append(results, IndexInfo{
			Name:      s.Name,
			Value:     safeFloat(fields[3]),
			ChangePct: safeFloat(fields[5]),
			ChangeAmt: safeFloat(fields[4]),
			URL:       s.URL,
		})
	}

	for ticker, info := range YahooIndices {
		data, err := fetchYahooTicker(ticker)
		if err != nil {
			continue
		}
		results = append(results, IndexInfo{
			Name:      info.Name,
			Value:     data["price"],
			ChangePct: data["change_pct"],
			ChangeAmt: data["change_amt"],
			URL:       info.URL,
		})
	}
	return APIResponse{Data: results}
}

func handleCommodities() APIResponse {
	type result struct {
		ticker string
		data   map[string]float64
		err    error
	}

	tickers := make([]string, 0, len(CommodityTickers))
	for t := range CommodityTickers {
		tickers = append(tickers, t)
	}

	ch := make(chan result, len(tickers))
	var wg sync.WaitGroup
	for _, t := range tickers {
		wg.Add(1)
		go func(ticker string) {
			defer wg.Done()
			data, err := fetchYahooTicker(ticker)
			ch <- result{ticker, data, err}
		}(t)
	}
	go func() { wg.Wait(); close(ch) }()

	var results []CommodityInfo
	var errs []string
	for r := range ch {
		if r.err != nil {
			errs = append(errs, r.ticker)
			continue
		}
		info := CommodityTickers[r.ticker]
		results = append(results, CommodityInfo{
			Name:      info.Name,
			Price:     r.data["price"],
			ChangePct: r.data["change_pct"],
			ChangeAmt: r.data["change_amt"],
			URL:       info.URL,
		})
	}
	errStr := ""
	if len(errs) > 0 {
		errStr = fmt.Sprintf("商品数据获取失败: %s", strings.Join(errs, ", "))
	}
	return APIResponse{Data: results, Error: errStr}
}

func handleCrypto(cfg *Config) APIResponse {
	if len(cfg.Crypto) == 0 {
		return APIResponse{Data: []CryptoInfo{}}
	}
	symbols := make([]string, len(cfg.Crypto))
	nameMap := map[string]string{}
	for i, c := range cfg.Crypto {
		symbols[i] = c.Symbol
		nameMap[c.Symbol] = c.Name
	}
	symsJSON, _ := json.Marshal(symbols)
	url := fmt.Sprintf("https://api3.binance.com/api/v3/ticker/24hr?symbols=%s", string(symsJSON))
	text, err := fetchURL(url)
	if err != nil {
		return APIResponse{Error: fmt.Sprintf("加密货币获取失败: %v", err)}
	}

	var raw []struct {
		Symbol             string `json:"symbol"`
		LastPrice          string `json:"lastPrice"`
		PriceChangePercent string `json:"priceChangePercent"`
	}
	if err := json.Unmarshal([]byte(text), &raw); err != nil {
		return APIResponse{Error: fmt.Sprintf("加密货币解析失败: %v", err)}
	}

	var results []CryptoInfo
	for _, r := range raw {
		name := r.Symbol
		if n, ok := nameMap[r.Symbol]; ok {
			name = n
		}
		results = append(results, CryptoInfo{
			Symbol:       r.Symbol,
			Name:         name,
			PriceUSD:     safeFloat(r.LastPrice),
			ChangePct24h: safeFloat(r.PriceChangePercent),
		})
	}
	return APIResponse{Data: results}
}

func loadConfig() *Config {
	cfg := &Config{Port: 8000}
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		data, err = os.ReadFile("config.sample.yaml")
		if err != nil {
			log.Fatal("No config file found")
		}
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		log.Fatal("Invalid config: ", err)
	}
	if cfg.Port == 0 {
		cfg.Port = 8000
	}
	return cfg
}

func main() {
	cfg := loadConfig()
	log.Printf("Stock Watcher starting on :%d | %d stocks, %d crypto", cfg.Port, len(cfg.StocksCN), len(cfg.Crypto))

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(indexHTML))
	})

	apiHandler := func(fn func() APIResponse) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			resp := fn()
			json.NewEncoder(w).Encode(resp)
		}
	}

	apiHandlerCfg := func(fn func(*Config) APIResponse) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			resp := fn(cfg)
			json.NewEncoder(w).Encode(resp)
		}
	}

	mux.HandleFunc("/api/indices", apiHandler(handleIndices))
	mux.HandleFunc("/api/stocks", apiHandlerCfg(handleStocks))
	mux.HandleFunc("/api/commodities", apiHandler(handleCommodities))
	mux.HandleFunc("/api/crypto", apiHandlerCfg(handleCrypto))

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	log.Fatal(server.ListenAndServe())
}
