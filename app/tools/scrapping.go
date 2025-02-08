package tools

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"golang.org/x/net/html"

	"GoWorkerAI/app/utils"
)

func fetchHTMLContent(task ToolTask) (string, error) {
	action, err := utils.CastAny[ExtractAction](task.Parameters)
	if err != nil || action.URL == "" {
		return "", errors.New("invalid parameters: 'url' is required")
	}

	resp, err := http.Get(action.URL)
	if err != nil {
		log.Printf("❌ Error fetching URL content %s: %v\n", action.URL, err)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("⚠️ URL %s returned status code: %d\n", action.URL, resp.StatusCode)
		return "", fmt.Errorf("failed to fetch URL content: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("❌ Error reading content from URL %s: %v\n", action.URL, err)
		return "", err
	}

	contentStr := string(body)
	if action.FilePath != "" {
		if _, writeErr := writeToFile("", action.FilePath, contentStr); writeErr != nil {
			log.Printf("❌ Error writing fetched HTML to file: %v\n", writeErr)
			return "", writeErr
		}
		log.Printf("✅ Fetched HTML saved to file: %s\n", action.FilePath)
	}

	return contentStr, nil
}

func extractLinks(task ToolTask) (string, error) {
	action, err := utils.CastAny[ExtractAction](task.Parameters)
	if err != nil || action.HTML == "" {
		return "", errors.New("invalid parameters: 'html' is required")
	}

	doc, err := html.Parse(strings.NewReader(action.HTML))
	if err != nil {
		log.Printf("❌ Error parsing HTML content: %v\n", err)
		return "", err
	}

	var links []string
	var extract func(*html.Node)
	extract = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					links = append(links, attr.Val)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}
	extract(doc)

	linksStr := strings.Join(links, " ")
	log.Printf("✅ Successfully extracted %d links\n", len(links))

	if action.FilePath != "" {
		return writeToFile("", action.FilePath, linksStr)
	}

	return linksStr, nil
}

func extractTextContent(task ToolTask) (string, error) {
	action, err := utils.CastAny[ExtractAction](task.Parameters)
	if err != nil || action.HTML == "" {
		return "", errors.New("invalid parameters: 'html' is required")
	}

	doc, err := html.Parse(strings.NewReader(action.HTML))
	if err != nil {
		log.Printf("❌ Error parsing HTML content: %v\n", err)
		return "", err
	}

	var textContent []string
	var extractText func(*html.Node)
	extractText = func(n *html.Node) {
		if n.Type == html.TextNode {
			trimmed := strings.TrimSpace(n.Data)
			if len(trimmed) > 0 {
				textContent = append(textContent, trimmed)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extractText(c)
		}
	}
	extractText(doc)

	result := strings.Join(textContent, " ")
	log.Printf("✅ Successfully extracted %d text blocks\n", len(textContent))

	if action.FilePath != "" {
		return writeToFile("", action.FilePath, result)
	}

	return result, nil
}

func extractMetaTags(task ToolTask) (string, error) {
	action, err := utils.CastAny[ExtractAction](task.Parameters)
	if err != nil || action.HTML == "" {
		return "", errors.New("invalid parameters: 'html' is required")
	}

	doc, err := html.Parse(strings.NewReader(action.HTML))
	if err != nil {
		log.Printf("❌ Error parsing HTML content: %v\n", err)
		return "", err
	}

	metaData := make(map[string]string)
	var extract func(*html.Node)
	extract = func(n *html.Node) {
		if n.Type == html.ElementNode && (n.Data == "meta" || n.Data == "title") {
			if n.Data == "title" && n.FirstChild != nil {
				metaData["title"] = n.FirstChild.Data
			} else if n.Data == "meta" {
				var name, content string
				for _, attr := range n.Attr {
					if attr.Key == "name" {
						name = attr.Val
					} else if attr.Key == "content" {
						content = attr.Val
					}
				}
				if name != "" {
					metaData[name] = content
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}
	extract(doc)

	result := fmt.Sprintf("%v", metaData)
	log.Printf("✅ Successfully extracted %d meta tags\n", len(metaData))

	if action.FilePath != "" {
		return writeToFile("", action.FilePath, result)
	}

	return result, nil
}
