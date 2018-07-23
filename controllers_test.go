package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/minio/minio-go"
	"github.com/stretchr/testify/assert"
)

const bucketName = "test"
const location = "us-east-1"

var uploadedFileName string

func prepareTestBucket(client *minio.Client) {
	err := client.MakeBucket(bucketName, location)
	if err != nil {
		exists, err := client.BucketExists(bucketName)
		if !(err == nil && exists) {
			panic(err)
		}
	}
	fmt.Printf("Successfully created %s\n", bucketName)
}

// postFile helps to post image for testing upload handler.
func postFile(filename string, targetURL string) (*http.Request, error) {
	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)
	fileWriter, err := bodyWriter.CreateFormFile("file", filename)
	if err != nil {
		fmt.Println("Error while writing to file")
		return nil, err
	}

	fh, err := os.Open(filename)
	if err != nil {
		fmt.Println("Error while opening file")
		return nil, err
	}
	defer fh.Close()

	_, err = io.Copy(fileWriter, fh)
	if err != nil {
		return nil, err
	}

	contentType := bodyWriter.FormDataContentType()
	bodyWriter.Close()

	req, err := http.NewRequest("POST", targetURL, bodyBuf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return req, nil
}

func TestHealthController(t *testing.T) {
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HealthController)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, rr.Code, http.StatusOK)
}

func TestUploadHandler(t *testing.T) {
	cases := []struct {
		file     string
		expected string
	}{
		{"./fixtures/a.jpg", ".jpg"},
		{"./fixtures/a.png", ".png"},
		{"./fixtures/a.gif", ".gif"},
	}

	appCtx := CreateContext()
	router := InitRouter(appCtx)
	prepareTestBucket(appCtx.Storage)

	for _, c := range cases {
		req, err := postFile(c.file, fmt.Sprintf("/%s/%s/upload", ApiVersion, bucketName))
		if err != nil {
			assert.Nil(t, err)
		}

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		resp := rr.Result()
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			assert.Nil(t, err)
		}

		ur := UploadResponse{}
		err = json.Unmarshal(respBody, &ur)
		if err != nil {
			assert.Nil(t, err)
		}

		if c.expected == ".jpg" {
			uploadedFileName = ur.File
		}

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, c.expected, filepath.Ext(ur.File))
	}
}

func TestDownloadHandler(t *testing.T) {
	req, err := http.NewRequest("GET", fmt.Sprintf("/%s/%s/%s?resize=200x200", ApiVersion, bucketName, uploadedFileName), nil)
	if err != nil {
		assert.Nil(t, err)
	}

	appCtx := CreateContext()
	router := InitRouter(appCtx)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestGenerateFileName(t *testing.T) {
	cases := []struct {
		fileType string
		expected string
	}{
		{"image/jpeg", ".jpg"},
		{"image/gif", ".gif"},
		{"image/png", ".png"},
	}
	for _, c := range cases {
		actual, err := GenerateFileName(c.fileType)
		assert.Nil(t, err)
		assert.Equal(t, c.expected, filepath.Ext(actual))
	}
}
