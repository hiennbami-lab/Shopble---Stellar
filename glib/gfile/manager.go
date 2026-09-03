package gfile

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/csv"
	"shopble/common/comerr"
	"shopble/common/comutils"
	"io"
	"regexp"
	"strings"

	libDocumentRead "github.com/go-shiori/go-readability"
	"github.com/gocolly/colly/v2"
	"github.com/ledongthuc/pdf"
)

type Encoder func([]byte) string

type Option struct {
	encoder                      Encoder
	extractedLinksLookupKeywords []string
}

type ExtractedLink struct {
	URL  string `json:"url"`
	Type string `json:"type"` // pdf, google_sheet, google_doc, excel, word, etc.
	Text string `json:"text"` // anchor text
}

type WebLinkResult struct {
	Content        string          `json:"content"`
	ExtractedLinks []ExtractedLink `json:"extracted_links,omitempty"`
}

func EncodeContentFromTxtFile(ctx context.Context, txtFile io.Reader, optSetters ...func(*Option)) (_ string, err error) {
	var option Option
	for _, setter := range optSetters {
		setter(&option)
	}

	content, err := io.ReadAll(txtFile)
	if err != nil {
		err = comerr.WrapStack(err, "txt: read content failed")
		return
	}

	if option.encoder != nil {
		return option.encoder(content), nil
	}
	return base64.StdEncoding.EncodeToString(content), nil
}

func EncodeContentFromCsv(ctx context.Context, csvFile io.Reader, optSetters ...func(*Option)) (_ string, err error) {
	reader := csv.NewReader(csvFile)
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = false

	records, err := reader.ReadAll()
	if err != nil {
		err = comerr.WrapStack(err, "csv: read records failed")
		return
	}

	var buffer bytes.Buffer
	for _, record := range records {
		for colIndex, field := range record {
			// Write field with preserved formatting
			if colIndex > 0 {
				buffer.WriteString("\t") // Tab separator between columns
			}
			// Unescape literal \n and \t sequences if they exist
			field = strings.ReplaceAll(field, "\\n", "\n")
			field = strings.ReplaceAll(field, "\\t", "\t")
			buffer.WriteString(field)
		}
		buffer.WriteString("\n") // New line after each row
	}

	return base64.StdEncoding.EncodeToString(buffer.Bytes()), nil
}

var (
	newlineRegex = regexp.MustCompile(`[\n\r]{3,}`)          // 3 or more newlines
	spaceRegex   = regexp.MustCompile(`[ \t]{2,}`)           // 2+ spaces/tabs
	lineTrim     = regexp.MustCompile(`(?m)^[ \t]+|[ \t]+$`) // trim leading/trailing whitespace per line
	specialCodes = regexp.MustCompile(`&(?:#\d+|#x[0-9A-Fa-f]+|[A-Za-z][A-Za-z0-9]+);`)

	// Patterns for downloadable/external resource links
	downloadableLinkPatterns = map[string]*regexp.Regexp{
		"pdf":          regexp.MustCompile(`(?i)\.pdf(\?.*)?$`),
		"google_sheet": regexp.MustCompile(`(?i)docs\.google\.com/spreadsheets/`),
		"google_doc":   regexp.MustCompile(`(?i)docs\.google\.com/document/`),
		"google_drive": regexp.MustCompile(`(?i)drive\.google\.com/`),
		"excel":        regexp.MustCompile(`(?i)\.(xlsx?|csv)(\?.*)?$`),
		"word":         regexp.MustCompile(`(?i)\.(docx?)(\?.*)?$`),
		"powerpoint":   regexp.MustCompile(`(?i)\.(pptx?)(\?.*)?$`),
		"zip":          regexp.MustCompile(`(?i)\.(zip|rar|7z|tar\.gz)(\?.*)?$`),
	}
)

func cleanArticleContent(content string) string {
	// Split into paragraphs for better processing
	paragraphs := strings.Split(content, "\n")
	var cleanParagraphs []string

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		para = lineTrim.ReplaceAllString(para, "")
		para = newlineRegex.ReplaceAllString(para, "\n\n")
		para = spaceRegex.ReplaceAllString(para, " ")
		para = specialCodes.ReplaceAllString(para, "")
		if para == "" || len(para) < 50 {
			continue
		}

		// use regexp.Compile instead of MatchString
		matcher, _ := regexp.Compile(`^(\s*-\s*)?Ảnh:`)
		if matched := matcher.MatchString(para); matched {
			continue
		}

		// Skip short paragraphs that are likely captions or metadata
		if len(para) < 40 && (strings.Contains(para, ":") || strings.HasPrefix(para, "-")) {
			continue
		}

		src_matcher, _ := regexp.Compile(`^(Source|Nguồn):`)
		// Skip paragraphs that look like source citations
		if matched := src_matcher.MatchString(para); matched {
			continue
		}

		cleanParagraphs = append(cleanParagraphs, para)
	}

	return strings.Join(cleanParagraphs, "\n")
}

func EncodeContentFromPdfUrl(ctx context.Context, pdfURL string, optSetters ...func(*Option)) (_ string, err error) {
	if pdfURL == "" {
		return "", comerr.WrapStack(comerr.ErrorDataInvalid, "pdf url is empty")
	}

	var (
		pdfCollector = colly.NewCollector()
		callbackErr  error
		pdfContent   []byte
	)

	pdfCollector.OnError(func(r *colly.Response, err error) {
		if err == nil {
			return
		}
		switch r.StatusCode {
		case 403, 404, 0:
			callbackErr = comerr.ErrorNotFound.Wrap(err)
		default:
			callbackErr = comerr.WrapStack(err, "download pdf failed")
		}
	})

	pdfCollector.OnResponse(func(r *colly.Response) {
		contentType := r.Headers.Get("Content-Type")
		if !strings.Contains(contentType, "application/pdf") && !strings.HasSuffix(strings.ToLower(pdfURL), ".pdf") {
			callbackErr = comerr.WrapStack(comerr.ErrorDataInvalid, "url is not a pdf file").WithField("content_type", contentType)
			return
		}
		pdfContent = r.Body
	})

	if visitErr := pdfCollector.Visit(pdfURL); visitErr != nil {
		return "", comerr.WrapStack(visitErr, "visit pdf url failed").WithField("url", pdfURL)
	}
	if callbackErr != nil {
		return "", callbackErr
	}

	if len(pdfContent) == 0 {
		return "", comerr.WrapStack(comerr.ErrorDataInvalid, "pdf content is empty")
	}

	return EncodeContentFromPdf(ctx, bytes.NewReader(pdfContent), optSetters...)
}

func EncodeContentFromPdf(ctx context.Context, pdfFile io.Reader, optSetters ...func(*Option)) (_ string, err error) {
	var option Option
	for _, setter := range optSetters {
		setter(&option)
	}

	// Read all content into memory to get a ReaderAt
	pdfBytes, err := io.ReadAll(pdfFile)
	if err != nil {
		return "", comerr.WrapStack(err, "failed to read pdf content")
	}

	reader, err := pdf.NewReader(bytes.NewReader(pdfBytes), int64(len(pdfBytes)))
	if err != nil {
		return "", comerr.WrapStack(err, "failed to read PDF")
	}

	var buf bytes.Buffer
	for i := 1; i <= reader.NumPage(); i++ {
		page := reader.Page(i)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			continue // skip pages that can't be read
		}
		buf.WriteString(text)
		buf.WriteString("\n")
	}

	content := cleanPdfContent(buf.String())
	if option.encoder != nil {
		return option.encoder([]byte(content)), nil
	}
	return content, nil
}

var (
	// Regex patterns for cleaning PDF content
	multipleSpaces = regexp.MustCompile(`[ \t]{2,}`)
)

func cleanPdfContent(content string) string {
	// Normalize line endings
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	// Split into lines for processing
	lines := strings.Split(content, "\n")
	var result []string
	var currentLine strings.Builder

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			// If we have accumulated content, save it
			if currentLine.Len() > 0 {
				result = append(result, currentLine.String())
				currentLine.Reset()
			}
			continue
		}

		// If line is very short (likely a broken word/phrase from PDF table)
		// append to current line with space
		if len(line) < 30 && currentLine.Len() > 0 {
			currentLine.WriteString(" ")
			currentLine.WriteString(line)
		} else if currentLine.Len() > 0 {
			// Longer line - save current and start new
			result = append(result, currentLine.String())
			currentLine.Reset()
			currentLine.WriteString(line)
		} else {
			currentLine.WriteString(line)
		}
	}

	// Don't forget the last line
	if currentLine.Len() > 0 {
		result = append(result, currentLine.String())
	}

	// Join with single newline and clean up extra spaces
	content = strings.Join(result, "\n")
	content = multipleSpaces.ReplaceAllString(content, " ")
	content = strings.TrimSpace(content)

	return content
}

func EncodeBase64() func(*Option) {
	return func(o *Option) {
		o.encoder = func(b []byte) string {
			return base64.StdEncoding.EncodeToString([]byte(b))
		}
	}
}

func EncodeRawBase64() func(*Option) {
	return func(o *Option) {
		o.encoder = func(b []byte) string {
			return base64.RawURLEncoding.EncodeToString([]byte(b))
		}
	}
}

func ExtractContentLinks(lookupKeywords []string) func(*Option) {
	return func(o *Option) {
		o.extractedLinksLookupKeywords = lookupKeywords
	}
}

func EncodeContentFromWebLink(ctx context.Context, webLink string, optSetters ...func(*Option)) (_ *WebLinkResult, err error) {
	if webLink == "" {
		err = comerr.WrapStack(comerr.ErrorDataInvalid, "web link is empty")
		return
	}

	var (
		option Option
	)
	for _, setter := range optSetters {
		setter(&option)
	}

	var (
		htmlCollector  = colly.NewCollector()
		callbackErr    error
		extractedLinks []ExtractedLink

		content string
	)

	htmlCollector.OnError(func(r *colly.Response, err error) {
		if err == nil {
			return
		}
		switch r.StatusCode {
		case 403, 404, 0:
			callbackErr = comerr.ErrorNotFound.Wrap(err)
		}
	})

	if len(option.extractedLinksLookupKeywords) > 0 {
		htmlCollector.OnHTML("a[href]", func(e *colly.HTMLElement) {
			href := e.Attr("href")
			if href == "" {
				return
			}

			absoluteURL := e.Request.AbsoluteURL(href)
			if absoluteURL == "" {
				return
			}

			for linkType, pattern := range downloadableLinkPatterns {
				if pattern.MatchString(absoluteURL) {
					for _, keyword := range option.extractedLinksLookupKeywords {
						if strings.Contains(strings.ToLower(strings.TrimSpace(e.Text)), keyword) {
							extractedLinks = append(extractedLinks, ExtractedLink{
								URL:  absoluteURL,
								Type: linkType,
								Text: strings.TrimSpace(e.Text),
							})
						}
					}
					break
				}
			}
		})
	}

	htmlCollector.OnResponse(func(r *colly.Response) {
		var article libDocumentRead.Article
		article, callbackErr = libDocumentRead.FromReader(bytes.NewBuffer(r.Body), nil)
		if callbackErr != nil {
			return
		}
		content = cleanArticleContent(article.TextContent)
	})

	if err := htmlCollector.Visit(webLink); err != nil {
		if !comutils.IsSameError(callbackErr, comerr.ErrorNotFound) {
			err = comerr.WrapStack(err, "visit link failed").WithField("link", webLink)
			return nil, err
		}
		callbackErr = nil
	}
	if callbackErr != nil {
		err = comerr.WrapStack(callbackErr, "extract content err").WithField("link", webLink)
		return nil, err
	}

	result := &WebLinkResult{
		Content:        content,
		ExtractedLinks: extractedLinks,
	}

	if option.encoder != nil {
		result.Content = option.encoder([]byte(content))
	}
	return result, nil
}
