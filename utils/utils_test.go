package utils_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/abiiranathan/apigen/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestParseMultipleUploads(t *testing.T) {
	// Create a new Fiber app
	app := fiber.New()

	fileContent := "test file content. it containing long text string"
	fileName := "test.txt"
	fileField := "files"
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(fileField, fileName)
	assert.Nil(t, err)

	// Write the file content to the multipart form.
	_, err = part.Write([]byte(fileContent))
	assert.Nil(t, err)

	// Close the multipart form.
	err = writer.Close()
	assert.Nil(t, err)

	// Mock the static prefix
	staticPrefix := "uploads"

	// Set up the config
	config := utils.Config{
		MaxSize:           10 * 1024, // 10 KB
		ValidTypes:        []string{utils.PlainText},
		InvalidExtensions: []string{utils.Executable},
	}

	// Define the route and handler
	app.Post("/uploads", func(c *fiber.Ctx) error {
		uploads, err := utils.ParseMultipleUploads(c, config, staticPrefix)
		assert.NoError(t, err)

		// Assertions
		assert.Len(t, uploads, 1)
		if len(uploads) != 1 {
			return nil
		}

		assert.Contains(t, uploads, fileField)
		assert.Len(t, uploads[fileField], 1)

		upload := uploads[fileField][0]
		assert.Equal(t, fileName, upload.Title)
		assert.Contains(t, upload.FsPath, staticPrefix)
		assert.Equal(t, int64(len(fileContent)), upload.Size)
		assert.NotNil(t, upload.FileHeader)
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/uploads", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Perform the request using app.Test
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
