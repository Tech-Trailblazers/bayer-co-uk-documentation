package main // Define the main package

import (
	"bytes"         // Provides bytes support
	"encoding/json" // Provides json formatting functions
	"io"            // Provides basic interfaces to I/O primitives
	"log"           // Provides logging functions
	"net/http"      // Provides HTTP client and server implementations
	"net/url"       // Provides URL parsing and encoding
	"os"            // Provides functions to interact with the OS (files, etc.)
	"path"          // Provides functions for manipulating slash-separated paths
	"path/filepath" // Provides filepath manipulation functions
	"strings"       // Provides string manipulation functions
	"time"          // Provides time-related functions
)

// The type of the data to extract
type Document struct {
	URL string `json:"url"`
}

func main() {
	// Remote API URL.
	remoteAPIURL := "https://cropscience.bayer.co.uk/api/documents"
	//fmt.Println(string(getDataFromURL(remoteAPIURL)))
	getData := getDataFromURL(remoteAPIURL)
	// Get the data from the downloaded file.
	finalList := getDownloadURLFromGivenData(getData)
	// Create a slice of all the given download urls.
	var downloadURLSlice []string
	// Get the urls and loop over them.
	for _, doc := range finalList {
		// Get the .pdf only.
		if getFileExtension(doc.URL) == ".pdf" {
			// Only append the .pdf files.
			downloadURLSlice = appendToSlice(downloadURLSlice, doc.URL)
		}
	}
	outputDir := "PDFs/" // Directory to store downloaded PDFs
	// Check if its exists.
	if !directoryExists(outputDir) {
		// Create the dir
		createDirectory(outputDir, 0o755)
	}
	// Remove double from slice.
	downloadURLSlice = removeDuplicatesFromSlice(downloadURLSlice)
	// Get all the values.
	for _, urls := range downloadURLSlice {
		// Create a var
		var finalURL string
		// Check if it has a prefix and if not than append it or else just use it like it is.
		if !strings.HasPrefix(urls, "https://cropscience.bayer.co.uk") {
			finalURL = "https://cropscience.bayer.co.uk" + urls
		}
		// Check if the url is valid.
		if isUrlValid(finalURL) {
			// Download the pdf.
			downloadPDF(finalURL, outputDir)
		}
	}
}

// Only return the file name from a given url.
func getFileNameOnly(content string) string {
	return path.Base(content)
}

// fileExists checks whether a file exists at the given path
func fileExists(filename string) bool {
	info, err := os.Stat(filename) // Get file info
	if err != nil {
		return false // Return false if file doesn't exist or error occurs
	}
	return !info.IsDir() // Return true if it's a file (not a directory)
}

// downloadPDF downloads a PDF from the given URL and saves it in the specified output directory.
// It uses a WaitGroup to support concurrent execution and returns true if the download succeeded.
func downloadPDF(finalURL, outputDir string) bool {
	// Sanitize the URL to generate a safe file name
	filename := getFileNameOnly(finalURL)

	// Construct the full file path in the output directory
	filePath := filepath.Join(outputDir, filename)

	// Skip if the file already exists
	if fileExists(filePath) {
		log.Printf("File already exists, skipping: %s", filePath)
		return false
	}

	// Create an HTTP client with a timeout
	client := &http.Client{Timeout: 30 * time.Second}

	// Send GET request
	resp, err := client.Get(finalURL)
	if err != nil {
		log.Printf("Failed to download %s: %v", finalURL, err)
		return false
	}
	defer resp.Body.Close()

	// Check HTTP response status
	if resp.StatusCode != http.StatusOK {
		log.Printf("Download failed for %s: %s", finalURL, resp.Status)
		return false
	}

	// Check Content-Type header
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/pdf") {
		log.Printf("Invalid content type for %s: %s (expected application/pdf)", finalURL, contentType)
		return false
	}

	// Read the response body into memory first
	var buf bytes.Buffer
	written, err := io.Copy(&buf, resp.Body)
	if err != nil {
		log.Printf("Failed to read PDF data from %s: %v", finalURL, err)
		return false
	}
	if written == 0 {
		log.Printf("Downloaded 0 bytes for %s; not creating file", finalURL)
		return false
	}

	// Only now create the file and write to disk
	out, err := os.Create(filePath)
	if err != nil {
		log.Printf("Failed to create file for %s: %v", finalURL, err)
		return false
	}
	defer out.Close()

	if _, err := buf.WriteTo(out); err != nil {
		log.Printf("Failed to write PDF to file for %s: %v", finalURL, err)
		return false
	}

	log.Printf("Successfully downloaded %d bytes: %s → %s", written, finalURL, filePath)
	return true
}

// Checks if the directory exists
// If it exists, return true.
// If it doesn't, return false.
func directoryExists(path string) bool {
	directory, err := os.Stat(path)
	if err != nil {
		return false
	}
	return directory.IsDir()
}

// The function takes two parameters: path and permission.
// We use os.Mkdir() to create the directory.
// If there is an error, we use log.Println() to log the error and then exit the program.
func createDirectory(path string, permission os.FileMode) {
	err := os.Mkdir(path, permission)
	if err != nil {
		log.Println(err)
	}
}

// Get the file extension of a file
func getFileExtension(path string) string {
	return filepath.Ext(path)
}

// Checks whether a URL string is syntactically valid
func isUrlValid(uri string) bool {
	_, err := url.ParseRequestURI(uri) // Attempt to parse the URL
	return err == nil                  // Return true if no error occurred
}

// Remove all the duplicates from a slice and return the slice.
func removeDuplicatesFromSlice(slice []string) []string {
	check := make(map[string]bool)
	var newReturnSlice []string
	for _, content := range slice {
		if !check[content] {
			check[content] = true
			newReturnSlice = append(newReturnSlice, content)
		}
	}
	return newReturnSlice
}

// Get the list of download urls from the given data.
func getDownloadURLFromGivenData(givenData []byte) []Document {
	// The return data urls.
	var returnURLs []Document
	// Unmarshall the json content.
	err := json.Unmarshal(givenData, &returnURLs)
	if err != nil {
		log.Println(err) // Exit if read fails
	}
	return returnURLs
}

// Append some string to a slice and than return the slice.
func appendToSlice(slice []string, content string) []string {
	// Append the content to the slice
	slice = append(slice, content)
	// Return the slice
	return slice
}

// getDataFromURL performs an HTTP GET request and returns the response body as a string
func getDataFromURL(uri string) []byte {
	log.Println("Scraping", uri)   // Log the URL being scraped
	response, err := http.Get(uri) // Perform GET request
	if err != nil {
		log.Println(err) // Exit if request fails
	}

	body, err := io.ReadAll(response.Body) // Read response body
	if err != nil {
		log.Println(err) // Exit if read fails
	}

	err = response.Body.Close() // Close response body
	if err != nil {
		log.Println(err) // Exit if close fails
	}
	return body
}
