package main

import (
	"fmt"
	"net/http"
	"log"
	"strconv"
	"strings"
	"time"
	"encoding/json"
	"sync"
	"os"

	"go.uber.org/ratelimit"
	"github.com/PuerkitoBio/goquery"
)

type Mod struct {
	Title string
	PicURL string
	Authors string
	ReleaseDate int64
	Views int
	Rating float64
	Reviews int
	Description string
	Video bool
	Guide bool
	Screens bool
	Review bool
	Tags []string
	Platform int
	Url string
}

type Data struct {
	Scraped int64
	Data []Mod
}

func main() {
	fmt.Println("yep")

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	var modUrls []string

	for i := 0; i < 50; i++ {
		doc := getDoc("https://ap-pro.ru/stuff/page/" + strconv.Itoa(i + 1))
		doc.Find("header h2 a[title^=\"Подробнее о\"]").Each(func(i int, s *goquery.Selection) {
			modUrls = append(modUrls, s.AttrOr("href", ""))
		})
		if doc.Find(".ipsPagination_next.ipsPagination_inactive").Size() > 0 {
			break;
		}
	}

	now := time.Now()

	docs := make([]*goquery.Document, len(modUrls))

	rl := ratelimit.New(10)

	var wg sync.WaitGroup

	for i, v := range modUrls {
		wg.Add(1)
		go func(i int, v string) {
			defer wg.Done()
			_ = rl.Take()
			docs[i] = getDoc(v)
		}(i, v)
	}

	wg.Wait()

	fmt.Println("mod pages downloaded", time.Since(now))

	mods := make([]Mod, len(modUrls))

	for i, doc := range docs {
		var mod Mod
		mod.Title = doc.Find(".ipsPageHeader .ipsType_pageTitle span").Text();
		modInfoGrid := doc.Find(".modInfoGrid")
		mod.PicURL = strings.Split(modInfoGrid.Find(".cCmsRecord_image").AttrOr("style", ""), ")")[0][15:]
		mod.Authors = strings.TrimSpace(modInfoGrid.Find(".fa-user").Parent().Text())
		time, _ := time.Parse("02.01.2006", strings.TrimSpace(modInfoGrid.Find(".fa-clock-o").Parent().Text()))
		mod.ReleaseDate = time.Unix()
		views, _ := strconv.Atoi(strings.TrimSpace(strings.Split(modInfoGrid.Find(".fa-eye").Parent().Text(), "|")[0]))
		mod.Views = views
		rating := modInfoGrid.Find(".fa-line-chart").Parent().Text()
		rating = strings.Split(rating, " ")[1] // Это не обычный пробел, а &ensp;
		rating = strings.Replace(rating, ",", ".", -1)
		ratingFloat, _ := strconv.ParseFloat(rating, 64)
		mod.Rating = ratingFloat
		reviews := modInfoGrid.Find(".fa-line-chart").Parent().Text()
		reviews = strings.Split(reviews, " ")[3] // Это не обычный пробел, а &ensp;
		reviewsInt, _ := strconv.Atoi(strings.Split(reviews, " ")[1][1:])
		mod.Reviews = reviewsInt
		mod.Description = strings.TrimSpace(doc.Find("article section p").Text())
		additionalButtons := doc.Find("article .additionalButtons").Text()
		mod.Video = strings.Contains(additionalButtons, "Видео")
		mod.Guide = strings.Contains(additionalButtons, "Прохожден")
		mod.Screens = strings.Contains(additionalButtons, "Скрин")
		mod.Review = strings.Contains(additionalButtons, "Обзор")
		platform := strings.TrimSpace(modInfoGrid.Find(".fa-folder-open-o").Parent().Text())
		switch platform {
		case "Тень Чернобыля":
			mod.Platform = 0
		case "Чистое небо":
			mod.Platform = 1
		case "Зов Припяти":
			mod.Platform = 2
		case "Arma 3":
			mod.Platform = 3
		case "DayZ":
			mod.Platform = 4
		case "Minecraft":
			mod.Platform = 5
		case "Cry Engine 2":
			mod.Platform = 6
		default:
			log.Fatalf("Unknown platform %s (%s). Update code.", platform, mod.Title)
		}
		doc.Find(".ipsTags.ipsList_inline li span").Each(func(i int, s *goquery.Selection) {
			mod.Tags = append(mod.Tags, strings.TrimSpace(s.Text()))
		})
		mod.Url = modUrls[i]
		mods[i] = mod
	}

	now = time.Now()
	b, _ := json.MarshalIndent(Data{now.Unix(), mods}, "", "    ")
	err := os.WriteFile(strings.Replace(now.Format(time.RFC3339), ":", "-", -1) + ".json", b, 0666)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("ok")
}

func getDoc(url string) *goquery.Document {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	if resp.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", resp.StatusCode, resp.Status)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	return doc
}