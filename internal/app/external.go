package app

import (
	"log"
	"net/url"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const openExternalPrefix = "dsh-go:open:"

// externalLinkJS sends http(s)/mailto clicks and window.open to the shell so
// they open in the system browser instead of a new WebView. Same-origin
// in-app navigation is left alone.
var externalLinkJS = `(function(){
  if (window.__dshGoExtLinks) return;
  window.__dshGoExtLinks = true;
  function post(url){
    var msg = "dsh-go:open:" + url;
    try {
      if (window.chrome && window.chrome.webview && typeof window.chrome.webview.postMessage === "function") {
        window.chrome.webview.postMessage(msg);
        return;
      }
    } catch (e) {}
    try { window.webkit.messageHandlers.external.postMessage(msg); } catch (e2) {}
    try {
      if (window._wails && typeof window._wails.invoke === "function") {
        window._wails.invoke(msg);
      }
    } catch (e3) {}
  }
  function resolve(href){
    try { return new URL(href, location.href).href; } catch (e) { return ""; }
  }
  function isExternal(href){
    try {
      var u = new URL(href, location.href);
      if (u.protocol === "mailto:") return true;
      if (u.protocol !== "http:" && u.protocol !== "https:") return false;
      return u.host !== location.host;
    } catch (e) { return false; }
  }
  function openExt(href){
    var url = resolve(href);
    if (!url) return false;
    post(url);
    return true;
  }
  document.addEventListener("click", function(ev){
    var a = ev.target && ev.target.closest ? ev.target.closest("a[href]") : null;
    if (!a) return;
    var href = a.getAttribute("href") || "";
    if (!href || href.charAt(0) === "#") return;
    var blank = (a.target || "").toLowerCase() === "_blank";
    var modified = ev.metaKey || ev.ctrlKey || ev.shiftKey || ev.altKey;
    if (blank || modified || isExternal(href)) {
      if (openExt(href)) {
        ev.preventDefault();
        ev.stopPropagation();
      }
    }
  }, true);
  document.addEventListener("auxclick", function(ev){
    if (ev.button !== 1) return;
    var a = ev.target && ev.target.closest ? ev.target.closest("a[href]") : null;
    if (!a) return;
    if (openExt(a.getAttribute("href") || "")) {
      ev.preventDefault();
      ev.stopPropagation();
    }
  }, true);
  var nativeOpen = window.open;
  window.open = function(url){
    if (url && openExt(String(url))) return null;
    if (typeof nativeOpen === "function") return nativeOpen.apply(this, arguments);
    return null;
  };
})();`

func parseOpenExternalMessage(message string) (string, bool) {
	if !strings.HasPrefix(message, openExternalPrefix) {
		return "", false
	}
	raw := strings.TrimSpace(strings.TrimPrefix(message, openExternalPrefix))
	if !shouldOpenExternally(raw) {
		return "", false
	}
	return raw, true
}

func shouldOpenExternally(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https", "mailto":
		return true
	default:
		return false
	}
}

func handleOpenExternalMessage(app *application.App, message string) {
	raw, ok := parseOpenExternalMessage(message)
	if !ok || app == nil {
		return
	}
	if err := app.Browser.OpenURL(raw); err != nil {
		log.Printf("open external: %v", err)
	}
}

func handleRawWebviewMessage(app *application.App, message string) {
	if text, ok := parseBootFailMessage(message); ok {
		if app != nil {
			app.Event.Emit(bootFailEvent, text)
		}
		return
	}
	handleOpenExternalMessage(app, message)
}

func harnessInitJS() string {
	return themeWatchJS + "\n" + externalLinkJS + "\n" + bootFailWatchJS
}
