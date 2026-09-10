package app

import (
	"os"
	"strings"
	"sync"
)

const (
	localeZH        = "zh"
	localeEN        = "en"
	localePrefEvent = "dsh-go:locale"
)

var (
	localeOnce sync.Once
	localePref string
)

func parseLocalePreference(raw string) string {
	pref := strings.ToLower(parseYAMLPreference(raw, "locale"))
	switch pref {
	case "en", "en-us", "en-gb":
		return localeEN
	case "zh", "zh-cn", "zh-hans", "zh-tw", "zh-hant":
		return localeZH
	default:
		return ""
	}
}

func readLocalePreference() string {
	b, err := os.ReadFile(settingsYAMLPath())
	if err != nil {
		return localeZH
	}
	if pref := parseLocalePreference(string(b)); pref != "" {
		return pref
	}
	return localeZH
}

// loadStartupLocale reads Harness locale.preference once. Later UI language
// changes are picked up the next time the shell starts.
func loadStartupLocale() string {
	localeOnce.Do(func() {
		localePref = readLocalePreference()
	})
	return localePref
}

func prepPageURL() string {
	if loadStartupLocale() == localeEN {
		return "/?lang=en"
	}
	return "/"
}

type uiText struct {
	Downloading      string
	Installing       string
	SelectingSource  string
	InstallingClient string
	UpdatingRuntime  string
	OfferApp         string
	OfferBoth        string
	OfferDSH         string
	UpdateNow        string
	Client           string
	Runtime          string
	NPMMissing       string
	NPMFailed        string
	DisableRestart   string
	RecoverFail      string
}

var (
	uiZH = uiText{
		Downloading:      "正在下载…",
		Installing:       "正在安装…",
		SelectingSource:  "正在选择下载源…",
		InstallingClient: "正在安装客户端…",
		UpdatingRuntime:  "正在更新运行时到 %s…",
		OfferApp:         "客户端有新版本，更新后会重启。",
		OfferBoth:        "将一并更新客户端和运行时。",
		OfferDSH:         "运行时有新版本，更新后会重新启动 Harness。",
		UpdateNow:        "立即更新",
		Client:           "客户端",
		Runtime:          "运行时",
		NPMMissing:       "未找到 npm，无法更新全局 dsh",
		NPMFailed:        "npm i -g 失败: %s",
		DisableRestart:   "禁用并重启",
		RecoverFail:      "Harness 插件加载失败",
	}
	uiEN = uiText{
		Downloading:      "Downloading…",
		Installing:       "Installing…",
		SelectingSource:  "Choosing a download source…",
		InstallingClient: "Installing the client…",
		UpdatingRuntime:  "Updating runtime to %s…",
		OfferApp:         "A new client version is available. The app will restart after updating.",
		OfferBoth:        "The client and runtime will be updated together.",
		OfferDSH:         "A new runtime version is available. Harness will restart after updating.",
		UpdateNow:        "Update now",
		Client:           "Client",
		Runtime:          "Runtime",
		NPMMissing:       "npm was not found; cannot update global dsh",
		NPMFailed:        "npm i -g failed: %s",
		DisableRestart:   "Disable and restart",
		RecoverFail:      "Harness failed to load plugins",
	}
)

func currentUI() uiText {
	if loadStartupLocale() == localeEN {
		return uiEN
	}
	return uiZH
}
