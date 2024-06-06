package templating

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Template struct {
	Folder     string
	FileName   string
	Name       string
	Date       string
	Title      string
	FontSize   int
	Paragraphs []Paragraph
}

type Paragraph struct {
	Action string
	Branch string
	Job    string
	Hash   string
}

// XML content for the DOCX structure
const contentTypesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
	<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
	<Default Extension="xml" ContentType="application/xml"/>
	<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
	<Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>
</Types>`

const relsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
	<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`

const documentRelsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
</Relationships>`

const stylesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
</w:styles>`

func (t Template) GenerateDocx() error {

	// Define the structure of the DOCX file
	files := map[string]string{
		"[Content_Types].xml":          contentTypesXML,
		"_rels/.rels":                  relsXML,
		"word/_rels/document.xml.rels": documentRelsXML,
		"word/document.xml":            t.generateDocumentXML(),
		"word/styles.xml":              stylesXML,
	}

	err := os.MkdirAll(t.Folder, os.ModePerm)

	if err != nil {
		return err
	}

	// Create the docx file
	docxFileName := filepath.Join(t.Folder, t.FileName)
	zipFile, err := os.Create(docxFileName)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return err
	}
	defer zipFile.Close()

	// Create a new zip writer
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Add files to the zip
	for name, content := range files {
		f, err := zipWriter.Create(name)
		if err != nil {
			fmt.Println("Error creating zip entry:", err)
			return err
		}
		_, err = io.Copy(f, bytes.NewReader([]byte(content)))
		if err != nil {
			fmt.Println("Error writing content to zip entry:", err)
			return err
		}
	}

	fmt.Println("Reports has been created successfully!")

	return nil
}

// Function to generate document.xml content with dynamic content and formatting
func (t Template) generateDocumentXML() string {
	// Start of the document
	xmlContent := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
	<w:body>`

	// Title part
	xmlContent += fmt.Sprintf(`
		<w:p>
			<w:pPr>
				<w:jc w:val="center"/>
			</w:pPr>
			<w:r>
				<w:rPr>
					<w:b/>
					<w:sz w:val="%d"/>
				</w:rPr>
				<w:t>%s</w:t>
			</w:r>
		</w:p>`, t.FontSize*2, t.Title)

	// Sub title part
	xmlContent += fmt.Sprintf(`
		<w:p>
			<w:r>
				<w:rPr>
					<w:b/>
				</w:rPr>
				<w:t>Nama: %s</w:t>
			</w:r>
			<w:br w:type="textWrapping"/>
			<w:r>
				<w:rPr>
					<w:b/>
				</w:rPr>
				<w:t>Tanggal: %s</w:t>
				<w:br/>
			</w:r>
		</w:p>`, t.Name, t.Date)

	// Contens part
	for _, paragraph := range t.Paragraphs {
		xmlContent += fmt.Sprintf(`
		<w:p>
			<w:r>
				<w:t>Action: %s</w:t>
				<w:br w:type="textWrapping"/>
			</w:r>
			<w:r>
				<w:t>Branch: %s</w:t>
				<w:br w:type="textWrapping"/>
			</w:r>
			<w:r>
				<w:t>Title: %s</w:t>
				<w:br w:type="textWrapping"/>
			</w:r>
			<w:r>
				<w:t>Commit Hash: %s</w:t>
			</w:r>
		</w:p>`, paragraph.Action, paragraph.Branch, paragraph.Job, paragraph.Hash)
	}

	// End of the document
	xmlContent += `
	</w:body>
</w:document>`

	return xmlContent
}
