package main

func main() {
	go runSender()
	// fetchAndSend()

	runServer("127.0.0.1:7777")
}
