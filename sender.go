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
	Feeds             []Feed
	BadFeeds          []string
	From, To, Subject string
}

type Feed struct {
	URL     string
	Updates []Update
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
		yesterday  = time.Now().UTC().Add(-24 * time.Hour).Format("2006-01-02")
		users, err = D.GetUsers()
	)

	if err != nil {
		fmt.Printf("[ERROR] could not get users/feeds: %s\n", err)
		return
	}

	fmt.Printf("processing feeds, looking for: %s\n", yesterday)

	for _, u := range users {
		fmt.Printf("Processing feeds for %s\n", u.Email)

		updates := &Updates{}

		for _, feed := range u.Feeds {
			processFeed(feed.URL, yesterday, updates)
		}

		if len(updates.Feeds) == 0 && len(updates.BadFeeds) == 0 {
			fmt.Println("no updates today, skipping")
			return
		}

		sendEmail(u.Email, yesterday, updates)
	}
	fmt.Println("done")
}

func processFeed(feed, date string, updates *Updates) {
	fmt.Printf("trying feed %q\n", feed)

	f := Feed{URL: feed}

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

		if strings.HasPrefix(name, date) {
			name = strings.TrimSpace(name[len(date):])
		} else {
			continue
		}
		if strings.HasPrefix(name, ": ") {
			name = strings.TrimSpace(name[2:])
		}
		if strings.HasPrefix(name, "- ") {
			name = strings.TrimSpace(name[2:])
		}

		u, err := url.Parse(path)
		if err != nil {
			fmt.Printf("error parsing url on line %q of feed %q: %s\n", path, feed, err)
			continue
		}

		addr := base.ResolveReference(u).String()
		f.Updates = append(f.Updates, Update{URL: addr, Title: name})
	}

	if err := lines.Err(); err != nil {
		fmt.Printf("error reading lines of feed %s: %s\n", feed, err)
		updates.BadFeeds = append(updates.BadFeeds, feed)
		return
	}

	if len(f.Updates) > 0 {
		updates.Feeds = append(updates.Feeds, f)
	}
}

func sendEmail(email, date string, updates *Updates) {
	var (
		buf  = &bytes.Buffer{}
		host = os.Getenv("SMTP_HOST")
		addr = fmt.Sprintf("%s:%s", host, os.Getenv("SMTP_PORT"))
		user = os.Getenv("SMTP_USERNAME")
		pass = os.Getenv("SMTP_PASSWORD")
	)

	updates.From = os.Getenv("SMTP_FROM")
	updates.To = email
	updates.Subject = fmt.Sprintf("Daily Gemini News for %s", date)

	fmt.Printf("Sending updates from %d feeds (and %d bad feeds)\n", len(updates.Feeds), len(updates.BadFeeds))
	err := emailT.Execute(buf, updates)
	if err != nil {
		fmt.Printf("error rendering email: %s\n", err)
		return
	}

	err = smtp.SendMail(addr, smtp.PlainAuth("", user, pass, host), updates.From, []string{updates.To}, buf.Bytes())
	if err != nil {
		fmt.Printf("error sending email: %s\n", err)
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
