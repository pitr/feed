package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"fmt"
	"mime/quotedprintable"
	"net/smtp"
	"net/url"
	"os"
	"strings"
	"text/template"
	"time"

	"git.sr.ht/~adnano/go-gemini"
	"github.com/jasonlvhit/gocron"
)

const (
	ws       = " \t\n\r"
	spacetab = " \t"
)

var (
	client = &gemini.Client{}
	//go:embed email.tmpl
	emailTmpl string
	emailT    = template.Must(template.New("email").
			Funcs(template.FuncMap{"quoteprintable": toQuotedPrintable}).
			Parse(emailTmpl))
)

type Updates struct {
	Updates           []Update
	Feeds, BadFeeds   []string
	From, To, Subject string
}

type Update struct {
	URL, Title string
}

func runSender() {
	err := gocron.Every(1).Day().At("17:00").Do(fetchAndSend)
	if err != nil {
		panic(err)
	}
	<-gocron.Start()
}

func fetchAndSend() {
	var (
		yesterday = time.Now().UTC().Add(-24 * time.Hour).Format("2006-01-02")
		updates   = &Updates{}
		feeds     = getFeeds()
	)

	fmt.Printf("processing feeds, looking for: %s\n", yesterday)

	for _, feed := range feeds {
		processFeed(feed, yesterday, updates)
	}

	if len(updates.Updates) == 0 {
		fmt.Println("no updates today, skipping")
		return
	}

	sendEmail(yesterday, updates)
	fmt.Println("done")
}

func getFeeds() []string {
	feeds := []string{}

	f, err := os.Open("feeds.txt")
	if err != nil {
		fmt.Printf("[ERROR] could not read feed file: %s\n", err)
		return feeds
	}
	defer f.Close()

	lines := bufio.NewScanner(f)
	for lines.Scan() {
		feeds = append(feeds, lines.Text())
	}
	if err = lines.Err(); err != nil {
		fmt.Printf("[ERROR] could not parse feed file: %s\n", err)
		return feeds
	}
	fmt.Printf("processing %d feeds\n", len(feeds))
	return feeds
}

func processFeed(feed, date string, updates *Updates) {
	fmt.Printf("trying feed %q\n", feed)

	found := false

	switch {
	case strings.HasPrefix(feed, "gemini://"): // expected
	case strings.HasPrefix(feed, "//"):
		feed = "gemini:" + feed
	case strings.HasPrefix(feed, "/"):
		feed = "gemini:/" + feed
	default:
		feed = "gemini://" + feed
	}

	base, err := url.Parse(feed)
	if err != nil {
		fmt.Printf("error parsing url of feed %s: %s\n", feed, err)
		updates.BadFeeds = append(updates.BadFeeds, feed)
		return
	}

	if base.Scheme != "gemini" {
		fmt.Printf("feed %q is not gemini, but %q\n", feed, base.Scheme)
		updates.BadFeeds = append(updates.BadFeeds, feed)
		return
	}

	res, err := client.Do(gemini.NewRequestFromURL(base))
	if err != nil {
		fmt.Printf("error reading feed %s: %s\n", feed, err)
		updates.BadFeeds = append(updates.BadFeeds, feed)
		return
	}
	if res.Status != gemini.StatusSuccess {
		fmt.Printf("unknown status for feed %s: %d\n", feed, res.Status)
		updates.BadFeeds = append(updates.BadFeeds, feed)
		return
	}
	defer res.Body.Close()
	lines := bufio.NewScanner(res.Body)
	for lines.Scan() {
		line := lines.Text()
		if !strings.HasPrefix(line, "=>") {
			continue
		}
		if !strings.Contains(line, date) {
			continue
		}

		line = line[2:]
		line = strings.TrimLeft(line, spacetab)
		split := strings.IndexAny(line, spacetab)
		if split == -1 {
			fmt.Printf("unexpected, perhaps URL has date, skipping: %q", line)
			continue
		}

		path := line[:split]
		name := strings.TrimLeft(line[split:], spacetab)

		u, err := url.Parse(path)
		if err != nil {
			fmt.Printf("error parsing url on line %q of feed %q: %s\n", path, feed, err)
			continue
		}

		addr := base.ResolveReference(u).String()
		updates.Updates = append(updates.Updates, Update{URL: addr, Title: name})
		found = true
	}

	if err := lines.Err(); err != nil {
		fmt.Printf("error reading lines of feed %s: %s\n", feed, err)
		updates.BadFeeds = append(updates.BadFeeds, feed)
		return
	}

	if found {
		updates.Feeds = append(updates.Feeds, feed)
	}
}

func sendEmail(date string, updates *Updates) {
	var (
		buf  = &bytes.Buffer{}
		host = os.Getenv("SMTP_HOST")
		addr = fmt.Sprintf("%s:%s", host, os.Getenv("SMTP_PORT"))
		user = os.Getenv("SMTP_USERNAME")
		pass = os.Getenv("SMTP_PASSWORD")
	)

	updates.From = os.Getenv("SMTP_FROM")
	updates.To = os.Getenv("SMTP_TO")
	updates.Subject = fmt.Sprintf("Daily Gemini News for %s", date)

	fmt.Printf("Sending %d updates from %d feeds (and %d bad feeds)\n", len(updates.Updates), len(updates.Feeds), len(updates.BadFeeds))
	err := emailT.Execute(buf, updates)
	if err != nil {
		panic(err)
	}

	err = smtp.SendMail(addr, smtp.PlainAuth("", user, pass, host), updates.From, []string{updates.To}, buf.Bytes())
	if err != nil {
		panic(err)
	}
}

func toQuotedPrintable(s string) (string, error) {
	var ac bytes.Buffer
	w := quotedprintable.NewWriter(&ac)
	_, err := w.Write([]byte(s))
	if err != nil {
		return "", err
	}
	err = w.Close()
	if err != nil {
		return "", err
	}
	return ac.String(), nil
}
