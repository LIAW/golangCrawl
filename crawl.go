package main

import (
	"fmt"
	"github.com/PuerkitoBio/gocrawl"
	"github.com/PuerkitoBio/goquery"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	DEPTH = 5
)

var (
	downloaded map[string]bool = make(map[string]bool)
	lockx                      = make(chan int, 1)
	imgDir     string
)

func addDownloadImgUrl(url string) {
	lockx <- 1
	downloaded[url] = true
	<-lockx
}

func getName(url string) string {
	namelist := strings.Split(url, "/")
	fileName := namelist[len(namelist)-1]
	return fileName
}

func downImg(url string, chann chan int) {

	defer func() {
		chann <- 1
	}()
	if downloaded[url] {
		fmt.Println("====url: ", url, "已下载")
		return
	}

	resp, err := http.Get(url)
	delay := time.AfterFunc(3*time.Second, func() {
		return
	})

	defer delay.Stop()

	if err != nil {
		fmt.Println("下载图片: ", url, "失败, 原因: ", err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.ContentLength < 10000 {
		return
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("读取图片: ", url, "失败, 原因: ", err.Error())
		return
	}

	f, _ := os.Create("./img/" + getName(url))
	defer f.Close()

	f.Write(body)
	fmt.Println("----", resp.Request.URL)

	addDownloadImgUrl(url)
}

func parsingImgUrl(resp *http.Response) {
	fmt.Println("解析图片链接, 来自: ", resp.Request.URL)

	if resp == nil {
		fmt.Println("resp 为空")
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("读取网页失败")
	}
	str := string(body)
	re, _ := regexp.Compile("http://img\\S+?\\.jpg")
	newstr := re.FindAllString(str, -1)

	if len(newstr) == 0 {
		return
	}

	subChan := make(chan int, len(newstr))
	fmt.Println("图片数量: ", len(newstr))

	for i := 0; i < len(newstr); i++ {
		go downImg(newstr[i], subChan)
	}

	for i := 0; i < len(newstr); i++ {
		<-subChan
	}

}

type ExampleExtender struct {
	gocrawl.DefaultExtender
}

func (this *ExampleExtender) Visit(ctx *gocrawl.URLContext, res *http.Response, doc *goquery.Document) (interface{}, bool) {
	fmt.Println("visit url: ", ctx.URL(), "state: ", ctx.State)
	go parsingImgUrl(res)
	urls := processLinks(doc)
	links := make(map[*url.URL]interface{})
	i, _ := ctx.State.(int)
	nextDepth := i - 1
	if nextDepth <= 0 {
		return nil, false
	}
	for _, u := range urls {
		links[u] = nextDepth
	}
	return links, false
}

func (this *ExampleExtender) Filter(ctx *gocrawl.URLContext, isVisited bool) bool {
	if ctx.SourceURL() == nil {
		ctx.State = DEPTH
		return !isVisited
	}
	if ctx.State != nil {
		i, ok := ctx.State.(int)
		if ok && i > 0 {
			return !isVisited
		}
	} else {
		fmt.Println("ctx.state nil, ctx.sourceURL: ", ctx.SourceURL())
	}
	return false
}

//copy from worker.go
func processLinks(doc *goquery.Document) (result []*url.URL) {
	urls := doc.Find("a[href]").Map(func(_ int, s *goquery.Selection) string {
		val, _ := s.Attr("href")
		return val
	})
	for _, s := range urls {
		if len(s) > 0 && !strings.HasPrefix(s, "#") {
			if parsed, e := url.Parse(s); e == nil {
				parsed = doc.Url.ResolveReference(parsed)
				result = append(result, parsed)
			}
		}
	}
	return
}

func main() {
	cDIr, _ := filepath.Abs("")
	imgDir = cDIr + "/img"
	os.MkdirAll("img", 0755)
	fmt.Println(os.Args[0])
	fmt.Println(len(os.Args))
	// return
	opts := gocrawl.NewOptions(new(ExampleExtender))
	opts.CrawlDelay = 0
	opts.LogFlags = gocrawl.LogNone
	opts.EnqueueChanBuffer = 10000
	// opts.MaxVisits = 4
	c := gocrawl.NewCrawlerWithOptions(opts)
	c.Run(gocrawl.S{"http://pp.163.com/": DEPTH})
}
