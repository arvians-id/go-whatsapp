package handler

import (
	"context"
	"fmt"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/PullRequestInc/go-gpt3"
	"github.com/arvians-id/go-whatsapp/utils"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

type WhatsMeowHandler struct {
	Client *whatsmeow.Client
	DB     *sqlstore.Container
}

func (wa *WhatsMeowHandler) ImageToSticker(evt interface{}) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	switch v := evt.(type) {
	case *events.Message:
		if !v.Info.IsFromMe && v.Message.GetImageMessage() != nil {
			newPath := filepath.Join(".", "assets/raw")
			err := os.MkdirAll(newPath, os.ModePerm)
			if err != nil {
				log.Println("[WhatsMeow][ImageToSticker][MkdirAll]", err)
				return
			}

			newPath = filepath.Join(".", "assets/converted")
			err = os.MkdirAll(newPath, os.ModePerm)
			if err != nil {
				log.Println("[WhatsMeow][ImageToSticker][MkdirAll]", err)
				return
			}

			image := v.Message.GetImageMessage()
			data, err := wa.Client.Download(image)
			if err != nil {
				log.Println("[WhatsMeow][ImageToSticker][Download]", err)
				return
			}

			exts, _ := mime.ExtensionsByType(image.GetMimetype())
			rawPath := fmt.Sprintf("assets/raw/%s%s", v.Info.ID, exts[0])
			convertedPath := fmt.Sprintf("assets/converted/%s%s", v.Info.ID, ".webp")
			err = os.WriteFile(rawPath, data, 0600)
			if err != nil {
				log.Println("[WhatsMeow][ImageToSticker][WriteFile]", err)
				return
			}

			err = utils.ConvertImage(rawPath, convertedPath)
			if err != nil {
				log.Println("[WhatsMeow][ImageToSticker][ConvertImage]", err)
				return
			}

			//utils.GenerateMetadata(convertedPath)

			dataBytes, err := os.ReadFile(convertedPath)
			if err != nil {
				log.Println("[WhatsMeow][ImageToSticker][ReadFile]", err)
				return
			}

			uploaded, err := wa.Client.Upload(ctx, dataBytes, whatsmeow.MediaImage)
			if err != nil {
				log.Println("[WhatsMeow][ImageToSticker][Upload]", err)
				return
			}

			_, err = wa.Client.SendMessage(ctx, v.Info.Sender, "", &waProto.Message{
				StickerMessage: &waProto.StickerMessage{
					Url:           proto.String(uploaded.URL),
					DirectPath:    proto.String(uploaded.DirectPath),
					MediaKey:      uploaded.MediaKey,
					Mimetype:      proto.String(http.DetectContentType(dataBytes)),
					FileEncSha256: uploaded.FileEncSHA256,
					FileSha256:    uploaded.FileSHA256,
					FileLength:    proto.Uint64(uint64(len(dataBytes))),
				},
			})
			if err != nil {
				log.Println("[WhatsMeow][ImageToSticker][SendMessage]", err)
				return
			}

			os.Remove(convertedPath)
			os.Remove(rawPath)
		}
	}
}

func (wa *WhatsMeowHandler) StickerToImage(evt interface{}) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	switch v := evt.(type) {
	case *events.Message:
		if !v.Info.IsFromMe && v.Message.GetStickerMessage() != nil {
			_, err := wa.Client.SendMessage(ctx, v.Info.Sender, "", &waProto.Message{
				ImageMessage: &waProto.ImageMessage{
					Mimetype:      proto.String("image/png"),
					Url:           v.Message.StickerMessage.Url,
					DirectPath:    v.Message.StickerMessage.DirectPath,
					MediaKey:      v.Message.StickerMessage.MediaKey,
					FileEncSha256: v.Message.StickerMessage.FileEncSha256,
					FileSha256:    v.Message.StickerMessage.FileSha256,
					FileLength:    v.Message.StickerMessage.FileLength,
				},
			})
			if err != nil {
				log.Println("[WhatsMeow][StickerToImage][SendMessage]", err)
				return
			}
		}
	}
}

func (wa *WhatsMeowHandler) ConversationWithOpenAICompletion(evt interface{}) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	switch v := evt.(type) {
	case *events.Message:
		if !v.Info.IsFromMe && v.Message.GetConversation() != "" {
			conversation := v.Message.GetConversation()
			arrayConversation := strings.Split(conversation, " ")
			if arrayConversation[0] == "#comp" {
				apiKey := os.Getenv("API_KEY")
				clientAI := gpt3.NewClient(apiKey)
				resp, err := clientAI.Completion(ctx, gpt3.CompletionRequest{
					Prompt: []string{conversation},
					Echo:   true,
				})
				if err != nil {
					log.Println("[WhatsMeow][ConversationWithOpenAICompletion][Completion]", err)
					return
				}
				getCompletion := resp.Choices[0].Text

				_, err = wa.Client.SendMessage(ctx, v.Info.Sender, "", &waProto.Message{
					Conversation: proto.String(getCompletion),
				})
				if err != nil {
					log.Println("[WhatsMeow][ConversationWithOpenAICompletion][SendMessage]", err)
					return
				}
			}
		}
	}
}
