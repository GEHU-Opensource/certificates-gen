package pdf

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

type HTMLGenerator struct {
	templatesDir string
	browser      *rod.Browser
}

type CertificateData struct {
	Name           string
	StudentID      string
	Course         string
	Event          string
	Club           string
	Date           string
	SideDesignImage string
	OrgLogo        string
	ClubLogo       string
	Signature1Image string
	Signature2Image string
	Signature3Image string
	Signature4Image string
	Signer1Title   string
	Signer2Title   string
	Signer3Title   string
	Signer4Title   string
}

func NewHTMLGenerator(templatesDir string) (*HTMLGenerator, error) {
	launcher := launcher.New().
		Headless(true).
		Set("disable-gpu").
		Set("no-sandbox").
		Set("disable-dev-shm-usage")
	
	url, err := launcher.Launch()
	if err != nil {
		return nil, fmt.Errorf("failed to launch browser: %w", err)
	}

	browser := rod.New().ControlURL(url)
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to browser: %w", err)
	}

	return &HTMLGenerator{
		templatesDir: templatesDir,
		browser:      browser,
	}, nil
}

func (g *HTMLGenerator) Close() error {
	if g.browser != nil {
		return g.browser.Close()
	}
	return nil
}

func (g *HTMLGenerator) Generate(data map[string]string) ([]byte, error) {
	return g.GenerateWithTemplate("certificate.html", data)
}

func (g *HTMLGenerator) GenerateWithTemplate(templateName string, data map[string]string) ([]byte, error) {
	templatePath := filepath.Join(g.templatesDir, templateName)
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	certData := g.prepareDataWithImages(data)

	var htmlBuf bytes.Buffer
	if err := tmpl.Execute(&htmlBuf, certData); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	htmlContent := htmlBuf.String()

	page := g.browser.MustPage()
	defer page.MustClose()

	page.MustSetDocumentContent(htmlContent)
	page.MustWaitLoad()
	page.MustWaitStable()
	
	page.MustEval(`() => {
		return document.fonts.ready;
	}`)
	
	page.MustEval(`() => new Promise(resolve => setTimeout(resolve, 300))`)

	paperWidth := 8.27
	paperHeight := 11.69
	marginTop := 0.0
	marginRight := 0.0
	marginBottom := 0.0
	marginLeft := 0.0

	stream, err := page.PDF(&proto.PagePrintToPDF{
		PaperWidth:        &paperWidth,
		PaperHeight:       &paperHeight,
		MarginTop:         &marginTop,
		MarginRight:       &marginRight,
		MarginBottom:      &marginBottom,
		MarginLeft:        &marginLeft,
		PrintBackground:   true,
		PreferCSSPageSize: false,
		DisplayHeaderFooter: false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	pdfData, err := io.ReadAll(stream)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF data: %w", err)
	}

	return pdfData, nil
}

func (g *HTMLGenerator) prepareDataWithImages(data map[string]string) CertificateData {
	certData := CertificateData{
		Name:           getOrDefault(data, "name", ""),
		StudentID:      getOrDefault(data, "student_id", ""),
		Course:         getOrDefault(data, "course", ""),
		Event:          getOrDefault(data, "event", ""),
		Club:           getOrDefault(data, "club", ""),
		Date:           getOrDefault(data, "date", ""),
		Signer1Title:   getOrDefault(data, "signer1_title", "Event Coordinator"),
		Signer2Title:   getOrDefault(data, "signer2_title", "Head Of Department\n(CSE)"),
		Signer3Title:   getOrDefault(data, "signer3_title", "Head Of Department\n(SOC)"),
		Signer4Title:   getOrDefault(data, "signer4_title", "Director,\nHaldwani Campus"),
	}

	sideDesign := getOrDefault(data, "side_design", "side.svg")
	orgLogo := getOrDefault(data, "org_logo", "gehu-haldwani-logo.svg")
	clubLogo := getOrDefault(data, "club_logo", "club.svg")
	sig1 := getOrDefault(data, "signature1", "cc.png")
	sig2 := getOrDefault(data, "signature2", "hod_cse.png")
	sig3 := getOrDefault(data, "signature3", "hod_soc.png")
	sig4 := getOrDefault(data, "signature4", "hld_dir.png")

	certData.SideDesignImage = g.getImageDataURI(sideDesign)
	certData.OrgLogo = g.getImageDataURI(orgLogo)
	certData.ClubLogo = g.getImageDataURI(clubLogo)
	certData.Signature1Image = g.getImageDataURI(sig1)
	certData.Signature2Image = g.getImageDataURI(sig2)
	certData.Signature3Image = g.getImageDataURI(sig3)
	certData.Signature4Image = g.getImageDataURI(sig4)

	return certData
}

func (g *HTMLGenerator) getImageDataURI(filename string) string {
	possiblePaths := []string{
		filepath.Join(g.templatesDir, "images", filename),
		filepath.Join(g.templatesDir, filename),
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			dataURI, err := g.fileToDataURI(path)
			if err == nil && dataURI != "" {
				return dataURI
			}
		}
	}

	return ""
}

func (g *HTMLGenerator) fileToDataURI(filePath string) (string, error) {
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(g.templatesDir, filePath)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	mimeType := "image/png"
	if strings.HasSuffix(filePath, ".svg") {
		mimeType = "image/svg+xml"
	} else if strings.HasSuffix(filePath, ".jpg") || strings.HasSuffix(filePath, ".jpeg") {
		mimeType = "image/jpeg"
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:%s;base64,%s", mimeType, encoded), nil
}

func getOrDefault(data map[string]string, key, defaultValue string) string {
	if val, ok := data[key]; ok && val != "" {
		return val
	}
	return defaultValue
}
