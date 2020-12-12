package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/minio/minio-go"
	uuid "github.com/satori/go.uuid"
)

const MaxUploadSize = 10 * 1024 * 1024 // 10mb

// UploadHandler is a wrapper to provide appContext to upload handler
type UploadHandler struct {
	ctx *AppContext
}

// UploadResponse represents response for uploaded image
type UploadResponse struct {
	File string `json:"file"`
}

// DownloadHandler is a wrapper to provide appContext to download handler
type DownloadHandler struct {
	ctx *AppContext
}

// HealthController handles app health checking
func HealthController(w http.ResponseWriter, r *http.Request) {
	// swagger:route GET /health health
	// ---
	// summary: Health check
	// description: Returns health info.
	// produces:
	//   - application/json
	// responses:
	//   200: healthStats
	body, _ := json.Marshal(GetHealthStats())
	w.Header().Set("Content-Type", MIMEApplicationJSON)
	w.Write(body)
}

// UploadController handles image uploading
func (u *UploadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /{bucket}/upload upload
	// ---
	// summary: Upload
	// description: Upload image to storage. You need to specify bucket in URL.
	// consumes:
	//   - multipart/form-data
	// produces:
	//   - application/json
	// requestBody:
	//   content:
	//     multipart/form-data:
	//       schema:
	//         properties:
	//           file:
	//             type: array
	//             items:
	//               type: string
	//               format: binary
	// parameters:
	//   - in: path
	//     name: bucket
	//     type: string
	//     required: true
	//     description: Bucket name in storage for uploading.
	//   - in: query
	//     name: enlarge
	//     type: boolean
	//     description: Don't enlarge if original image width/height is smaller than in required resize option.
	//   - in: query
	//     name: resize
	//     type: string
	//     description: Resize string. It must follow one of two patterns, '100x100' as width x height or '100' as width.
	//   - in: formData
	//     name: file
	//     type: file
	//     required: true
	//     description: Image file for uploading.
	// responses:
	//   '200':
	//     description: OK. Returns new file name for successfully uploaded image.
	//     content:
	//       application/json:
	//     schema:
	//       type: object
	//       properties:
	//         file:
	//           type: string
	//           description: Generated file name of uploaded image.
	//   '400':
	//     description: Bad request (RESIZE_PARAMS_ARE_INVALID, FILE_TOO_BIG, INVALID_FILE_CONTENT, INVALID_FILE_TYPE).
	//   '500':
	//     description: Internal error (CANT_GENERATE_FILENAME, CANT_RESIZE_FILE, CANT_POST_TO_STORAGE, CANT_MARSHAL_DATA).
	var result io.Reader
	var err error
	var ctx = u.ctx

	// Get resize dimensions with validation
	ctx.Dimensions, err = PrepareDimensions(ctx.Options.Resize)
	if err != nil {
		renderError(w, err, "RESIZE_PARAMS_ARE_INVALID", http.StatusBadRequest)
		return
	}

	// Validate file size
	r.Body = http.MaxBytesReader(w, r.Body, MaxUploadSize)
	if err = r.ParseMultipartForm(MaxUploadSize); err != nil {
		renderError(w, err, "FILE_TOO_BIG", http.StatusBadRequest)
		return
	}

	// Validate file pt.1
	file, handler, err := r.FormFile("file")
	if err != nil {
		renderError(w, err, "INVALID_FILE_CONTENT", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate file pt.2
	buf := bytes.NewBuffer(nil)
	if _, err = io.Copy(buf, file); err != nil {
		renderError(w, err, "INVALID_FILE_CONTENT", http.StatusBadRequest)
		return
	}

	// Validate content type
	fileType := http.DetectContentType(buf.Bytes())
	if fileType != MIMEImageJPEG && fileType != MIMEImageGIF && fileType != MIMEImagePNG {
		renderError(w, nil, "INVALID_FILE_TYPE", http.StatusBadRequest)
		return
	}

	// Generate hashed filename
	fileName, err := GenerateFileName(fileType)
	if err != nil {
		renderError(w, err, "CANT_GENERATE_FILENAME", http.StatusInternalServerError)
	}

	// Resize image or don't
	if ctx.Config.ResizeOnUpload {
		result, err = ResizeImage(ctx, buf, fileType)
		if err != nil {
			renderError(w, err, "CANT_RESIZE_FILE", http.StatusInternalServerError)
			return
		}
	} else {
		result = buf
	}

	// Post to storage
	n, err := ctx.Storage.PutObject(mux.Vars(r)["bucket"], fileName, result, -1, minio.PutObjectOptions{
		ContentType: handler.Header.Get("Content-Type"),
	})
	if err != nil {
		renderError(w, err, "CANT_POST_TO_STORAGE", http.StatusInternalServerError)
		return
	}

	log.Println("Uploaded: ", n)
	log.Printf("%#v\n", ctx.Storage)

	// Prepare json with result file URL
	ur := UploadResponse{fileName}
	resultURL, err := json.Marshal(ur)
	if err != nil {
		renderError(w, err, "CANT_MARSHAL_DATA", http.StatusInternalServerError)
		return
	}

	// Respond
	w.Header().Set("Content-Type", MIMEApplicationJSON)
	w.WriteHeader(http.StatusOK)
	w.Write(resultURL)
}

// DownloadController handles image downloading
func (d *DownloadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// swagger:operation GET /{bucket}/{object} download
	// ---
	// summary: Download
	// description: Download image from storage. You need to specify bucket & image object in URL.
	// produces:
	//   - image/jpeg
	//   - image/gif
	//   - image/png
	// parameters:
	//   - in: path
	//     name: bucket
	//     type: string
	//     required: true
	//     description: Bucket name in storage.
	//   - in: path
	//     name: object
	//     type: string
	//     required: true
	//     description: Object name in storage.
	// responses:
	//   '200':
	//     description: OK. Returns file from storage.
	//     content:
	//       'image/jpeg':
	//         schema:
	//           type: string
	//           format: binary
	//       'image/gif':
	//         schema:
	//           type: string
	//           format: binary
	//       'image/png':
	//         schema:
	//           type: string
	//           format: binary
	//   '400':
	//     description: Bad request (RESIZE_PARAMS_ARE_INVALID, INVALID_FILE_TYPE).
	//   '404':
	//     description: Not found (CANT_PROCESS_FILE_ON_STORAGE).
	//   '500':
	//     description: Internal error (CANT_READ_FILE, CANT_RESIZE_FILE, CANT_RESPOND_FILE).
	var result io.Reader
	var err error
	var ctx = d.ctx

	// Get resize dimensions with validation
	ctx.Dimensions, err = PrepareDimensions(ctx.Options.Resize)
	if err != nil {
		renderError(w, err, "RESIZE_PARAMS_ARE_INVALID", http.StatusBadRequest)
		return
	}

	// Get bucket & object names from request url
	var bucketName string
	var objName string
	separator := string(os.PathSeparator)
	pathParts := strings.Split(r.RequestURI, separator)
	if len(pathParts) >= 2 {
		bucketName = pathParts[1]
		for _, pathPart := range pathParts[2:] {
			// GOTCHA: Minio requires unescaped symbols
			result, err := url.PathUnescape(pathPart)
			if err != nil {
				renderError(w, err, "CANT_UNESCAPE_URL_PART", http.StatusBadRequest)
			}
			// GOTCHA: Minio converts slashes ("/") in directory name (objName) to ":", so we need to convert it
			result = strings.ReplaceAll(result, "/", ":")
			objName += separator + result
		}
	}

	// Remove from objName all params (e.g. "resize")
	objName = strings.Split(objName, "?")[0]

	// Get object from storage
	obj, err := ctx.Storage.GetObject(bucketName, objName, minio.GetObjectOptions{})
	if err != nil {
		renderError(w, err, "CANT_PROCESS_FILE_ON_STORAGE", http.StatusNotFound)
		return
	}

	// Create buffer for following operations
	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, obj); err != nil {
		renderError(w, err, "CANT_READ_FILE", http.StatusInternalServerError)
		return
	}

	// Resize image or don't
	if ctx.Config.ResizeOnDownload {
		// Get content type
		fileType := http.DetectContentType(buf.Bytes())
		if fileType != MIMEImageJPEG && fileType != MIMEImageGIF && fileType != MIMEImagePNG {
			renderError(w, nil, "INVALID_FILE_TYPE", http.StatusBadRequest)
			return
		}

		result, err = ResizeImage(ctx, buf, fileType)
		if err != nil {
			renderError(w, err, "CANT_RESIZE_FILE", http.StatusInternalServerError)
			return
		}
	} else {
		result = buf
	}

	// Respond with data
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, result); err != nil {
		renderError(w, err, "CANT_RESPOND_FILE", http.StatusInternalServerError)
		return
	}
}

// PrepareDimensions returns dimensions from string input.
// Validate resize value. Valid cases: 100x100, 100 as width
func PrepareDimensions(input string) ([]int, error) {
	// Handle empty input
	if input == "" {
		return []int{}, nil
	}

	// Validate as single value for width
	if matched, _ := regexp.MatchString("^[0-9]+$", input); matched {
		intInput, _ := strconv.Atoi(input)
		return []int{intInput}, nil
	}

	// Validate 100x100 case
	if matched, _ := regexp.MatchString("^[0-9]+x[0-9]+$", input); matched {
		var result []int
		for _, item := range strings.Split(input, "x") {
			intInput, _ := strconv.Atoi(item)
			result = append(result, intInput)
		}
		return result, nil
	}

	return []int{}, errors.New("Input value is invalid")
}

// GenerateFileName generates a new hashed file name.
func GenerateFileName(fileType string) (string, error) {
	hash := generateID()
	fileExt, err := mime.ExtensionsByType(fileType)
	if err != nil {
		return "", err
	}
	if len(fileExt) == 0 {
		return "", errors.New("Can't get file extension from this file type")
	}
	return fmt.Sprintf("%v%s", hash, fileExt[0]), nil
}

func generateID() uuid.UUID {
	return uuid.NewV4()
}
