package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mrjones/oauth"
	"github.com/VividCortex/robustly"
)

func Usage() {
	fmt.Println("Usage:")
	fmt.Print("go run goneoquestions.go")
	fmt.Print("  --consumerkey <consumerkey>")
	fmt.Println("  --consumersecret <consumersecret>")
}

var posted = map[string]bool{}

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

func main() {
	flag.Parse()

	if len(*consumerKey) == 0 || len(*consumerSecret) == 0 {
		fmt.Println("You must set the --consumerkey and --consumersecret flags.")
		fmt.Println("---")
		Usage()
		os.Exit(1)
	}

	robustly.Run(loop, &robustly.RunOptions{
		RateLimit:  1.0,
		Timeout:    1 * time.Second,
		PrintStack: true,
		RetryDelay: 10 * time.Minute,
	})
}

func loop() {
	c := oauth.NewConsumer(
		*consumerKey,
		*consumerSecret,
		oauth.ServiceProvider{
			RequestTokenUrl:   "http://api.twitter.com/oauth/request_token",
			AuthorizeTokenUrl: "https://api.twitter.com/oauth/authorize",
			AccessTokenUrl:    "https://api.twitter.com/oauth/access_token",
		})

	accessToken := &oauth.AccessToken{*accessTokenKey, *accessTokenSecret}
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
				fmt.Println("sleeping for 10 seconds")
				time.Sleep(10 * time.Second)
			}
		}
		fmt.Println("sleeping for 5 minutes")
		time.Sleep(5 * time.Minute)
	}
}

type SOQueryResponse struct {
	Items        []SOItem `json:"items"`
	Backoff      uint     `json:"backoff"`
	ErrorName    string   `json:"error_name"`
	ErrorMessage string   `json:"error_message"`
}

type SOItem struct {
	Title string `json:"title"`
	Link  string `json:"link"`
}

var startTime int64 = time.Now().Unix() - (60 * 60)

func getLatestSOQuestions() []string {
	t := time.Now().Unix() - (60 * 60 * 24)
	if t < startTime {
		t = startTime
	}
	timeStr := fmt.Sprintf("%d", t)
	url := "https://api.stackexchange.com/2.1/search?fromdate=" + timeStr + "&order=asc&sort=creation&tagged=neo4j;cypher&site=stackoverflow"
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
	//{"error_name":"throttle_violation","error_message":"too many requests from this IP, more requests available in 76366 seconds","error_id":502}
	if len(response.Items) == 0 {
		fmt.Println("items length is 0... printing response:")
		fmt.Println(response.ErrorName)
		fmt.Println(response.ErrorMessage)
		if response.ErrorName == "throttle_violation" {
			split := strings.Split(response.ErrorMessage, " ")
			secs, err := strconv.Atoi(split[len(split)-2])
			if err != nil {
				fmt.Println(err)
			} else {
				if secs < 100000 {
					fmt.Println(fmt.Sprintf("throttled, sleeping for %ds", secs))
					time.Sleep(time.Duration(secs+1) * time.Second)
				}
			}
		}
	}
	if response.Backoff > 0 {
		fmt.Println(fmt.Sprintf("backoff set, sleeping for %ds", response.Backoff))
		time.Sleep(time.Duration(response.Backoff+1) * time.Second)
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
