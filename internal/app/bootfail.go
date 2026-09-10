package app

import "strings"

// Boot-fail watch recognizes dsh's official "Failed to load plugins" page.

const (
	bootFailPrefix = "dsh-go:boot-fail:"
	bootFailEvent  = "dsh-go:boot-fail"
)

// bootFailWatchJS observes the official web boot card. Same injection
// constraints as themeWatchJS: nil-guarded observers, no timers, retry
// once the document exists (WebView2 runs this before HTML is parsed).
var bootFailWatchJS = `(function(){
  if (window.__dshGoBootFail) return;
  window.__dshGoBootFail = true;
  var last = "";
  function post(text){
    var msg = "dsh-go:boot-fail:" + text;
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
  function scan(){
    var boot = document.querySelector("[data-dsh-boot]");
    if (!boot) return;
    var text = boot.innerText || boot.textContent || "";
    if (text.indexOf("Failed to load plugins") < 0) return;
    if (text === last) return;
    last = text;
    post(text);
  }
  function bind(el, opts, fn){
    if (!el || el.__dshGoBootObs) return false;
    el.__dshGoBootObs = true;
    new MutationObserver(fn).observe(el, opts);
    return true;
  }
  function start(){
    bind(document.documentElement, {childList:true, subtree:true}, scan);
    bind(document.body, {childList:true, subtree:true}, scan);
    scan();
    return !!document.body;
  }
  if (!start()) {
    document.addEventListener("readystatechange", start);
    document.addEventListener("DOMContentLoaded", start);
  }
})();`

func parseBootFailMessage(message string) (string, bool) {
	if !strings.HasPrefix(message, bootFailPrefix) {
		return "", false
	}
	text := strings.TrimSpace(strings.TrimPrefix(message, bootFailPrefix))
	if text == "" || !strings.Contains(text, "Failed to load plugins") {
		return "", false
	}
	return text, true
}
