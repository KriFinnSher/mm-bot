package main

func main() {
	app := &application{}
	app.init()
	app.sendMsgToChannel("Привет! Я бот.", "")
	app.startWebSocket()
}
