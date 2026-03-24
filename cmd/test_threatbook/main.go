package main

import (
	"encoding/json"
	"fmt"

	"github.com/secflow/client/pkg/rodutil"
)

func main() {
	browser, err := rodutil.GetBrowser(nil)
	if err != nil {
		panic(err)
	}

	page, err := rodutil.NewPage(browser)
	if err != nil {
		panic(err)
	}
	defer page.Close()

	fmt.Println("Navigating to main page...")
	page.Navigate("https://x.threatbook.com/v5/vulIntelligence")
	page.WaitLoad()

	script := `() => {
		return fetch('https://x.threatbook.com/v5/node/vul_module/homePage', {
			method: 'GET',
			headers: {
				'Accept': 'application/json, text/plain, */*',
				'Accept-Language': 'zh-CN,zh;q=0.9,en;q=0.8,en-GB;q=0.7,en-US;q=0.6',
				'Referer': 'https://x.threatbook.com/v5/vulIntelligence',
				'Origin': 'https://mp.weixin.qq.com/'
			}
		})
		.then(r => r.json())
		.catch(e => ({error: e.message}));
	}`
	result, err := page.Eval(script)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	var data interface{}
	json.Unmarshal([]byte(result.Value.String()), &data)
	if data == nil {
		fmt.Println("Got null response")
	} else {
		b, _ := json.MarshalIndent(data, "", "  ")
		fmt.Println(string(b)[:2000])
	}
}
