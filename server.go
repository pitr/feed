package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Payload struct {
	Message string `json:"Message"`
}

type Message struct {
	Mail struct {
		Source        string `json:"source"`
		CommonHeaders struct {
			ReturnPath string `json:"returnPath"`
			Subject    string `json:"subject"`
		} `json:"commonHeaders"`
	} `json:"mail"`
}

func runServer(addr string) {
	gin.SetMode(gin.ReleaseMode)
	gin.DisableConsoleColor()

	r := gin.Default()
	r.Any("/receive", func(c *gin.Context) {
		defer c.Status(http.StatusOK)

		var (
			payload Payload
			msg     Message
			err     error
		)
		if err = c.ShouldBindJSON(&payload); err != nil {
			fmt.Printf("payload bind err: %s\n", err)
			return
		}

		fmt.Println(payload.Message)

		err = json.Unmarshal([]byte(payload.Message), &msg)
		if err != nil {
			fmt.Printf("msg bind err: %s\n", err)
			return
		}

		fmt.Printf(":::: sub user '%s'/'%s' to '%s'\n", msg.Mail.Source, msg.Mail.CommonHeaders.ReturnPath, msg.Mail.CommonHeaders.Subject)

		u, err := D.FindOrCreateUser(msg.Mail.Source)
		if err != nil {
			fmt.Printf("could not find/create user: %s\n", err)
			return
		}

		err = D.AddFeed(u, msg.Mail.CommonHeaders.Subject)
		if err != nil {
			fmt.Printf("could not add feed: %s\n", err)
			return
		}
	})
	println("starting server on", addr)
	panic(r.Run(addr))
}
