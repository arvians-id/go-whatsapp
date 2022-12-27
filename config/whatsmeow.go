package config

import (
	"context"
	"fmt"
	"github.com/arvians-id/go-whatsapp/utils"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"strings"
)

var client *whatsmeow.Client

func eventHandler(evt interface{}) {
	ctx := context.Background()

	switch v := evt.(type) {
	case *events.Message:
		if !v.Info.IsFromMe {
			if v.Message.GetStickerMessage() != nil {
				_, err := utils.StickerToImage(ctx, v, client)
				if err != nil {
					fmt.Println("err", err)
					return
				}
			}
			if v.Message.GetImageMessage() != nil && v.Message.ImageMessage.GetCaption() == "#sticker" {
				_, err := utils.ImageToSticker(ctx, v, client)
				if err != nil {
					fmt.Println("err", err)
					return
				}
			}
			if v.Message.GetConversation() != "" {
				conversation := v.Message.GetConversation()
				arrayConversation := strings.Split(conversation, " ")
				if arrayConversation[0] == "#ai" {
					_, err := utils.ConversationWithOpenAI(ctx, v, client, conversation)
					if err != nil {
						fmt.Println("err", err)
						return
					}
				} else {
					_, err := utils.ConversationWithOpenAI(ctx, v, client, "undefined")
					if err != nil {
						fmt.Println("err", err)
						return
					}
				}
			}
		}
	}
}
func NewInitializedWhatsMeow() (*whatsmeow.Client, error) {
	dbLog := waLog.Stdout("Database", "DEBUG", true)
	container, err := sqlstore.New("sqlite3", "file:go_whatsapp.db?_foreign_keys=on", dbLog)
	if err != nil {
		return nil, err
	}

	// If you want multiple sessions, remember their JIDs and use .GetDevice(jid) or .GetAllDevices() instead.
	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		return nil, err
	}

	clientLog := waLog.Stdout("Client", "DEBUG", true)
	client = whatsmeow.NewClient(deviceStore, clientLog)
	client.AddEventHandler(eventHandler)

	if client.Store.ID == nil {
		// No ID stored, new login
		qrChan, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			return nil, err
		}

		for evt := range qrChan {
			if evt.Event == "code" {
				fmt.Println("QR code:", evt.Code)
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		err = client.Connect()
		if err != nil {
			return nil, err
		}
	}

	return client, nil
}
