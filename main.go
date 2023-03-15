package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type RequestParams struct {
	Method          string
	Headers         http.Header
	Body            io.Reader
	FollowRedirects bool
	Auth            *AuthParams
	QueryParams     url.Values
}

type AuthParams struct {
	Username string
	Password string
}

func main() {
	urlStr := flag.String("url", "", "The URL to make the request to")
	method := flag.String("method", "GET", "The HTTP method to use")
	reqBodyFile := flag.String("file", "", "The name of a file to use as the request body")
	followRedirects := flag.Bool("follow", false, "Whether to follow redirects")
	authStr := flag.String("auth", "", "The username and password for basic authentication in the format 'username:password'")
	flag.Parse()

	if *urlStr == "" {
		fmt.Println("Usage: httpreq -url <url> [options]")
		fmt.Println("Options:")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Parse the URL to ensure it's valid
	_, err := url.Parse(*urlStr)
	if err != nil {
		log.Fatalf("Error: Invalid URL '%s': %s", *urlStr, err)
	}

	// Read the request body from file if specified
	var reqBody io.Reader
	if *reqBodyFile != "" {
		file, err := os.Open(*reqBodyFile)
		if err != nil {
			log.Fatalf("Error: Could not read file '%s': %s", *reqBodyFile, err)
		}
		reqBody = file
	}

	// Parse the authentication credentials if specified
	var auth *AuthParams
	if *authStr != "" {
		authParts := strings.SplitN(*authStr, ":", 2)
		if len(authParts) != 2 {
			log.Fatalf("Error: Invalid auth credentials '%s'", *authStr)
		}
		auth = &AuthParams{Username: authParts[0], Password: authParts[1]}
	}

	reqParams := &RequestParams{
		Method:          *method,
		Body:            reqBody,
		FollowRedirects: *followRedirects,
		Auth:            auth,
	}

	// Add headers from command-line options
	for i := 0; i < flag.NArg(); i++ {
		header := flag.Arg(i)
		headerParts := strings.SplitN(header, ":", 2)
		if len(headerParts) != 2 {
			log.Fatalf("Error: Invalid header '%s'", header)
		}
		reqParams.Headers.Set(strings.TrimSpace(headerParts[0]), strings.TrimSpace(headerParts[1]))
	}

	// Create the HTTP request object
	req, err := http.NewRequest(reqParams.Method, *urlStr, reqParams.Body)
	if err != nil {
		log.Fatalf("Error creating request object: %s", err)
	}

	// Add headers to the request object
	for k, v := range reqParams.Headers {
		req.Header.Set(k, v[0])
	}

	// Add authentication to the request object if specified
	if reqParams.Auth != nil {
		authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(reqParams.Auth.Username+":"+reqParams.Auth.Password))
		req.Header.Set("Authorization", authHeader)
	}

	// Create a new HTTP client
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if !reqParams.FollowRedirects {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	// Make the HTTP request
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error making request: %s", err)
	}
	defer resp.Body.Close()

	// Print the response status code
	fmt.Printf("HTTP/%d.%d %s\n", resp.ProtoMajor, resp.ProtoMinor, resp.Status)

	// Print the response headers
	for k, v := range resp.Header {
		fmt.Printf("%s: %s\n", k, v[0])
	}
	fmt.Println()

	// get response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response body: %s", err)
	}
	// Create a map to hold the JSON data
	var data map[string]interface{}

	// Unmarshal the JSON into the map
	err = json.Unmarshal(body, &data)
	if err != nil {
		fmt.Println("Error unmarshaling JSON:", err)
		return
	}

	// Pretty print the JSON
	prettyJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return
	}

	fmt.Println(string(prettyJSON))
}
