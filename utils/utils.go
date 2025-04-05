package utils

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"slices"

	"github.com/gofiber/fiber/v2"
)

var (
	ErrUnSupportedMediaType     = fiber.NewError(http.StatusUnsupportedMediaType, "unsupported media type")
	ErrEmptyFile                = fiber.NewError(http.StatusUnsupportedMediaType, "empty file")
	ErrUnSupportedFileExtension = fiber.NewError(http.StatusBadRequest, "unsupported file extension")
)

var (

	// Image MIME Types
	ImageGIF  = "image/gif"
	ImageJPEG = "image/jpeg"
	ImagePNG  = "image/png"
	ImageBMP  = "image/bmp"
	ImageTIFF = "image/tiff"
	ImageWebP = "image/webp"
	ImageAVIF = "image/avif"

	// Document MIME Types
	DocumentPDF  = "application/pdf"
	DocumentDOCX = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	DocumentDOC  = "application/msword"
	DocumentXLS  = "application/vnd.ms-excel"
	DocumentXLSX = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	DocumentPPT  = "application/vnd.ms-powerpoint"
	DocumentPPTX = "application/vnd.openxmlformats-officedocument.presentationml.presentation"

	// Compressed MIME Types
	CompressedGZIP  = "application/gzip"
	CompressedBZIP2 = "application/x-bzip2"
	CompressedRar   = "application/x-rar"
	CompressedZip   = "application/zip"
	Compressed7Z    = "application/x-7z-compressed"

	// Audo formats
	AudioAAC  = "audio/aac"
	AudioFLAC = "audio/flac"
	AudioMP3  = "audio/mpeg"
	AudioOGG  = "audio/ogg"
	AudioWAV  = "audio/x-wav"
	AudioWMA  = "audio/x-ms-wma"

	// Video formats
	VideoMP4 = "video/mp4"
	VideoAVI = "video/x-msvideo"
	VideoMKV = "video/x-matroska"
	VideoMOV = "video/quicktime"
	VideoMPG = "video/mpeg"
	VideoWMV = "video/x-ms-wmv"

	// Plain Text MIME Types
	PlainText = "text/plain"

	// Executable Formats
	Executable = "application/octet-stream"

	// Font Formats
	FontOpenType = "application/vnd.ms-opentype"
)

// Config allows you to pass options for validating uploads.
type Config struct {
	MaxSize               int64    // Maximum file size in the uploads
	ValidTypes            []string // slice of mime types that are allowed.
	InvalidExtensions     []string // slice of file extensions not allowed
	DisableSizeCheck      bool     // Whether to perfom checks on file size
	DisableMimeCheck      bool     // Whether to perform checks on mimetypes
	DisableExtensionCheck bool     // Whether to perform check of file extensions
}

// Stores information about an uploaded file.
type Upload struct {
	Title  string `json:"title"`   // defaults to the original filename
	DbPath string `json:"db_path"` // database path stripped of file system infomation
	FsPath string `json:"fs_path"` // filesystem path where to save the upload
	Size   int64  `json:"size"`    // fileSize in bytes

	FileHeader *multipart.FileHeader `json:"file_header"` // pointer to fileHeader.
}

/*
Extracts uploads from the multipart form.
Only the specified file types are allowed.

maxSize: Maximum allowed size for an upload in bytes.

validTypes: List of allowed file types.

invalidExtensions: Explicit file extensions that are not allowed. This caters for
files with complex mimetypes. e.g .exe, .bat, .msi etc

staticPrefix: Prefix where the files are to be stored. This is the string you used for
app.Static e.g form app.Static("/uploads",....) staticPrefix is `uploads`.

Returns: A map of slices of uploads with each upload containing the title, database path and file system path.
The key for the map are the form field names. This means you can upload multiple files
with different and similar field names and they will be processed properly.

Usage:

	// fiber router to handle uploads
	app.Post("/uploads", func(c *fiber.Ctx) error {
		uploads, err := utils.ParseMultipleUploads(c, utils.DefaultConfig, "uploads")
		if err != nil {
			return err
		}

		fmt.Println(uploads)
		return c.JSON(uploads)
	})
*/
func ParseMultipleUploads(c *fiber.Ctx, config Config, staticPrefix string) (map[string][]Upload, error) {
	// Parse the multipart form data from the request
	form, err := c.MultipartForm()
	if err != nil {
		return nil, err
	}

	if config.MaxSize == 0 {
		config.DisableSizeCheck = true
	}
	if len(config.ValidTypes) == 0 {
		config.DisableMimeCheck = true
	}
	if len(config.InvalidExtensions) == 0 {
		config.DisableExtensionCheck = true
	}

	uploads := make(map[string][]Upload, len(form.File))
	for fieldName, fileHeaders := range form.File {
		// Initialize slice for each fieldName with capacity equal to the number of files it contains
		if _, exists := uploads[fieldName]; !exists {
			uploads[fieldName] = make([]Upload, 0, len(fileHeaders))
		}

		for _, fileHeader := range fileHeaders {
			if !config.DisableSizeCheck && fileHeader.Size > config.MaxSize {
				return nil, fmt.Errorf("file %s [%d Bytes] exceeds maximum allowed size %d Bytes",
					fileHeader.Filename, fileHeader.Size, config.MaxSize)
			}

			file, err := fileHeader.Open()
			if err != nil {
				return nil, err
			}
			defer file.Close()

			if !config.DisableExtensionCheck && !isValidExtension(fileHeader, config.InvalidExtensions) {
				return nil, ErrUnSupportedFileExtension
			}

			// Determine the size of the chunk to read for packet sniffing
			bufLen := min(fileHeader.Size, int64(512))

			// Create buffer to hold sniffed packets
			header := make([]byte, bufLen)
			if _, err = file.Read(header); err != nil {
				return nil, fmt.Errorf("can not read file header")
			}

			contentType, valid := isValidFileType(fileHeader, header, config.ValidTypes)
			if !config.DisableMimeCheck && !valid {
				return nil, fmt.Errorf("%w: %s", ErrUnSupportedMediaType, contentType)
			}

			dbPath, fsPath := generateFilePaths(fileHeader, staticPrefix)
			uploads[fieldName] = append(uploads[fieldName], Upload{
				Title:      fileHeader.Filename,
				DbPath:     dbPath,
				FsPath:     fsPath,
				Size:       fileHeader.Size,
				FileHeader: fileHeader,
			})
		}

	}
	return uploads, nil
}

func isValidExtension(fileHeader *multipart.FileHeader, invalidExtensions []string) bool {
	if len(invalidExtensions) == 0 {
		return true
	}

	ext := filepath.Ext(fileHeader.Filename)
	return !contains(invalidExtensions, ext)
}

// Check if string slice s contains e.
func contains(s []string, e string) bool {
	return slices.Contains(s, e)
}

/*
Checks if the file type of the given file header is valid.
*/
func isValidFileType(fileHeader *multipart.FileHeader, sniffedBytes []byte, validTypes []string) (string, bool) {
	if fileHeader.Size == 0 {
		return "", false
	}

	// Some times this return also the encoding which we are not interested in
	contentType := http.DetectContentType(sniffedBytes)
	parts := strings.SplitN(contentType, ";", 2)
	if len(parts) > 1 {
		contentType = strings.TrimSpace(parts[0])
	}

	if slices.Contains(validTypes, contentType) {
		return contentType, true
	}
	return contentType, false
}

/*
Generates file system and database paths for the given file header and timestamp.
*/
func generateFilePaths(fileHeader *multipart.FileHeader, staticPrefix string) (string, string) {
	ts := strconv.FormatInt(time.Now().UnixNano(), 10)
	baseName := fmt.Sprintf("%s-%s", ts, fileHeader.Filename)
	dbPath := fmt.Sprintf("/%s/%s", staticPrefix, baseName)
	workingDir, _ := os.Getwd()
	fsPath := filepath.Join(workingDir, staticPrefix, baseName)
	return dbPath, fsPath
}

// Saves multipart files to disk using c.SaveFile.
func SaveMultipartFiles(c *fiber.Ctx, uploads []Upload) (err error) {
	for _, upload := range uploads {
		if err = c.SaveFile(upload.FileHeader, upload.FsPath); err != nil {
			return err
		}
	}
	return nil
}
