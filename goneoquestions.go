package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"net/http"
	"os"
	"time"

	"github.com/mrjones/oauth"
)

func Usage() {
	fmt.Println("Usage:")
	fmt.Print("go run goneoquestions.go")
	fmt.Print("  --consumerkey <consumerkey>")
	fmt.Println("  --consumersecret <consumersecret>")
}

func main() {
	var consumerKey *string = flag.String(
		"consumerkey",
		"",
		"Consumer Key from Twitter. See: https://dev.twitter.com/apps/new")

	var consumerSecret *string = flag.String(
		"consumersecret",
		"",
		"Consumer Secret from Twitter. See: https://dev.twitter.com/apps/new")
	var accessTokenKey *string = flag.String(
		"accesstoken",
		"",
		"Access Token from Twitter.")

	var accessTokenSecret *string = flag.String(
		"accesstokensecret",
		"",
		"Access Token Secret from Twitter.")

	flag.Parse()

	if len(*consumerKey) == 0 || len(*consumerSecret) == 0 {
		fmt.Println("You must set the --consumerkey and --consumersecret flags.")
		fmt.Println("---")
		Usage()
		os.Exit(1)
	}

	c := oauth.NewConsumer(
		*consumerKey,
		*consumerSecret,
		oauth.ServiceProvider{
			RequestTokenUrl:   "http://api.twitter.com/oauth/request_token",
			AuthorizeTokenUrl: "https://api.twitter.com/oauth/authorize",
			AccessTokenUrl:    "https://api.twitter.com/oauth/access_token",
		})

	accessToken := &oauth.AccessToken{*accessTokenKey, *accessTokenSecret}

	posted := map[string]bool{}
	for {
		qs := getLatestSOQuestions()
		for _, q := range qs {
			if _, ok := posted[q]; !ok {
				fmt.Println(q)
				posted[q] = true
				_, err := c.Post(
					"http://api.twitter.com/1.1/statuses/update.json",
					map[string]string{
						"status": q,
					},
					accessToken)
				if err != nil {
					fmt.Println(err)
				}
				time.Sleep(10 * time.Second)
			}
		}
		time.Sleep(10 * time.Minute)
	}
}

type SOQueryResponse struct {
	Items []SOItem `json:"items"`
}

type SOItem struct {
	Title string `json:"title"`
	Link  string `json:"link"`
}

var startTime int64 = time.Now().Unix()

func getLatestSOQuestions() []string {
	t := time.Now().Unix() - (60 * 60 * 24)
	if t < startTime {
		t = startTime
	}
	timeStr := fmt.Sprintf("%d", t)
	url := "https://api.stackexchange.com/2.1/search?fromdate=" + timeStr + "&order=asc&sort=creation&tagged=neo4j&site=stackoverflow"
	fmt.Println("url: " + url)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
	}
	response := SOQueryResponse{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		fmt.Println(err)
	}
	questions := []string{}
	for _, item := range response.Items {
		q := "\"" + html.UnescapeString(item.Title)
		if len(q) > 100 {
			q = q[:100] + "..."
		}
		q = q + "\" #neo4j " + item.Link
		questions = append(questions, q)
	}
	return questions
}
