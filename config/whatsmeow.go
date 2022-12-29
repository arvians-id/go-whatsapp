package config

import (
	"context"
	"fmt"
	"github.com/arvians-id/go-whatsapp/handler"
	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
	"os"
)

func NewInitializedWhatsMeow(client *whatsmeow.Client, db *sqlstore.Container) handler.WhatsMeowHandler {
	whatsMeowHandler := handler.WhatsMeowHandler{
		Client: client,
		DB:     db,
	}

	// If want multiple sessions, remember their JIDs and use .GetDevice(jid) or .GetAllDevices() instead.
	deviceStore, err := whatsMeowHandler.DB.GetFirstDevice()
	if err != nil {
		return handler.WhatsMeowHandler{}
	}

	clientLog := waLog.Stdout("Client", "DEBUG", true)
	whatsMeowHandler.Client = whatsmeow.NewClient(deviceStore, clientLog)
	whatsMeowHandler.Client.AddEventHandler(whatsMeowHandler.ImageToSticker)
	whatsMeowHandler.Client.AddEventHandler(whatsMeowHandler.StickerToImage)
	whatsMeowHandler.Client.AddEventHandler(whatsMeowHandler.ConversationWithOpenAICompletion)

	if whatsMeowHandler.Client.Store.ID == nil {
		// No ID stored, new login
		qrChan, _ := whatsMeowHandler.Client.GetQRChannel(context.Background())
		err = whatsMeowHandler.Client.Connect()
		if err != nil {
			return handler.WhatsMeowHandler{}
		}

		for evt := range qrChan {
			if evt.Event == "code" {
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				fmt.Println("QR code:", evt.Code)
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		err = whatsMeowHandler.Client.Connect()
		if err != nil {
			return handler.WhatsMeowHandler{}
		}
	}

	return whatsMeowHandler
}
