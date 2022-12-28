package utils

import (
	"context"
	"fmt"
	"log"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/PullRequestInc/go-gpt3"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// Convert Image to WebP
// Using https://developers.google.com/speed/webp/docs/cwebp
func convertImage(mediaPath string, convertedPath string) error {
	cmd := *exec.Command("cwebp", mediaPath, "-resize", "0", "600", "-o", convertedPath)
	err := cmd.Run()

	return err
}

func ImageToSticker(ctx context.Context, v *events.Message, client *whatsmeow.Client) (whatsmeow.SendResponse, error) {
	newpath := filepath.Join(".", "images/raw")
	err := os.MkdirAll(newpath, os.ModePerm)
	if err != nil {
		return whatsmeow.SendResponse{}, err
	}

	newpath = filepath.Join(".", "images/converted")
	err = os.MkdirAll(newpath, os.ModePerm)
	if err != nil {
		return whatsmeow.SendResponse{}, err
	}

	image := v.Message.GetImageMessage()
	data, err := client.Download(image)
	if err != nil {
		return whatsmeow.SendResponse{}, err
	}

	exts, _ := mime.ExtensionsByType(image.GetMimetype())
	rawPath := fmt.Sprintf("images/raw/%s%s", v.Info.ID, exts[0])
	convertedPath := fmt.Sprintf("images/converted/%s%s", v.Info.ID, ".webp")
	err = os.WriteFile(rawPath, data, 0600)
	if err != nil {
		return whatsmeow.SendResponse{}, err
	}

	err = convertImage(rawPath, convertedPath)
	if err != nil {
		return whatsmeow.SendResponse{}, err
	}

	// metadata.GenerateMetadata(convertedPath)

	dataBytes, err := os.ReadFile(convertedPath)
	if err != nil {
		return whatsmeow.SendResponse{}, err
	}

	uploaded, err := client.Upload(ctx, dataBytes, whatsmeow.MediaImage)
	if err != nil {
		return whatsmeow.SendResponse{}, err
	}

	response, err := client.SendMessage(ctx, v.Info.Sender, "", &waProto.Message{
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
		return whatsmeow.SendResponse{}, err
	}

	os.Remove(convertedPath)
	os.Remove(rawPath)

	return response, nil
}

func StickerToImage(ctx context.Context, v *events.Message, client *whatsmeow.Client) (whatsmeow.SendResponse, error) {
	log.Println("To Image", v.Message.StickerMessage)
	response, err := client.SendMessage(ctx, v.Info.Sender, "", &waProto.Message{
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
		return whatsmeow.SendResponse{}, err
	}

	return response, nil
}

func ConversationWithOpenAI(ctx context.Context, v *events.Message, client *whatsmeow.Client, prompt string) (whatsmeow.SendResponse, error) {
	conversation := "Cannot find your conversation"
	if prompt != "undefined" {
		apiKey := os.Getenv("API_KEY")
		clientAI := gpt3.NewClient(apiKey)
		resp, err := clientAI.Completion(ctx, gpt3.CompletionRequest{
			Prompt:    []string{prompt},
			MaxTokens: gpt3.IntPtr(30),
			Stop:      []string{"."},
			Echo:      true,
		})
		if err != nil {
			return whatsmeow.SendResponse{}, err
		}
		conversation = resp.Choices[0].Text
	}

	response, err := client.SendMessage(ctx, v.Info.Sender, "", &waProto.Message{
		Conversation: proto.String(conversation),
	})
	if err != nil {
		return whatsmeow.SendResponse{}, err
	}

	return response, nil
}
