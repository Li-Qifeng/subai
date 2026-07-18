package rulerepo

// blackmatrix7Repo returns the index for blackmatrix7/ios_rule_script.
// The most comprehensive rule repository with 600+ rule sets.
// Rules are at: https://raw.githubusercontent.com/blackmatrix7/ios_rule_script/master/rule/Clash/{Name}/{Name}.yaml
func blackmatrix7Repo() Repo {
	base := "https://raw.githubusercontent.com/blackmatrix7/ios_rule_script/master/rule/Clash"
	makeURL := func(name string) string {
		return base + "/" + name + "/" + name + ".yaml"
	}

	return Repo{
		Name:        RepoBlackmatrix7,
		DisplayName: "blackmatrix7/ios_rule_script",
		Description: "最全面的分流规则仓库，600+ 规则集覆盖广告、AI、流媒体、社交、游戏等全品类，每日自动更新",
		BaseURL:     base,
		Rules: []RuleMeta{
			// === 去广告 / 隐私 ===
			{ID: "blackmatrix7/Advertising", Name: "Advertising", Category: "Ads", Tags: []string{"广告", "ads", "reject"}, Description: "综合广告拦截规则", URL: makeURL("Advertising"), Behavior: "classical"},
			{ID: "blackmatrix7/AdvertisingLite", Name: "AdvertisingLite", Category: "Ads", Tags: []string{"广告精简", "ads"}, Description: "广告拦截精简版", URL: makeURL("AdvertisingLite"), Behavior: "classical"},
			{ID: "blackmatrix7/AdvertisingTest", Name: "AdvertisingTest", Category: "Ads", Tags: []string{"广告测试", "ads"}, Description: "广告拦截测试版（更激进）", URL: makeURL("AdvertisingTest"), Behavior: "classical"},
			{ID: "blackmatrix7/Hijacking", Name: "Hijacking", Category: "Ads", Tags: []string{"反劫持", "dns"}, Description: "DNS 反劫持", URL: makeURL("Hijacking"), Behavior: "domain"},
			{ID: "blackmatrix7/Privacy", Name: "Privacy", Category: "Ads", Tags: []string{"隐私", "privacy"}, Description: "隐私保护", URL: makeURL("Privacy"), Behavior: "domain"},
			{ID: "blackmatrix7/ZhihuAds", Name: "ZhihuAds", Category: "Ads", Tags: []string{"知乎", "广告"}, Description: "知乎广告拦截", URL: makeURL("ZhihuAds"), Behavior: "domain"},

			// === AI ===
			{ID: "blackmatrix7/OpenAI", Name: "OpenAI", Category: "AI", Tags: []string{"ai", "chatgpt", "openai"}, Description: "OpenAI / ChatGPT", URL: makeURL("OpenAI"), Behavior: "domain"},
			{ID: "blackmatrix7/Copilot", Name: "Copilot", Category: "AI", Tags: []string{"ai", "copilot", "github", "microsoft"}, Description: "GitHub Copilot", URL: makeURL("Copilot"), Behavior: "domain"},
			{ID: "blackmatrix7/Gemini", Name: "Gemini", Category: "AI", Tags: []string{"ai", "gemini", "google"}, Description: "Google Gemini", URL: makeURL("Gemini"), Behavior: "domain"},
			{ID: "blackmatrix7/Claude", Name: "Claude", Category: "AI", Tags: []string{"ai", "claude", "anthropic"}, Description: "Anthropic Claude", URL: makeURL("Claude"), Behavior: "domain"},
			{ID: "blackmatrix7/Perplexity", Name: "Perplexity", Category: "AI", Tags: []string{"ai", "perplexity", "search"}, Description: "Perplexity AI", URL: makeURL("Perplexity"), Behavior: "domain"},

			// === 流媒体 ===
			{ID: "blackmatrix7/Netflix", Name: "Netflix", Category: "Streaming", Tags: []string{"netflix", "流媒体"}, Description: "Netflix", URL: makeURL("Netflix"), Behavior: "domain"},
			{ID: "blackmatrix7/YouTube", Name: "YouTube", Category: "Streaming", Tags: []string{"youtube", "流媒体", "google"}, Description: "YouTube", URL: makeURL("YouTube"), Behavior: "domain"},
			{ID: "blackmatrix7/Disney", Name: "Disney", Category: "Streaming", Tags: []string{"disney", "流媒体"}, Description: "Disney+", URL: makeURL("Disney"), Behavior: "domain"},
			{ID: "blackmatrix7/HBO", Name: "HBO", Category: "Streaming", Tags: []string{"hbo", "流媒体"}, Description: "HBO / HBO Max", URL: makeURL("HBO"), Behavior: "domain"},
			{ID: "blackmatrix7/AmazonPrimeVideo", Name: "AmazonPrimeVideo", Category: "Streaming", Tags: []string{"amazon", "prime", "流媒体"}, Description: "Amazon Prime Video", URL: makeURL("AmazonPrimeVideo"), Behavior: "domain"},
			{ID: "blackmatrix7/Spotify", Name: "Spotify", Category: "Streaming", Tags: []string{"spotify", "音乐"}, Description: "Spotify", URL: makeURL("Spotify"), Behavior: "domain"},
			{ID: "blackmatrix7/AppleMusic", Name: "AppleMusic", Category: "Streaming", Tags: []string{"apple", "music"}, Description: "Apple Music", URL: makeURL("AppleMusic"), Behavior: "domain"},
			{ID: "blackmatrix7/TikTok", Name: "TikTok", Category: "Streaming", Tags: []string{"tiktok", "短视频"}, Description: "TikTok", URL: makeURL("TikTok"), Behavior: "domain"},
			{ID: "blackmatrix7/Bilibili", Name: "Bilibili", Category: "Streaming", Tags: []string{"bilibili", "b站"}, Description: "Bilibili", URL: makeURL("Bilibili"), Behavior: "domain"},
			{ID: "blackmatrix7/YouTubeMusic", Name: "YouTubeMusic", Category: "Streaming", Tags: []string{"youtube", "music", "google"}, Description: "YouTube Music", URL: makeURL("YouTubeMusic"), Behavior: "domain"},
			{ID: "blackmatrix7/GlobalMedia", Name: "GlobalMedia", Category: "Streaming", Tags: []string{"全球媒体", "streaming"}, Description: "全球综合媒体", URL: makeURL("GlobalMedia"), Behavior: "domain"},
			{ID: "blackmatrix7/ChinaMedia", Name: "ChinaMedia", Category: "Streaming", Tags: []string{"国内媒体", "streaming"}, Description: "中国大陆媒体合集", URL: makeURL("ChinaMedia"), Behavior: "domain"},

			// === 社交 / 通讯 ===
			{ID: "blackmatrix7/Telegram", Name: "Telegram", Category: "Social", Tags: []string{"telegram", "电报"}, Description: "Telegram", URL: makeURL("Telegram"), Behavior: "domain"},
			{ID: "blackmatrix7/Twitter", Name: "Twitter", Category: "Social", Tags: []string{"twitter", "x", "社交"}, Description: "Twitter / X", URL: makeURL("Twitter"), Behavior: "domain"},
			{ID: "blackmatrix7/Instagram", Name: "Instagram", Category: "Social", Tags: []string{"instagram", "社交"}, Description: "Instagram", URL: makeURL("Instagram"), Behavior: "domain"},
			{ID: "blackmatrix7/Whatsapp", Name: "Whatsapp", Category: "Social", Tags: []string{"whatsapp", "社交"}, Description: "WhatsApp", URL: makeURL("Whatsapp"), Behavior: "domain"},
			{ID: "blackmatrix7/Signal", Name: "Signal", Category: "Social", Tags: []string{"signal", "通讯"}, Description: "Signal", URL: makeURL("Signal"), Behavior: "domain"},
			{ID: "blackmatrix7/Discord", Name: "Discord", Category: "Social", Tags: []string{"discord", "社交"}, Description: "Discord", URL: makeURL("Discord"), Behavior: "domain"},
			{ID: "blackmatrix7/TelegramUS", Name: "TelegramUS", Category: "Social", Tags: []string{"telegram", "us", "美国"}, Description: "Telegram 美国节点", URL: makeURL("TelegramUS"), Behavior: "domain"},
			{ID: "blackmatrix7/TelegramSG", Name: "TelegramSG", Category: "Social", Tags: []string{"telegram", "sg", "新加坡"}, Description: "Telegram 新加坡节点", URL: makeURL("TelegramSG"), Behavior: "domain"},
			{ID: "blackmatrix7/Clubhouse", Name: "Clubhouse", Category: "Social", Tags: []string{"clubhouse", "社交"}, Description: "Clubhouse", URL: makeURL("Clubhouse"), Behavior: "domain"},

			// === 搜索引擎 ===
			{ID: "blackmatrix7/Google", Name: "Google", Category: "Search", Tags: []string{"google", "搜索"}, Description: "Google 服务", URL: makeURL("Google"), Behavior: "domain"},
			{ID: "blackmatrix7/GoogleSearch", Name: "GoogleSearch", Category: "Search", Tags: []string{"google", "搜索"}, Description: "Google 搜索", URL: makeURL("GoogleSearch"), Behavior: "domain"},
			{ID: "blackmatrix7/Bing", Name: "Bing", Category: "Search", Tags: []string{"bing", "搜索", "microsoft"}, Description: "Bing 搜索", URL: makeURL("Bing"), Behavior: "domain"},
			{ID: "blackmatrix7/Duckduckgo", Name: "Duckduckgo", Category: "Search", Tags: []string{"duckduckgo", "搜索"}, Description: "DuckDuckGo", URL: makeURL("Duckduckgo"), Behavior: "domain"},
			{ID: "blackmatrix7/Yandex", Name: "Yandex", Category: "Search", Tags: []string{"yandex", "搜索"}, Description: "Yandex", URL: makeURL("Yandex"), Behavior: "domain"},

			// === 微软 ===
			{ID: "blackmatrix7/Microsoft", Name: "Microsoft", Category: "Microsoft", Tags: []string{"microsoft", "微软"}, Description: "Microsoft 综合服务", URL: makeURL("Microsoft"), Behavior: "domain"},
			{ID: "blackmatrix7/OneDrive", Name: "OneDrive", Category: "Microsoft", Tags: []string{"onedrive", "云盘", "microsoft"}, Description: "OneDrive", URL: makeURL("OneDrive"), Behavior: "domain"},
			{ID: "blackmatrix7/Teams", Name: "Teams", Category: "Microsoft", Tags: []string{"teams", "microsoft"}, Description: "Microsoft Teams", URL: makeURL("Teams"), Behavior: "domain"},
			{ID: "blackmatrix7/Windows", Name: "Windows", Category: "Microsoft", Tags: []string{"windows", "microsoft"}, Description: "Windows 更新等服务", URL: makeURL("Windows"), Behavior: "domain"},

			// === 苹果 ===
			{ID: "blackmatrix7/Apple", Name: "Apple", Category: "Apple", Tags: []string{"apple", "苹果"}, Description: "Apple 综合服务", URL: makeURL("Apple"), Behavior: "domain"},
			{ID: "blackmatrix7/AppleNews", Name: "AppleNews", Category: "Apple", Tags: []string{"apple", "news"}, Description: "Apple News", URL: makeURL("AppleNews"), Behavior: "domain"},
			{ID: "blackmatrix7/AppleTV", Name: "AppleTV", Category: "Apple", Tags: []string{"apple", "tv"}, Description: "Apple TV", URL: makeURL("AppleTV"), Behavior: "domain"},
			{ID: "blackmatrix7/TestFlight", Name: "TestFlight", Category: "Apple", Tags: []string{"apple", "testflight"}, Description: "TestFlight", URL: makeURL("TestFlight"), Behavior: "domain"},
			{ID: "blackmatrix7/iCloud", Name: "iCloud", Category: "Apple", Tags: []string{"apple", "icloud", "云盘"}, Description: "iCloud", URL: makeURL("iCloud"), Behavior: "domain"},

			// === 游戏 ===
			{ID: "blackmatrix7/Steam", Name: "Steam", Category: "Gaming", Tags: []string{"steam", "游戏"}, Description: "Steam", URL: makeURL("Steam"), Behavior: "domain"},
			{ID: "blackmatrix7/Epic", Name: "Epic", Category: "Gaming", Tags: []string{"epic", "游戏"}, Description: "Epic Games", URL: makeURL("Epic"), Behavior: "domain"},
			{ID: "blackmatrix7/Xbox", Name: "Xbox", Category: "Gaming", Tags: []string{"xbox", "游戏", "microsoft"}, Description: "Xbox", URL: makeURL("Xbox"), Behavior: "domain"},
			{ID: "blackmatrix7/PlayStation", Name: "PlayStation", Category: "Gaming", Tags: []string{"playstation", "ps", "游戏"}, Description: "PlayStation Network", URL: makeURL("PlayStation"), Behavior: "domain"},
			{ID: "blackmatrix7/Nintendo", Name: "Nintendo", Category: "Gaming", Tags: []string{"nintendo", "任天堂", "游戏"}, Description: "Nintendo", URL: makeURL("Nintendo"), Behavior: "domain"},
			{ID: "blackmatrix7/Riot", Name: "Riot", Category: "Gaming", Tags: []string{"riot", "英雄联盟", "游戏"}, Description: "Riot Games（英雄联盟/Valorant）", URL: makeURL("Riot"), Behavior: "domain"},
			{ID: "blackmatrix7/Game", Name: "Game", Category: "Gaming", Tags: []string{"游戏", "综合"}, Description: "游戏综合", URL: makeURL("Game"), Behavior: "domain"},

			// === 代理 / 直连 ===
			{ID: "blackmatrix7/Proxy", Name: "Proxy", Category: "Proxy", Tags: []string{"代理", "proxy"}, Description: "代理综合规则", URL: makeURL("Proxy"), Behavior: "domain"},
			{ID: "blackmatrix7/GFW", Name: "GFW", Category: "Proxy", Tags: []string{"gfw", "墙"}, Description: "GFW 规则", URL: makeURL("GFW"), Behavior: "domain"},
			{ID: "blackmatrix7/China", Name: "China", Category: "Direct", Tags: []string{"国内", "china", "直连"}, Description: "中国大陆域名合集", URL: makeURL("China"), Behavior: "domain"},
			{ID: "blackmatrix7/ChinaIPs", Name: "ChinaIPs", Category: "Direct", Tags: []string{"国内", "ip", "直连"}, Description: "中国大陆 IP", URL: makeURL("ChinaIPs"), Behavior: "ipcidr"},
			{ID: "blackmatrix7/Lan", Name: "Lan", Category: "Direct", Tags: []string{"局域网", "lan"}, Description: "本地局域网地址", URL: makeURL("Lan"), Behavior: "ipcidr"},
			{ID: "blackmatrix7/Direct", Name: "Direct", Category: "Direct", Tags: []string{"直连", "direct"}, Description: "直连综合", URL: makeURL("Direct"), Behavior: "domain"},

			// === 国内应用 (Mainland China) ===
			{ID: "blackmatrix7/Alibaba", Name: "Alibaba", Category: "Mainland", Tags: []string{"阿里巴巴", "电商"}, Description: "阿里巴巴", URL: makeURL("Alibaba"), Behavior: "domain"},
			{ID: "blackmatrix7/AliPay", Name: "AliPay", Category: "Mainland", Tags: []string{"支付宝", "支付"}, Description: "支付宝", URL: makeURL("AliPay"), Behavior: "domain"},
			{ID: "blackmatrix7/Baidu", Name: "Baidu", Category: "Mainland", Tags: []string{"百度", "搜索"}, Description: "百度", URL: makeURL("Baidu"), Behavior: "domain"},
			{ID: "blackmatrix7/BaiDuTieBa", Name: "BaiDuTieBa", Category: "Mainland", Tags: []string{"百度贴吧", "社区"}, Description: "百度贴吧", URL: makeURL("BaiDuTieBa"), Behavior: "domain"},
			{ID: "blackmatrix7/ByteDance", Name: "ByteDance", Category: "Mainland", Tags: []string{"字节跳动", "综合"}, Description: "字节跳动", URL: makeURL("ByteDance"), Behavior: "domain"},
			{ID: "blackmatrix7/CCTV", Name: "CCTV", Category: "Mainland", Tags: []string{"央视", "电视"}, Description: "CCTV", URL: makeURL("CCTV"), Behavior: "domain"},
			{ID: "blackmatrix7/ChinaMobile", Name: "ChinaMobile", Category: "Mainland", Tags: []string{"中国移动", "运营商"}, Description: "中国移动", URL: makeURL("ChinaMobile"), Behavior: "domain"},
			{ID: "blackmatrix7/ChinaTelecom", Name: "ChinaTelecom", Category: "Mainland", Tags: []string{"中国电信", "运营商"}, Description: "中国电信", URL: makeURL("ChinaTelecom"), Behavior: "domain"},
			{ID: "blackmatrix7/ChinaUnicom", Name: "ChinaUnicom", Category: "Mainland", Tags: []string{"中国联通", "运营商"}, Description: "中国联通", URL: makeURL("ChinaUnicom"), Behavior: "domain"},
			{ID: "blackmatrix7/Coolapk", Name: "Coolapk", Category: "Mainland", Tags: []string{"酷安", "社区"}, Description: "酷安", URL: makeURL("Coolapk"), Behavior: "domain"},
			{ID: "blackmatrix7/CSDN", Name: "CSDN", Category: "Mainland", Tags: []string{"csdn", "开发"}, Description: "CSDN", URL: makeURL("CSDN"), Behavior: "domain"},
			{ID: "blackmatrix7/DiDi", Name: "DiDi", Category: "Mainland", Tags: []string{"滴滴", "出行"}, Description: "滴滴出行", URL: makeURL("DiDi"), Behavior: "domain"},
			{ID: "blackmatrix7/DingTalk", Name: "DingTalk", Category: "Mainland", Tags: []string{"钉钉", "办公"}, Description: "钉钉", URL: makeURL("DingTalk"), Behavior: "domain"},
			{ID: "blackmatrix7/DouBan", Name: "DouBan", Category: "Mainland", Tags: []string{"豆瓣", "社区"}, Description: "豆瓣", URL: makeURL("DouBan"), Behavior: "domain"},
			{ID: "blackmatrix7/DouYin", Name: "DouYin", Category: "Mainland", Tags: []string{"抖音", "短视频"}, Description: "抖音", URL: makeURL("DouYin"), Behavior: "domain"},
			{ID: "blackmatrix7/Douyu", Name: "Douyu", Category: "Mainland", Tags: []string{"斗鱼", "直播"}, Description: "斗鱼", URL: makeURL("Douyu"), Behavior: "domain"},
			{ID: "blackmatrix7/Eleme", Name: "Eleme", Category: "Mainland", Tags: []string{"饿了么", "外卖"}, Description: "饿了么", URL: makeURL("Eleme"), Behavior: "domain"},
			{ID: "blackmatrix7/GaoDe", Name: "GaoDe", Category: "Mainland", Tags: []string{"高德", "地图"}, Description: "高德", URL: makeURL("GaoDe"), Behavior: "domain"},
			{ID: "blackmatrix7/Gitee", Name: "Gitee", Category: "Mainland", Tags: []string{"码云", "开发"}, Description: "Gitee 码云", URL: makeURL("Gitee"), Behavior: "domain"},
			{ID: "blackmatrix7/Himalaya", Name: "Himalaya", Category: "Mainland", Tags: []string{"喜马拉雅", "音频"}, Description: "喜马拉雅", URL: makeURL("Himalaya"), Behavior: "domain"},
			{ID: "blackmatrix7/HoYoverse", Name: "HoYoverse", Category: "Mainland", Tags: []string{"米哈游", "原神", "游戏"}, Description: "米哈游 HoYoverse（原神/星穹铁道）", URL: makeURL("Game/HoYoverse"), Behavior: "domain"},
			{ID: "blackmatrix7/Huawei", Name: "Huawei", Category: "Mainland", Tags: []string{"华为", "手机"}, Description: "华为", URL: makeURL("Huawei"), Behavior: "domain"},
			{ID: "blackmatrix7/Hupu", Name: "Hupu", Category: "Mainland", Tags: []string{"虎扑", "社区"}, Description: "虎扑", URL: makeURL("Hupu"), Behavior: "domain"},
			{ID: "blackmatrix7/HuYa", Name: "HuYa", Category: "Mainland", Tags: []string{"虎牙", "直播"}, Description: "虎牙", URL: makeURL("HuYa"), Behavior: "domain"},
			{ID: "blackmatrix7/JingDong", Name: "JingDong", Category: "Mainland", Tags: []string{"京东", "电商"}, Description: "京东", URL: makeURL("JingDong"), Behavior: "domain"},
			{ID: "blackmatrix7/KuaiShou", Name: "KuaiShou", Category: "Mainland", Tags: []string{"快手", "短视频"}, Description: "快手", URL: makeURL("KuaiShou"), Behavior: "domain"},
			{ID: "blackmatrix7/KugouKuwo", Name: "KugouKuwo", Category: "Mainland", Tags: []string{"酷狗", "酷我", "音乐"}, Description: "酷狗/酷我", URL: makeURL("KugouKuwo"), Behavior: "domain"},
			{ID: "blackmatrix7/MeiTuan", Name: "MeiTuan", Category: "Mainland", Tags: []string{"美团", "外卖"}, Description: "美团", URL: makeURL("MeiTuan"), Behavior: "domain"},
			{ID: "blackmatrix7/MeiZu", Name: "MeiZu", Category: "Mainland", Tags: []string{"魅族", "手机"}, Description: "魅族", URL: makeURL("MeiZu"), Behavior: "domain"},
			{ID: "blackmatrix7/NetEase", Name: "NetEase", Category: "Mainland", Tags: []string{"网易", "门户"}, Description: "网易", URL: makeURL("NetEase"), Behavior: "domain"},
			{ID: "blackmatrix7/NetEaseMusic", Name: "NetEaseMusic", Category: "Mainland", Tags: []string{"网易云音乐", "音乐"}, Description: "网易云音乐", URL: makeURL("NetEaseMusic"), Behavior: "domain"},
			{ID: "blackmatrix7/OPPO", Name: "OPPO", Category: "Mainland", Tags: []string{"oppo", "手机"}, Description: "OPPO", URL: makeURL("OPPO"), Behavior: "domain"},
			{ID: "blackmatrix7/Pinduoduo", Name: "Pinduoduo", Category: "Mainland", Tags: []string{"拼多多", "电商"}, Description: "拼多多", URL: makeURL("Pinduoduo"), Behavior: "domain"},
			{ID: "blackmatrix7/Sina", Name: "Sina", Category: "Mainland", Tags: []string{"新浪", "门户"}, Description: "新浪", URL: makeURL("Sina"), Behavior: "domain"},
			{ID: "blackmatrix7/Sohu", Name: "Sohu", Category: "Mainland", Tags: []string{"搜狐", "门户"}, Description: "搜狐", URL: makeURL("Sohu"), Behavior: "domain"},
			{ID: "blackmatrix7/SuNing", Name: "SuNing", Category: "Mainland", Tags: []string{"苏宁", "电商"}, Description: "苏宁", URL: makeURL("SuNing"), Behavior: "domain"},
			{ID: "blackmatrix7/Tencent", Name: "Tencent", Category: "Mainland", Tags: []string{"腾讯", "综合"}, Description: "腾讯", URL: makeURL("Tencent"), Behavior: "domain"},
			{ID: "blackmatrix7/TencentVideo", Name: "TencentVideo", Category: "Mainland", Tags: []string{"腾讯视频", "视频"}, Description: "腾讯视频", URL: makeURL("TencentVideo"), Behavior: "domain"},
			{ID: "blackmatrix7/Vivo", Name: "Vivo", Category: "Mainland", Tags: []string{"vivo", "手机"}, Description: "Vivo", URL: makeURL("Vivo"), Behavior: "domain"},
			{ID: "blackmatrix7/WeChat", Name: "WeChat", Category: "Mainland", Tags: []string{"微信", "社交"}, Description: "微信", URL: makeURL("WeChat"), Behavior: "domain"},
			{ID: "blackmatrix7/Weibo", Name: "Weibo", Category: "Mainland", Tags: []string{"微博", "社交"}, Description: "微博", URL: makeURL("Weibo"), Behavior: "domain"},
			{ID: "blackmatrix7/XianYu", Name: "XianYu", Category: "Mainland", Tags: []string{"闲鱼", "电商"}, Description: "闲鱼", URL: makeURL("XianYu"), Behavior: "domain"},
			{ID: "blackmatrix7/XiaoHongShu", Name: "XiaoHongShu", Category: "Mainland", Tags: []string{"小红书", "社区"}, Description: "小红书", URL: makeURL("XiaoHongShu"), Behavior: "domain"},
			{ID: "blackmatrix7/XiaoMi", Name: "XiaoMi", Category: "Mainland", Tags: []string{"小米", "手机"}, Description: "小米", URL: makeURL("XiaoMi"), Behavior: "domain"},
			{ID: "blackmatrix7/XieCheng", Name: "XieCheng", Category: "Mainland", Tags: []string{"携程", "旅游"}, Description: "携程", URL: makeURL("XieCheng"), Behavior: "domain"},
			{ID: "blackmatrix7/Xunlei", Name: "Xunlei", Category: "Mainland", Tags: []string{"迅雷", "下载"}, Description: "迅雷", URL: makeURL("Xunlei"), Behavior: "domain"},
			{ID: "blackmatrix7/Youku", Name: "Youku", Category: "Mainland", Tags: []string{"优酷", "视频"}, Description: "优酷", URL: makeURL("Youku"), Behavior: "domain"},
			{ID: "blackmatrix7/Zhihu", Name: "Zhihu", Category: "Mainland", Tags: []string{"知乎", "社区"}, Description: "知乎", URL: makeURL("Zhihu"), Behavior: "domain"},
			{ID: "blackmatrix7/iQIYI", Name: "iQIYI", Category: "Mainland", Tags: []string{"爱奇艺", "视频"}, Description: "爱奇艺", URL: makeURL("iQIYI"), Behavior: "domain"},
			{ID: "blackmatrix7/360", Name: "360", Category: "Mainland", Tags: []string{"360", "安全"}, Description: "奇虎360", URL: makeURL("360"), Behavior: "domain"},

			// === 云服务 / 开发 ===
			{ID: "blackmatrix7/GitHub", Name: "GitHub", Category: "Dev", Tags: []string{"github", "开发"}, Description: "GitHub", URL: makeURL("GitHub"), Behavior: "domain"},
			{ID: "blackmatrix7/Docker", Name: "Docker", Category: "Dev", Tags: []string{"docker", "开发"}, Description: "Docker", URL: makeURL("Docker"), Behavior: "domain"},
			{ID: "blackmatrix7/Cloudflare", Name: "Cloudflare", Category: "Dev", Tags: []string{"cloudflare", "cdn"}, Description: "Cloudflare", URL: makeURL("Cloudflare"), Behavior: "domain"},
			{ID: "blackmatrix7/Developer", Name: "Developer", Category: "Dev", Tags: []string{"开发者", "开发"}, Description: "开发者相关合集", URL: makeURL("Developer"), Behavior: "domain"},
			{ID: "blackmatrix7/AmazonCloud", Name: "AmazonCloud", Category: "Dev", Tags: []string{"aws", "amazon", "云"}, Description: "Amazon Web Services", URL: makeURL("Cloud/AmazonCloud"), Behavior: "ipcidr"},
			{ID: "blackmatrix7/CloudCN", Name: "CloudCN", Category: "Dev", Tags: []string{"国内云", "云计算"}, Description: "国内云计算合集", URL: makeURL("Cloud/CloudCN"), Behavior: "ipcidr"},
			{ID: "blackmatrix7/CloudGlobal", Name: "CloudGlobal", Category: "Dev", Tags: []string{"全球云", "云计算"}, Description: "全球云计算合集", URL: makeURL("Cloud/CloudGlobal"), Behavior: "ipcidr"},

			// === 下载 ===
			{ID: "blackmatrix7/Download", Name: "Download", Category: "Download", Tags: []string{"下载", "download"}, Description: "下载工具合集", URL: makeURL("Download"), Behavior: "domain"},
			{ID: "blackmatrix7/PrivateTracker", Name: "PrivateTracker", Category: "Download", Tags: []string{"pt", "下载", "tracker"}, Description: "PT 下载", URL: makeURL("PrivateTracker"), Behavior: "domain"},
		},
	}
}