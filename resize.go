package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
)

const MaxResizeSize = 3000 // In px, same as f.ex. Uploadcare resize
const ResizeJpegQuality = 92

// ImageInfo represents image info.
type ImageInfo struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// ResizeImage resizes image.
func ResizeImage(ctx *AppContext, input io.Reader, mimeType string) (io.Reader, error) {
	var reqBody io.Reader

	// Skip if requested dimensions aren't presented
	if len(ctx.Dimensions) == 0 {
		return input, nil
	}

	// GIF resizing isn't supported in Imaginary, so skip it
	if mimeType == MIMEImageGIF {
		return input, nil
	}

	// Validate abnormal dimensions
	err := validateMaxDimensions(ctx.Dimensions)
	if err != nil {
		return nil, err
	}

	// We need to duplicate stream to reuse it later in requests & returns on non-200
	buf := bytes.NewBuffer(nil)
	teeInput := io.TeeReader(input, buf)

	// Prevent extra request for info if `enlarge` option isn't presented
	if !ctx.Options.Enlarge {
		// Check image suitability for `enlarge` option
		isSuitable, err := checkImageForEnlarge(&ctx.Config.ImageServiceURL, &teeInput, &mimeType, ctx.Dimensions)
		if !isSuitable || err != nil {
			return buf, nil
		}
		// Assign different readers for reqBody due to condition
		reqBody = buf
	} else {
		reqBody = teeInput
	}

	// Prepare URL
	resizeURL := fmt.Sprintf(
		"%s/thumbnail?quality=%d&width=%d",
		ctx.Config.ImageServiceURL, ResizeJpegQuality, ctx.Dimensions[0],
	)
	if len(ctx.Dimensions) == 2 {
		resizeURL += fmt.Sprintf("&height=%v", ctx.Dimensions[1])
	}

	// Post request
	resp, err := http.Post(resizeURL, mimeType, reqBody)
	// If Image Service isn't available return input
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			log.Printf("Image Service responded unsuccessfully with status %v", resp.StatusCode)
		}
		return input, nil
	}

	// Copy response body to buffer for body closing in this scope
	result := bytes.NewBuffer(nil)
	if _, err = io.Copy(result, resp.Body); err != nil {
		return input, err
	}

	// Always use defer after a succesful resource allocation
	if resp != nil {
		defer resp.Body.Close()
	}

	return result, err
}

// Prevent resizing abnormal dimensions from user
func validateMaxDimensions(input []int) error {
	dict := [2]string{
		"width",
		"height",
	}

	if len(input) == 1 {
		if input[0] > MaxResizeSize {
			return errors.New("Requested resize width is abnormal")
		}
	}
	if len(input) == 2 {
		for i, item := range input {
			if item > MaxResizeSize {
				return fmt.Errorf("Requested resize %s is abnormal", dict[i])
			}
		}
	}

	return nil
}

// Check image suitability for `enlarge` option
// Don't enlarge if original image width/height is smaller than in required resize option
func checkImageForEnlarge(imageServiceURL *string, input *io.Reader, mimeType *string, dimensions []int) (bool, error) {
	imageInfo, err := getImageInfo(imageServiceURL, *input, mimeType)
	if err != nil {
		return false, err
	}

	// Compare requested dimensions with input image width & height
	if imageInfo.Width < dimensions[0] {
		log.Println("Image width is smaller than resize dimension")
		return false, nil
	} else if len(dimensions) == 2 {
		if imageInfo.Width < dimensions[0] {
			log.Println("Image width is smaller than resize dimension")
			return false, nil
		}
		if imageInfo.Height < dimensions[1] {
			log.Println("Image height is smaller than resize dimension")
			return false, nil
		}
	}

	return true, nil
}

func getImageInfo(imageServiceURL *string, input io.Reader, mimeType *string) (*ImageInfo, error) {
	infoURL := fmt.Sprintf("%s/info", *imageServiceURL)
	resp, err := http.Post(infoURL, *mimeType, input)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Image Service responded unsuccessfully with status %v", resp.StatusCode)
	}

	// Always use defer after a succesful resource allocation
	if resp != nil {
		defer resp.Body.Close()
	}

	imageInfo := &ImageInfo{}
	err = json.NewDecoder(resp.Body).Decode(imageInfo)
	if err != nil {
		return nil, err
	}

	return imageInfo, nil
}
