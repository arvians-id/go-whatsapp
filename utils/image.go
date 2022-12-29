package utils

import "os/exec"

// Convert Image to WebP
// Using https://developers.google.com/speed/webp/docs/cwebp
func ConvertImage(mediaPath string, convertedPath string) error {
	cmd := *exec.Command("cwebp", mediaPath, "-resize", "0", "600", "-o", convertedPath)
	err := cmd.Run()

	return err
}
