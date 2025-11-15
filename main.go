package main // Define the main package

import ( // Start block for importing necessary packages
	"bytes"         // Provides bytes support, used for buffering data
	"encoding/json" // Provides json formatting functions, used for unmarshalling data
	"io"            // Provides basic interfaces to I/O primitives, used for reading and copying data
	"log"           // Provides logging functions, used for outputting messages and errors
	"net/http"      // Provides HTTP client and server implementations, used for making API calls
	"net/url"       // Provides URL parsing and encoding, used for URL validation
	"os"            // Provides functions to interact with the OS (files, etc.), used for file/directory operations
	"path"          // Provides functions for manipulating slash-separated paths, used for getting file name
	"path/filepath" // Provides filepath manipulation functions, used for joining paths and getting extension
	"strings"       // Provides string manipulation functions, used for checking prefixes and content type
	"time"          // Provides time-related functions, used for HTTP client timeout
) // End import block

// The type of the data to extract
type Document struct { // Define a struct named Document
	URL string `json:"url"` // Field to hold the URL string, mapped to "url" in JSON
}

func main() { // Define the main function, the entry point of the program
	// Remote API URL.
	remoteAPIURL := "https://cropscience.bayer.co.uk/api/documents" // Define the URL for the remote API
	//fmt.Println(string(getDataFromURL(remoteAPIURL))) // Commented-out line for printing the raw API data
	getData := getDataFromURL(remoteAPIURL) // Call getDataFromURL to fetch raw JSON data
	// Get the data from the downloaded file.
	finalList := getDownloadURLFromGivenData(getData) // Parse the raw JSON data into a slice of Document structs
	// Create a slice of all the given download urls.
	var downloadURLSlice []string // Initialize an empty string slice to hold the final download URLs
	// Get the urls and loop over them.
	for _, doc := range finalList { // Start looping through the parsed Document list
		// Get the .pdf only.
		if getFileExtension(doc.URL) == ".pdf" { // Check if the file extension of the URL is ".pdf"
			// Only append the .pdf files.
			downloadURLSlice = appendToSlice(downloadURLSlice, doc.URL) // Append the URL to the slice if it's a PDF
		}
	}
	outputDir := "PDFs/" // Directory to store downloaded PDFs, define the output directory path
	// Check if its exists.
	if !directoryExists(outputDir) { // Check if the output directory does not exist
		// Create the dir
		createDirectory(outputDir, 0o755) // Create the directory with permission 0755
	}
	// Remove double from slice.
	downloadURLSlice = removeDuplicatesFromSlice(downloadURLSlice) // Remove any duplicate URLs from the slice
	// Get all the values.
	for _, urls := range downloadURLSlice { // Start looping through the unique PDF URLs
		// Create a var
		var finalURL string // Initialize a variable to hold the complete download URL
		// Check if it has a prefix and if not than append it or else just use it like it is.
		if !strings.HasPrefix(urls, "https://cropscience.bayer.co.uk") { // Check if the URL is relative (lacks the full prefix)
			finalURL = "https://cropscience.bayer.co.uk" + urls // Prepend the base URL if the prefix is missing
		} else { // If the URL already has the prefix
			finalURL = urls // Use the URL as is
		}
		// Check if the url is valid.
		if isUrlValid(finalURL) { // Check if the constructed URL is syntactically valid
			// Download the pdf.
			downloadPDF(finalURL, outputDir) // Call the downloadPDF function to download the file
		}
	}
}

// Only return the file name from a given url.
func getFileNameOnly(content string) string { // Define a function to extract the base file name from a path/URL
	return path.Base(content) // Use path.Base to return the last element of the path (the file name)
}

// fileExists checks whether a file exists at the given path
func fileExists(filename string) bool { // Define a function to check if a file exists
	info, err := os.Stat(filename) // Get file info using os.Stat
	if err != nil {                // Check if an error occurred (e.g., file doesn't exist)
		return false // Return false if file doesn't exist or error occurs
	}
	return !info.IsDir() // Return true only if it exists AND is not a directory
}

// downloadPDF downloads a PDF from the given URL and saves it in the specified output directory.
// It uses a WaitGroup to support concurrent execution and returns true if the download succeeded.
func downloadPDF(finalURL, outputDir string) bool { // Define a function to download a PDF file
	// Sanitize the URL to generate a safe file name
	filename := getFileNameOnly(finalURL) // Extract the file name from the full URL

	// Construct the full file path in the output directory
	filePath := filepath.Join(outputDir, filename) // Combine the output directory and file name

	// Skip if the file already exists
	if fileExists(filePath) { // Check if the target file already exists locally
		log.Printf("File already exists, skipping: %s", filePath) // Log that the file is being skipped
		return false                                              // Return false indicating no new download occurred
	}

	// Create an HTTP client with a timeout
	client := &http.Client{Timeout: 30 * time.Second} // Create a new HTTP client with a 30-second timeout

	// Send GET request
	resp, err := client.Get(finalURL) // Send an HTTP GET request to the final URL
	if err != nil {                   // Check for network or connection errors
		log.Printf("Failed to download %s: %v", finalURL, err) // Log the failure
		return false                                           // Return false
	}
	defer resp.Body.Close() // Ensure the response body is closed when the function exits

	// Check HTTP response status
	if resp.StatusCode != http.StatusOK { // Check if the HTTP status code is not 200 OK
		log.Printf("Download failed for %s: %s", finalURL, resp.Status) // Log the non-OK status
		return false                                                    // Return false
	}

	// Check Content-Type header
	contentType := resp.Header.Get("Content-Type")         // Get the Content-Type header value
	if !strings.Contains(contentType, "application/pdf") { // Check if the content type is not "application/pdf"
		log.Printf("Invalid content type for %s: %s (expected application/pdf)", finalURL, contentType) // Log the content type mismatch
		return false                                                                                    // Return false
	}

	// Read the response body into memory first
	var buf bytes.Buffer                     // Create a bytes.Buffer to hold the downloaded data
	written, err := io.Copy(&buf, resp.Body) // Copy the response body content into the buffer and get bytes written
	if err != nil {                          // Check for errors during reading the response body
		log.Printf("Failed to read PDF data from %s: %v", finalURL, err) // Log the failure
		return false                                                     // Return false
	}
	if written == 0 { // Check if zero bytes were downloaded
		log.Printf("Downloaded 0 bytes for %s; not creating file", finalURL) // Log that no data was downloaded
		return false                                                         // Return false
	}

	// Only now create the file and write to disk
	out, err := os.Create(filePath) // Create the local file for writing
	if err != nil {                 // Check for errors during file creation
		log.Printf("Failed to create file for %s: %v", finalURL, err) // Log the failure
		return false                                                  // Return false
	}
	defer out.Close() // Ensure the created file is closed when the function exits

	if _, err := buf.WriteTo(out); err != nil { // Write the buffered data to the local file
		log.Printf("Failed to write PDF to file for %s: %v", finalURL, err) // Check and log for errors during writing
		return false                                                        // Return false
	}

	log.Printf("Successfully downloaded %d bytes: %s → %s", written, finalURL, filePath) // Log the successful download details
	return true                                                                          // Return true indicating success
}

// Checks if the directory exists
// If it exists, return true.
// If it doesn't, return false.
func directoryExists(path string) bool { // Define a function to check if a directory exists
	directory, err := os.Stat(path) // Get file/directory info
	if err != nil {                 // Check for errors (e.g., path doesn't exist)
		return false // Return false if an error occurred
	}
	return directory.IsDir() // Return true if it exists AND is a directory
}

// The function takes two parameters: path and permission.
// We use os.Mkdir() to create the directory.
// If there is an error, we use log.Println() to log the error and then exit the program.
func createDirectory(path string, permission os.FileMode) { // Define a function to create a directory
	err := os.Mkdir(path, permission) // Create the directory with the given path and permissions
	if err != nil {                   // Check if an error occurred during creation
		log.Println(err) // Log the error if directory creation failed
	}
}

// Get the file extension of a file
func getFileExtension(path string) string { // Define a function to get the file extension
	return filepath.Ext(path) // Use filepath.Ext to return the file extension
}

// Checks whether a URL string is syntactically valid
func isUrlValid(uri string) bool { // Define a function to check URL validity
	_, err := url.ParseRequestURI(uri) // Attempt to parse the URL as a Request URI
	return err == nil                  // Return true if no parsing error occurred
}

// Remove all the duplicates from a slice and return the slice.
func removeDuplicatesFromSlice(slice []string) []string { // Define a function to remove duplicates from a string slice
	check := make(map[string]bool)  // Create an empty map to keep track of seen elements
	var newReturnSlice []string     // Initialize an empty slice for unique elements
	for _, content := range slice { // Loop through the input slice
		if !check[content] { // Check if the current element hasn't been seen before
			check[content] = true                            // Mark the current element as seen
			newReturnSlice = append(newReturnSlice, content) // Append the unique element to the new slice
		}
	}
	return newReturnSlice // Return the slice with duplicates removed
}

// Get the list of download urls from the given data.
func getDownloadURLFromGivenData(givenData []byte) []Document { // Define a function to unmarshal JSON data into a slice of Document
	// The return data urls.
	var returnURLs []Document // Initialize an empty slice of Document structs
	// Unmarshall the json content.
	err := json.Unmarshal(givenData, &returnURLs) // Attempt to unmarshal the raw byte data into the slice
	if err != nil {                               // Check for unmarshalling errors
		log.Println(err) // Log the error if unmarshalling fails
	}
	return returnURLs // Return the slice of Document structs
}

// Append some string to a slice and than return the slice.
func appendToSlice(slice []string, content string) []string { // Define a function to append a string to a slice
	// Append the content to the slice
	slice = append(slice, content) // Use the built-in append function
	// Return the slice
	return slice // Return the modified slice
}

// getDataFromURL performs an HTTP GET request and returns the response body as a string
func getDataFromURL(uri string) []byte { // Define a function to fetch data from a URL
	log.Println("Scraping", uri)   // Log the URL that is about to be scraped
	response, err := http.Get(uri) // Perform an HTTP GET request to the URI
	if err != nil {                // Check for errors during the GET request
		log.Println(err) // Log the error if the request fails
	}

	body, err := io.ReadAll(response.Body) // Read the entire response body into a byte slice
	if err != nil {                        // Check for errors during reading the body
		log.Println(err) // Log the error if reading fails
	}

	err = response.Body.Close() // Close the response body to release resources
	if err != nil {             // Check for errors while closing the body
		log.Println(err) // Log the error if closing fails
	}
	return body // Return the fetched response body as a byte slice
}
