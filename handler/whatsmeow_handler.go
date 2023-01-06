package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/store/sqlstore"

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
			conversation := v.Message.ImageMessage.Caption
			if conversation == nil {
				log.Println("[WhatsMeow][ImageToSticker][Caption]", "No caption found")
				return
			}

			if *conversation == "#sticker" {
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
			if arrayConversation[0] == "#gpt" {
				apiKey := os.Getenv("API_KEY_GPT")
				clientAI := gpt3.NewClient(apiKey)
				maxTokens := 4000
				resp, err := clientAI.CompletionWithEngine(ctx, gpt3.TextDavinci003Engine, gpt3.CompletionRequest{
					Prompt:      []string{conversation},
					MaxTokens:   &maxTokens,
					Temperature: proto.Float32(1.0),
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

func (wa *WhatsMeowHandler) RemoveBackground(evt interface{}) {
	// ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	// defer cancel()

	switch v := evt.(type) {
	case *events.Message:
		if !v.Info.IsFromMe && v.Message.GetImageMessage() != nil {
			conversation := v.Message.ImageMessage.Caption
			if conversation == nil {
				log.Println("[WhatsMeow][RemoveBackground][Caption]", "No caption found")
				return
			}

			if *conversation == "#bg" {
				image := v.Message.GetImageMessage()
				data, err := wa.Client.Download(image)
				if err != nil {
					log.Println("[WhatsMeow][RemoveBackground][Download]", err)
					return
				}

				exts, _ := mime.ExtensionsByType(image.GetMimetype())
				rawPath := fmt.Sprintf("assets/raw/%s%s", v.Info.ID, exts[2])
				err = os.WriteFile(rawPath, data, 0600)
				if err != nil {
					log.Println("[WhatsMeow][RemoveBackground][WriteFile]", err)
					return
				}

				pr, pw := io.Pipe()
				form := multipart.NewWriter(pw)

				go func() {
					defer pw.Close()

					err := form.WriteField("name", v.Info.ID)
					if err != nil {
						return
					}

					err = form.WriteField("image_extension", exts[2])

					file, err := os.Open(rawPath) // path to image file
					if err != nil {
						return
					}

					w, err := form.CreateFormFile("image_file", "sampleImageFileName.png")
					if err != nil {
						return
					}

					_, err = io.Copy(w, file)
					if err != nil {
						return
					}

					form.Close()
				}()

				url := "https://sdk.photoroom.com/v1/segment"
				req, err := http.NewRequest("POST", url, pr)
				if err != nil {
					log.Println("[WhatsMeow][RemoveBackground][NewRequest]", err)
					return
				}
				req.Header.Set("Content-Type", form.FormDataContentType())
				req.Header.Set("x-api-key", os.Getenv("API_KEY_PHOTO_ROOM"))

				res, err := http.DefaultClient.Do(req)
				if err != nil {
					log.Println("[WhatsMeow][RemoveBackground][Do]", err)
					return
				}
				defer res.Body.Close()

				decoder := json.NewDecoder(req.Body)
				log.Println(decoder)
			}
		}
	}
}
