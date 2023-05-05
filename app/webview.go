package app

import (
	"bookget/config"
	"github.com/jchv/go-webview2"
	"log"
	"os"
)

func OpenWebview(sUrl string, decodeURI bool) {
	w := webview2.NewWithOptions(webview2.WebViewOptions{
		Debug:     true,
		AutoFocus: true,
		WindowOptions: webview2.WindowOptions{
			Title:  "右键【刷新】，页面加载完后，请关闭窗口",
			Width:  800,
			Height: 600,
			IconId: 2, // icon resource id
			Center: true,
		},
	})
	if w == nil {
		log.Fatalln("Failed to load webview.")
	}
	defer w.Destroy()
	w.SetSize(800, 600, webview2.HintFixed)
	w.Navigate(sUrl)

	w.Bind("getCookie", func(returned string) {
		go getCookie(w, returned)
	})
	if decodeURI {
		w.Init(`(function() {
					let s = "User-Agent:" + navigator.userAgent + "\n" + "Cookie:" + decodeURIComponent(document.cookie)  
					window.getCookie(s).then(function(res) {
						console.log('getCookie res', res);
					});
				})();`)
	} else {
		w.Init(`(function() {
					let s = "User-Agent:" + navigator.userAgent + "\n" + "Cookie:" + document.cookie  
					window.getCookie(s).then(function(res) {
						console.log('getCookie res', res);
					});
				})();`)
	}
	w.Run()
}

func getCookie(w webview2.WebView, returned string) {
	//fmt.Println(returned)
	w.Dispatch(func() {
		w.Eval("")
	})
	os.WriteFile(config.Conf.CookieFile, []byte(returned), os.ModePerm)
	//w.Destroy()
}
