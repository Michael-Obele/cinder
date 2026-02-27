package image

import (
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/standard-user/cinder/internal/domain"
)

// ExtractPageImages parses HTML and extracts image metadata.
// It prioritizes OG images, Twitter card images, then content images.
func ExtractPageImages(htmlBody string, pageURL string, maxImages int) []domain.ImageData {
	if maxImages <= 0 {
		maxImages = 10
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlBody))
	if err != nil {
		return nil
	}

	var images []domain.ImageData
	seen := make(map[string]bool)

	// 1. OG Image (highest priority for AI consumption)
	if ogImage, exists := doc.Find(`meta[property="og:image"]`).Attr("content"); exists {
		absURL := resolveURL(ogImage, pageURL)
		if absURL != "" && !seen[absURL] {
			images = append(images, domain.ImageData{
				URL:        absURL,
				SourceType: "og:image",
			})
			seen[absURL] = true
		}
	}

	// 2. Twitter card image
	doc.Find(`meta[property="twitter:image"], meta[name="twitter:image"]`).Each(func(i int, s *goquery.Selection) {
		if content, exists := s.Attr("content"); exists {
			absURL := resolveURL(content, pageURL)
			if absURL != "" && !seen[absURL] {
				images = append(images, domain.ImageData{
					URL:        absURL,
					SourceType: "twitter:image",
				})
				seen[absURL] = true
			}
		}
	})

	// 3. Content images
	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		if len(images) >= maxImages {
			return
		}

		src, exists := s.Attr("src")
		if !exists || src == "" {
			return
		}

		absURL := resolveURL(src, pageURL)
		if absURL == "" || seen[absURL] {
			return
		}

		if isTrackingPixel(absURL) {
			return
		}

		alt, _ := s.Attr("alt")
		title, _ := s.Attr("title")

		images = append(images, domain.ImageData{
			URL:        absURL,
			Alt:        alt,
			Title:      title,
			SourceType: "content",
		})
		seen[absURL] = true
	})

	return images
}

func resolveURL(rawURL, pageURL string) string {
	if strings.HasPrefix(rawURL, "data:") {
		return ""
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	if parsed.IsAbs() {
		return rawURL
	}

	base, err := url.Parse(pageURL)
	if err != nil {
		return ""
	}

	return base.ResolveReference(parsed).String()
}

func isTrackingPixel(imgURL string) bool {
	trackers := []string{
		"pixel", "tracking", "beacon", "analytics",
		"1x1", "spacer", "blank",
	}
	lower := strings.ToLower(imgURL)
	for _, t := range trackers {
		if strings.Contains(lower, t) {
			return true
		}
	}
	return false
}
