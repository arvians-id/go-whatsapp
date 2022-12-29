package main

import (
	"github.com/arvians-id/go-whatsapp/config"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalln(err)
	}

	// Initialize database
	db, err := config.NewInitializedSQLiteDatabase()
	if err != nil {
		log.Fatalln(err)
	}

	// Setup WhatsApp
	var whatsMeowClient *whatsmeow.Client
	client := config.NewInitializedWhatsMeow(whatsMeowClient, db)
	if err != nil {
		log.Fatalln(err)
	}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	client.Client.Disconnect()
}
