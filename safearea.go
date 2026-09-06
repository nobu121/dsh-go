package main

import "runtime"

// Enough to sit under the traffic lights without a second empty band.
const harnessSafeTopPX = 16

// Injected via the window CSS/JS options so it does not depend on the Wails
// runtime (the harness page is http://127.0.0.1, so ExecJS stays queued).
//
// Only the sidebar brand row (CSS-module suffix logoRow) is shifted.
const harnessSafeAreaCSS = `[class*="_logoRow"]{margin-top:16px!important}`

const harnessSafeAreaJS = `(function(){
  if (window.__dshGoSafeArea) return;
  window.__dshGoSafeArea = true;
  var css = "[class*=\\"_logoRow\\"]{margin-top:16px!important}";
  function put(){
    if (document.getElementById("dsh-go-safe-area")) return;
    var s = document.createElement("style");
    s.id = "dsh-go-safe-area";
    s.textContent = css;
    (document.head || document.documentElement).appendChild(s);
  }
  put();
  new MutationObserver(put).observe(document.documentElement, {childList:true, subtree:true});
})();`

func harnessWindowSafeArea() (css, js string) {
	if runtime.GOOS != "darwin" {
		return "", ""
	}
	return harnessSafeAreaCSS, harnessSafeAreaJS
}
