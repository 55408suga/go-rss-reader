package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// API_URLはdockerで公開されているエンドポイントに合わせて変更してください。
const API_URL = "http://localhost:8080/api/v1/feeds"

var feedURLs = []string{
	"http://feeds.feedburner.com/GoogleJapanDeveloperRelationsBlog?format=xml",
	"https://engineering.linecorp.com/ja/feed/",
	"https://medium.com/feed/mixi-developers",
	// 必要に応じて追加
}

func main() {
	for i, url := range feedURLs {
		if url == "" {
			continue
		}
		payload := map[string]string{"feed_url": url}
		body, _ := json.Marshal(payload)
		resp, err := http.Post(API_URL, "application/json", bytes.NewReader(body))
		if err != nil {
			fmt.Printf("failed to POST %s: %v\n", url, err)
			continue
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		fmt.Printf("%d: %s -> %d\n", i+1, url, resp.StatusCode)
	}
}
