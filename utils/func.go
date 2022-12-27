package utils

import (
	"context"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

func ImageToSticker(ctx context.Context, v *events.Message, client *whatsmeow.Client) (whatsmeow.SendResponse, error) {
	download, err := client.Download(v.Message.GetImageMessage())
	if err != nil {
		return whatsmeow.SendResponse{}, err
	}

	resp, err := client.Upload(ctx, download, whatsmeow.MediaImage)
	if err != nil {
		return whatsmeow.SendResponse{}, err
	}

	response, err := client.SendMessage(context.Background(), v.Info.Sender, "", &waProto.Message{
		StickerMessage: &waProto.StickerMessage{
			Mimetype:      proto.String("image/webp"),
			Url:           &resp.URL,
			DirectPath:    &resp.DirectPath,
			MediaKey:      resp.MediaKey,
			FileEncSha256: resp.FileEncSHA256,
			FileSha256:    resp.FileSHA256,
			FileLength:    &resp.FileLength,
		},
	})
	if err != nil {
		return whatsmeow.SendResponse{}, err
	}

	return response, nil
}

func StickerToImage(ctx context.Context, v *events.Message, client *whatsmeow.Client) (whatsmeow.SendResponse, error) {
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
