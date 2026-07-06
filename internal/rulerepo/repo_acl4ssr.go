package rulerepo

// acl4ssrRepo returns the index for ACL4SSR/ACL4SSR.
// Classic rule repository with 29 rule files on the master branch.
// Rules are at: https://cdn.jsdelivr.net/gh/ACL4SSR/ACL4SSR@master/Clash/{Name}.list
func acl4ssrRepo() Repo {
	base := "https://cdn.jsdelivr.net/gh/ACL4SSR/ACL4SSR@master/Clash"
	makeURL := func(name string) string {
		return base + "/" + name + ".list"
	}

	return Repo{
		Name:        RepoACL4SSR,
		DisplayName: "ACL4SSR/ACL4SSR",
		Description: "经典规则仓库，老牌分流规则，19+ 规则文件覆盖广告/代理/直连/媒体等，定期自动化更新",
		BaseURL:     base,
		Rules: []RuleMeta{
			{ID: "acl4ssr/BanAD", Name: "BanAD", Category: "Ads", Tags: []string{"广告", "reject"}, Description: "广告拦截合集", URL: makeURL("BanAD"), Behavior: "domain"},
			{ID: "acl4ssr/BanEasyList", Name: "BanEasyList", Category: "Ads", Tags: []string{"广告", "easylist"}, Description: "EasyList 广告规则", URL: makeURL("BanEasyList"), Behavior: "domain"},
			{ID: "acl4ssr/BanEasyListChina", Name: "BanEasyListChina", Category: "Ads", Tags: []string{"广告", "china"}, Description: "EasyList China 广告规则", URL: makeURL("BanEasyListChina"), Behavior: "domain"},
			{ID: "acl4ssr/BanEasyPrivacy", Name: "BanEasyPrivacy", Category: "Ads", Tags: []string{"隐私", "privacy"}, Description: "EasyPrivacy 隐私保护", URL: makeURL("BanEasyPrivacy"), Behavior: "domain"},
			{ID: "acl4ssr/BanProgramAD", Name: "BanProgramAD", Category: "Ads", Tags: []string{"广告", "程序"}, Description: "应用内广告拦截", URL: makeURL("BanProgramAD"), Behavior: "domain"},

			{ID: "acl4ssr/ProxyGFWlist", Name: "ProxyGFWlist", Category: "Proxy", Tags: []string{"代理", "gfw"}, Description: "GFWList 代理规则", URL: makeURL("ProxyGFWlist"), Behavior: "domain"},
			{ID: "acl4ssr/ProxyLite", Name: "ProxyLite", Category: "Proxy", Tags: []string{"代理", "精简"}, Description: "代理规则精简版", URL: makeURL("ProxyLite"), Behavior: "domain"},
			{ID: "acl4ssr/ProxyMedia", Name: "ProxyMedia", Category: "Proxy", Tags: []string{"代理", "媒体"}, Description: "代理媒体规则", URL: makeURL("ProxyMedia"), Behavior: "domain"},

			{ID: "acl4ssr/ChinaDomain", Name: "ChinaDomain", Category: "Direct", Tags: []string{"国内", "直连", "domain"}, Description: "中国大陆域名直连", URL: makeURL("ChinaDomain"), Behavior: "domain"},
			{ID: "acl4ssr/ChinaIp", Name: "ChinaIp", Category: "Direct", Tags: []string{"国内", "直连", "ip"}, Description: "中国大陆 IP 直连", URL: makeURL("ChinaIp"), Behavior: "ipcidr"},
			{ID: "acl4ssr/ChinaIpV6", Name: "ChinaIpV6", Category: "Direct", Tags: []string{"国内", "直连", "ipv6"}, Description: "中国大陆 IPv6 直连", URL: makeURL("ChinaIpV6"), Behavior: "ipcidr"},
			{ID: "acl4ssr/ChinaCompanyIp", Name: "ChinaCompanyIp", Category: "Direct", Tags: []string{"国内", "公司", "ip"}, Description: "中国大陆企业 IP 直连", URL: makeURL("ChinaCompanyIp"), Behavior: "ipcidr"},
			{ID: "acl4ssr/LocalAreaNetwork", Name: "LocalAreaNetwork", Category: "Direct", Tags: []string{"局域网", "lan"}, Description: "局域网地址直连", URL: makeURL("LocalAreaNetwork"), Behavior: "ipcidr"},
			{ID: "acl4ssr/UnBan", Name: "UnBan", Category: "Direct", Tags: []string{"解封", "直连"}, Description: "解封规则（被误杀的域名）", URL: makeURL("UnBan"), Behavior: "domain"},
			{ID: "acl4ssr/GoogleCN", Name: "GoogleCN", Category: "Direct", Tags: []string{"google", "国内", "直连"}, Description: "Google 国内服务直连", URL: makeURL("GoogleCN"), Behavior: "domain"},

			{ID: "acl4ssr/Apple", Name: "Apple", Category: "Apple", Tags: []string{"apple", "苹果"}, Description: "Apple 服务", URL: makeURL("Apple"), Behavior: "domain"},
			{ID: "acl4ssr/Microsoft", Name: "Microsoft", Category: "Microsoft", Tags: []string{"microsoft", "微软"}, Description: "Microsoft 服务", URL: makeURL("Microsoft"), Behavior: "domain"},
			{ID: "acl4ssr/Bing", Name: "Bing", Category: "Search", Tags: []string{"bing", "搜索", "microsoft"}, Description: "Bing 搜索", URL: makeURL("Bing"), Behavior: "domain"},

			{ID: "acl4ssr/Netflix", Name: "Netflix", Category: "Streaming", Tags: []string{"netflix", "流媒体"}, Description: "Netflix", URL: makeURL("Netflix"), Behavior: "domain"},
			{ID: "acl4ssr/YouTube", Name: "YouTube", Category: "Streaming", Tags: []string{"youtube", "流媒体"}, Description: "YouTube", URL: makeURL("YouTube"), Behavior: "domain"},
			{ID: "acl4ssr/ChinaMedia", Name: "ChinaMedia", Category: "Streaming", Tags: []string{"国内媒体", "流媒体"}, Description: "中国大陆媒体", URL: makeURL("ChinaMedia"), Behavior: "domain"},

			{ID: "acl4ssr/Telegram", Name: "Telegram", Category: "Social", Tags: []string{"telegram", "电报"}, Description: "Telegram", URL: makeURL("Telegram"), Behavior: "domain"},
			{ID: "acl4ssr/Download", Name: "Download", Category: "Download", Tags: []string{"下载", "download"}, Description: "下载工具", URL: makeURL("Download"), Behavior: "domain"},
			{ID: "acl4ssr/OneDrive", Name: "OneDrive", Category: "Microsoft", Tags: []string{"onedrive", "云盘"}, Description: "OneDrive", URL: makeURL("OneDrive"), Behavior: "domain"},
			{ID: "acl4ssr/Xbox", Name: "Xbox", Category: "Gaming", Tags: []string{"xbox", "游戏"}, Description: "Xbox", URL: makeURL("Xbox"), Behavior: "domain"},
		},
	}
}