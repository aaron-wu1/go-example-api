package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
)

// redis client
var redisClient *redis.Client = redis.NewClient(&redis.Options{
	Addr: "redis:6379",
})

// tag struct
type OpenGraphTags struct {
	Title string `json:"title"`
	Desc  string `json:"description"`
	Img   string `json:"image"`
	URL   string `json:"url"`
}

// fetch tags
func FetchHTML(url string) (*goquery.Document, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch url: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code error: %d %s", resp.StatusCode, resp.Status)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse html: %s", err)
	}
	return doc, nil
}

// parse tags
func ParseTagsFromHTML(doc *goquery.Document) *OpenGraphTags {
	// grabs content from meta tags
	tags := OpenGraphTags{
		Title: doc.Find("meta[property='og:title']").AttrOr("content", ""),
		Desc:  doc.Find("meta[property='og:description']").AttrOr("content", ""),
		Img:   doc.Find("meta[property='og:image']").AttrOr("content", ""),
		URL:   doc.Find("meta[property='og:url']").AttrOr("content", ""),
	}
	return &tags
}

// preview tags
func previewHandler(ctx *gin.Context) {
	// grabs url arg from query
	fmt.Printf("url: %s\n", ctx.Query("url"))
	url := ctx.Query("url")
	if url == "" {
		ctx.JSON((http.StatusBadRequest), gin.H{"error": "url is required"}) // bad request
		return
	}
	// check cache
	tags, err := CacheHandler(url)
	if err != nil {
		fmt.Print(err)
	}
	fmt.Printf("tags: %+v\n", tags)
	ctx.JSON(http.StatusOK, tags)
}

// cache tags
func CacheHandler(url string) (*OpenGraphTags, error) {
	// check if data is cached
	cachedData, err := redisClient.Get(url).Result()
	// if data not in redis
	if err == redis.Nil {
		fmt.Println("[LOG] data not in cache")
		doc, err := FetchHTML(url)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch html: %s", err)
		}
		tags := ParseTagsFromHTML(doc)
		// marshall tags to store in redis
		jsonTags, _ := json.Marshal(tags)
		redisClient.Set(url, jsonTags, time.Hour).Err()
		cachedData = string(jsonTags)
	} else if err != nil {
		return nil, fmt.Errorf("failed to get data from cache: %s", err)
	} else {
		fmt.Println("[LOG] data in cache")
	}
	// unmarshal cached data
	var tags OpenGraphTags
	if err := json.Unmarshal([]byte(cachedData), &tags); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached data: %s", err)
	}
	return &tags, nil
}

func main() {
	fmt.Println("STARTING SERVER1")
	r := gin.Default()
	r.GET("/preview", previewHandler)
	r.Run(":8080")
}
