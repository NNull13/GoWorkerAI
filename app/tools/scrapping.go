package tools

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

const maxFetchSize = 5 << 20

var httpClient = &http.Client{Timeout: 20 * time.Second}

func executeWebAction(action ToolTask) (string, error) {
	h, ok := webDispatch[action.Key]
	if !ok {
		log.Printf("❌ Unknown tool key: %s\n", action.Key)
		return "", fmt.Errorf("unknown tool key: %s", action.Key)
	}
	return h(action.Parameters)
}

var webDispatch = map[string]func(any) (string, error){
	fetch_html_content: func(p any) (string, error) {
		return withParsed[ExtractAction](p, fetch_html_content, func(a ExtractAction) (string, error) {
			return fetchHTMLContent(a)
		})
	},
	extract_links_html: func(p any) (string, error) {
		return withParsed[ExtractAction](p, extract_links_html, func(a ExtractAction) (string, error) {
			return extractLinks(a)
		})
	},
	extract_text_content: func(p any) (string, error) {
		return withParsed[ExtractAction](p, extract_text_content, func(a ExtractAction) (string, error) {
			return extractTextContent(a)
		})
	},
	extract_meta_tags: func(p any) (string, error) {
		return withParsed[ExtractAction](p, extract_meta_tags, func(a ExtractAction) (string, error) {
			return extractMetaTags(a)
		})
	},
}

func fetchHTMLContent(a ExtractAction) (string, error) {
	if a.URL == "" {
		return "", errors.New("invalid parameters: 'url' is required")
	}
	u, err := url.Parse(a.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return "", fmt.Errorf("invalid url: %s", a.URL)
	}

	req, err := http.NewRequest(http.MethodGet, a.URL, nil)
	if err != nil {
		return "", err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("❌ Error fetching URL content %s: %v\n", a.URL, err)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("⚠️ URL %s returned status code: %d\n", a.URL, resp.StatusCode)
		return "", fmt.Errorf("failed to fetch URL content: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	limited := io.LimitReader(resp.Body, maxFetchSize)
	body, err := io.ReadAll(limited)
	if err != nil {
		log.Printf("❌ Error reading content from URL %s: %v\n", a.URL, err)
		return "", err
	}

	content := string(body)
	if a.FilePath != "" {
		if _, err := writeToFile("", a.FilePath, content); err != nil {
			log.Printf("❌ Error writing fetched HTML to file: %v\n", err)
			return "", err
		}
		log.Printf("✅ Fetched HTML saved to file: %s\n", a.FilePath)
	}
	return content, nil
}

func extractLinks(a ExtractAction) (string, error) {
	if strings.TrimSpace(a.HTML) == "" {
		return "", errors.New("invalid parameters: 'html' is required")
	}
	doc, err := parseHTML(a.HTML)
	if err != nil {
		return "", err
	}

	var links []string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" && attr.Val != "" {
					links = append(links, attr.Val)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	out := strings.Join(links, " ")
	log.Printf("✅ Successfully extracted %d links\n", len(links))
	if a.FilePath != "" {
		return writeToFile("", a.FilePath, out)
	}
	return out, nil
}

func extractTextContent(a ExtractAction) (string, error) {
	if strings.TrimSpace(a.HTML) == "" {
		return "", errors.New("invalid parameters: 'html' is required")
	}
	doc, err := parseHTML(a.HTML)
	if err != nil {
		return "", err
	}

	var parts []string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			t := strings.TrimSpace(n.Data)
			if t != "" {
				parts = append(parts, t)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	out := strings.Join(parts, " ")
	log.Printf("✅ Successfully extracted %d text blocks\n", len(parts))
	if a.FilePath != "" {
		return writeToFile("", a.FilePath, out)
	}
	return out, nil
}

func extractMetaTags(a ExtractAction) (string, error) {
	if strings.TrimSpace(a.HTML) == "" {
		return "", errors.New("invalid parameters: 'html' is required")
	}
	doc, err := parseHTML(a.HTML)
	if err != nil {
		return "", err
	}

	meta := map[string]string{}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if n.Data == "title" && n.FirstChild != nil {
				meta["title"] = strings.TrimSpace(n.FirstChild.Data)
			}
			if n.Data == "meta" {
				var name, content string
				for _, a := range n.Attr {
					if a.Key == "name" {
						name = a.Val
					}
					if a.Key == "content" {
						content = a.Val
					}
				}
				if name != "" {
					meta[name] = content
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	out := fmt.Sprintf("%v", meta)
	log.Printf("✅ Successfully extracted %d meta tags\n", len(meta))
	if a.FilePath != "" {
		return writeToFile("", a.FilePath, out)
	}
	return out, nil
}

func parseHTML(s string) (*html.Node, error) {
	doc, err := html.Parse(strings.NewReader(s))
	if err != nil {
		log.Printf("❌ Error parsing HTML content: %v\n", err)
		return nil, err
	}
	return doc, nil
}
