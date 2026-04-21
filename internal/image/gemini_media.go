package image

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/jpeg"
	"image/png"
	"io"
	"strings"
)

func encodeDataURL(imageBytes []byte, mimeType string) (string, error) {
	if len(imageBytes) == 0 {
		return "", fmt.Errorf("no image bytes returned")
	}

	normalizedBytes, err := normalizePNG(imageBytes, mimeType)
	if err != nil {
		return "", err
	}

	return geminiDataPrefix + base64.StdEncoding.EncodeToString(normalizedBytes), nil
}

func decodeDataURL(url string) (io.ReadCloser, error) {
	header, payload, ok := strings.Cut(url, ",")
	if !ok || !strings.HasPrefix(header, "data:") || !strings.Contains(header, ";base64") {
		return nil, fmt.Errorf("unsupported data URI: %s", url)
	}

	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("decode data URI: %w", err)
	}

	return io.NopCloser(bytes.NewReader(data)), nil
}

func normalizePNG(imageBytes []byte, mimeType string) ([]byte, error) {
	if strings.EqualFold(strings.TrimSpace(mimeType), "image/png") {
		return append([]byte(nil), imageBytes...), nil
	}

	img, _, err := image.Decode(bytes.NewReader(imageBytes))
	if err != nil {
		return nil, fmt.Errorf("decode generated image: %w", err)
	}

	var buffer bytes.Buffer
	if err := png.Encode(&buffer, img); err != nil {
		return nil, fmt.Errorf("encode generated image as png: %w", err)
	}

	return buffer.Bytes(), nil
}

func decodedImageDimensions(imageBytes []byte) (int, int, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(imageBytes))
	if err != nil {
		return 0, 0, fmt.Errorf("decode generated image dimensions: %w", err)
	}

	return cfg.Width, cfg.Height, nil
}
