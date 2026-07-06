package rulerepo

// loyalsoldierRepo returns the index for Loyalsoldier/clash-rules.
// Clash Premium 规则集，27.5k stars，每天自动构建。
// 规则文件位于 jsDelivr CDN: https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/{name}.txt
func loyalsoldierRepo() Repo {
	base := "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release"
	makeURL := func(name string) string {
		return base + "/" + name + ".txt"
	}

	return Repo{
		Name:        RepoLoyalsoldier,
		DisplayName: "Loyalsoldier/clash-rules",
		Description: "专门为 Clash Premium 内核优化的规则集，27.5k Stars，每天自动构建，含代理/直连/广告/Apple/Google 等",
		BaseURL:     base,
		Rules: []RuleMeta{
			{ID: "loyalsoldier/proxy", Name: "proxy", Category: "Proxy", Tags: []string{"代理", "proxy"}, Description: "代理规则（境外常用网站）", URL: makeURL("proxy"), Behavior: "domain"},
			{ID: "loyalsoldier/gfw", Name: "gfw", Category: "Proxy", Tags: []string{"gfw", "墙"}, Description: "GFW 规则", URL: makeURL("gfw"), Behavior: "domain"},
			{ID: "loyalsoldier/greatfire", Name: "greatfire", Category: "Proxy", Tags: []string{"greatfire", "防火墙"}, Description: "GreatFire 规则", URL: makeURL("greatfire"), Behavior: "domain"},

			{ID: "loyalsoldier/direct", Name: "direct", Category: "Direct", Tags: []string{"直连", "direct"}, Description: "直连规则（国内常用网站）", URL: makeURL("direct"), Behavior: "domain"},
			{ID: "loyalsoldier/private", Name: "private", Category: "Direct", Tags: []string{"私有", "直连"}, Description: "私有网络/内网地址直连", URL: makeURL("private"), Behavior: "domain"},
			{ID: "loyalsoldier/cncidr", Name: "cncidr", Category: "Direct", Tags: []string{"国内", "ip", "cidr"}, Description: "中国大陆 IP 地址段", URL: makeURL("cncidr"), Behavior: "ipcidr"},
			{ID: "loyalsoldier/lancidr", Name: "lancidr", Category: "Direct", Tags: []string{"局域网", "lan", "cidr"}, Description: "局域网及保留 IP 地址段", URL: makeURL("lancidr"), Behavior: "ipcidr"},

			{ID: "loyalsoldier/reject", Name: "reject", Category: "Ads", Tags: []string{"广告", "reject"}, Description: "广告/恶意网站拦截", URL: makeURL("reject"), Behavior: "domain"},

			{ID: "loyalsoldier/apple", Name: "apple", Category: "Apple", Tags: []string{"apple", "苹果"}, Description: "Apple 服务", URL: makeURL("apple"), Behavior: "domain"},
			{ID: "loyalsoldier/icloud", Name: "icloud", Category: "Apple", Tags: []string{"apple", "icloud"}, Description: "iCloud 服务", URL: makeURL("icloud"), Behavior: "domain"},
			{ID: "loyalsoldier/google", Name: "google", Category: "Search", Tags: []string{"google", "搜索"}, Description: "Google 服务", URL: makeURL("google"), Behavior: "domain"},

			{ID: "loyalsoldier/telegramcidr", Name: "telegramcidr", Category: "Social", Tags: []string{"telegram", "ip", "cidr"}, Description: "Telegram 使用的 IP 地址段", URL: makeURL("telegramcidr"), Behavior: "ipcidr"},
			{ID: "loyalsoldier/applications", Name: "applications", Category: "Other", Tags: []string{"应用", "进程"}, Description: "应用进程规则", URL: makeURL("applications"), Behavior: "classical"},
			{ID: "loyalsoldier/tld-not-cn", Name: "tld-not-cn", Category: "Proxy", Tags: []string{"tld", "顶级域名"}, Description: "非中国大陆使用的顶级域名", URL: makeURL("tld-not-cn"), Behavior: "domain"},
		},
	}
}