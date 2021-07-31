package main

import "github.com/pitr/feed/db"

var D *db.Conn

func main() {
	D = db.NewConn()

	// go runSender()
	fetchAndSend()

	runServer("127.0.0.1:7777")
}
