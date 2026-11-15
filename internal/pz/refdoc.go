package pz

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// Оформление по требованиям кафедры ВМСС с допустимым кеглем 12 пт.

var cm = 567.0 // twips в сантиметре (переменная: константа 1,25·cm не целая)

func stylesXML(ds DocStyle) string {
	rpr := func(sz int, extra string) string {
		return fmt.Sprintf(`<w:rPr><w:rFonts w:ascii="Times New Roman" w:hAnsi="Times New Roman" w:cs="Times New Roman" w:eastAsia="Times New Roman"/>%s<w:sz w:val="%d"/><w:szCs w:val="%d"/></w:rPr>`, extra, sz, sz)
	}
	para := func(id, name, based, ppr, rp string, extra string) string {
		b := ""
		if based != "" {
			b = `<w:basedOn w:val="` + based + `"/>`
		}
		return fmt.Sprintf(`<w:style w:type="paragraph" w:customStyle="1" w:styleId="%s"><w:name w:val="%s"/>%s%s<w:qFormat/><w:pPr>%s</w:pPr>%s</w:style>`, id, name, b, extra, ppr, rp)
	}
	body := fmt.Sprintf(`<w:spacing w:before="0" w:after="0" w:line="360" w:lineRule="auto"/><w:ind w:firstLine="%d"/><w:jc w:val="both"/>`, int(1.25*cm))
	center := `<w:spacing w:before="0" w:after="0" w:line="240" w:lineRule="auto"/><w:ind w:firstLine="0"/><w:jc w:val="center"/>`
	s := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
<w:docDefaults><w:rPrDefault>` + rpr(24, `<w:lang w:val="ru-RU" w:eastAsia="ru-RU" w:bidi="ar-SA"/>`) + `</w:rPrDefault>
<w:pPrDefault><w:pPr><w:spacing w:after="0" w:line="240" w:lineRule="auto"/></w:pPr></w:pPrDefault></w:docDefaults>
<w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/><w:qFormat/><w:pPr>` + body + `</w:pPr>` + rpr(24, "") + `</w:style>
` + para("BodyText", "Body Text", "Normal", body, "", "") +
		para("FirstParagraph", "First Paragraph", "BodyText", "", "", `<w:next w:val="BodyText"/>`) +
		para("Compact", "Compact", "Normal", `<w:ind w:firstLine="0"/><w:jc w:val="left"/>`, "", "") +
		para("Title", "Title", "Normal", center, rpr(28, "<w:b/>"), "") +
		para("Subtitle", "Subtitle", "Normal", center, "", "") +
		para("TitlePage", "TitlePage", "Normal", center, "", "") +
		para("TitleRight", "TitleRight", "Normal", fmt.Sprintf(`<w:ind w:left="%d" w:firstLine="0"/><w:jc w:val="left"/>`, int(9*cm)), "", "") +
		para("Center", "Center", "Normal", center, "", "") +
		para("SectionHeading", "SectionHeading", "Normal", `<w:keepNext/><w:spacing w:after="240"/><w:ind w:firstLine="0"/><w:jc w:val="center"/><w:outlineLvl w:val="0"/>`, rpr(24, "<w:b/>"), "") +
		para("Formula", "Formula", "Normal", `<w:keepLines/><w:tabs><w:tab w:val="center" w:pos="4961"/><w:tab w:val="right" w:pos="9921"/></w:tabs><w:spacing w:before="240" w:after="240"/><w:ind w:firstLine="0"/><w:jc w:val="left"/>`, "", "") +
		para("ListingTitle", "ListingTitle", "Normal", `<w:keepNext/><w:spacing w:before="240" w:after="60"/><w:ind w:firstLine="0"/><w:jc w:val="left"/>`, rpr(24, "<w:b/>"), "") +
		para("Fill", "Fill", "Normal", `<w:shd w:val="clear" w:color="auto" w:fill="FFF2A8"/><w:pBdr><w:left w:val="single" w:sz="18" w:space="4" w:color="E0A800"/></w:pBdr>`, rpr(24, "<w:i/>"), "") +
		para("SourceCode", "Source Code", "Normal", `<w:ind w:firstLine="0"/><w:jc w:val="left"/><w:wordWrap w:val="on"/>`, fmt.Sprintf(`<w:rPr><w:rFonts w:ascii="Courier New" w:hAnsi="Courier New" w:cs="Courier New"/><w:sz w:val="%d"/><w:szCs w:val="%[1]d"/></w:rPr>`, ds.codeSz()), "") +
		para("BlockText", "Block Text", "Normal", body, "", "") +
		para("Caption", "Caption", "Normal", center+`<w:spacing w:before="60" w:after="240"/>`, "", "") +
		para("ImageCaption", "Image Caption", "Caption", center+`<w:spacing w:before="60" w:after="240"/>`, "", "") +
		para("TableCaption", "Table Caption", "Caption", center+`<w:spacing w:before="0" w:after="60"/><w:keepNext/>`, "", "") +
		para("TableNum", "TableNum", "Normal", `<w:ind w:firstLine="0"/><w:jc w:val="right"/><w:spacing w:before="240" w:after="0"/><w:keepNext/>`, "", "") +
		para("Figure", "Figure", "Normal", center+`<w:keepNext/><w:spacing w:before="240"/>`, "", "") +
		para("CaptionedFigure", "Captioned Figure", "Figure", center+`<w:keepNext/><w:spacing w:before="240"/>`, "", "") +
		para("TOCHeading", "TOC Heading", "Normal", center+`<w:pageBreakBefore/>`, rpr(24, "<w:b/><w:caps/>"), "")
	h1jc, h1ind, caps := `<w:jc w:val="left"/>`, int(1.25*cm), ""
	if ds.H1Caps {
		caps = "<w:caps/>"
	}
	for i, h := range []struct {
		size int
		ppr  string
		caps string
		jc   string
		ind  int
	}{{24, `<w:pageBreakBefore/><w:spacing w:before="0" w:after="240"/>`, caps, h1jc, h1ind},
		{24, `<w:spacing w:before="240" w:after="120"/>`, "", `<w:jc w:val="left"/>`, int(1.25 * cm)},
		{24, `<w:spacing w:before="120" w:after="60"/>`, "", `<w:jc w:val="left"/>`, int(1.25 * cm)}} {
		s += fmt.Sprintf(`<w:style w:type="paragraph" w:styleId="Heading%d"><w:name w:val="heading %d"/><w:basedOn w:val="Normal"/><w:next w:val="BodyText"/><w:qFormat/><w:pPr><w:keepNext/><w:keepLines/>%s<w:ind w:firstLine="%d"/>%s<w:outlineLvl w:val="%d"/></w:pPr>%s</w:style>`,
			i+1, i+1, h.ppr, h.ind, h.jc, i, rpr(h.size, "<w:b/>"+h.caps))
	}
	// оглавление: уровни 1–2
	for i := 1; i <= 3; i++ {
		s += fmt.Sprintf(`<w:style w:type="paragraph" w:styleId="TOC%d"><w:name w:val="toc %d"/><w:basedOn w:val="Normal"/><w:pPr><w:tabs><w:tab w:val="right" w:leader="dot" w:pos="9627"/></w:tabs><w:ind w:left="%d" w:firstLine="0"/><w:jc w:val="left"/></w:pPr></w:style>`, i, i, (i-1)*240)
	}
	s += `<w:style w:type="character" w:default="1" w:styleId="DefaultParagraphFont"><w:name w:val="Default Paragraph Font"/><w:uiPriority w:val="1"/><w:semiHidden/></w:style>
<w:style w:type="character" w:customStyle="1" w:styleId="BodyTextChar"><w:name w:val="Body Text Char"/><w:basedOn w:val="DefaultParagraphFont"/></w:style>
<w:style w:type="character" w:customStyle="1" w:styleId="VerbatimChar"><w:name w:val="Verbatim Char"/><w:basedOn w:val="DefaultParagraphFont"/><w:rPr><w:rFonts w:ascii="Courier New" w:hAnsi="Courier New"/></w:rPr></w:style>
<w:style w:type="character" w:styleId="Hyperlink"><w:name w:val="Hyperlink"/><w:basedOn w:val="DefaultParagraphFont"/></w:style>
<w:style w:type="character" w:styleId="FootnoteReference"><w:name w:val="footnote reference"/><w:basedOn w:val="DefaultParagraphFont"/><w:rPr><w:vertAlign w:val="superscript"/></w:rPr></w:style>
<w:style w:type="paragraph" w:styleId="FootnoteText"><w:name w:val="footnote text"/><w:basedOn w:val="Normal"/><w:pPr><w:ind w:firstLine="0"/></w:pPr><w:rPr><w:sz w:val="20"/></w:rPr></w:style>
<w:style w:type="table" w:default="1" w:styleId="TableNormal"><w:name w:val="Normal Table"/><w:uiPriority w:val="99"/><w:semiHidden/><w:tblPr><w:tblInd w:w="0" w:type="dxa"/><w:tblCellMar><w:top w:w="0" w:type="dxa"/><w:left w:w="108" w:type="dxa"/><w:bottom w:w="0" w:type="dxa"/><w:right w:w="108" w:type="dxa"/></w:tblCellMar></w:tblPr></w:style>
<w:style w:type="table" w:styleId="Table"><w:name w:val="Table"/><w:basedOn w:val="TableNormal"/><w:qFormat/><w:pPr><w:ind w:firstLine="0"/><w:jc w:val="left"/></w:pPr><w:tblPr><w:jc w:val="center"/><w:tblBorders><w:top w:val="single" w:sz="4" w:space="0" w:color="000000"/><w:left w:val="single" w:sz="4" w:space="0" w:color="000000"/><w:bottom w:val="single" w:sz="4" w:space="0" w:color="000000"/><w:right w:val="single" w:sz="4" w:space="0" w:color="000000"/><w:insideH w:val="single" w:sz="4" w:space="0" w:color="000000"/><w:insideV w:val="single" w:sz="4" w:space="0" w:color="000000"/></w:tblBorders><w:tblCellMar><w:left w:w="85" w:type="dxa"/><w:right w:w="85" w:type="dxa"/></w:tblCellMar></w:tblPr><w:tblStylePr w:type="firstRow"><w:pPr><w:jc w:val="center"/></w:pPr><w:rPr><w:b/></w:rPr><w:tcPr><w:vAlign w:val="center"/></w:tcPr></w:tblStylePr></w:style>
</w:styles>`
	return s
}

func documentXML() string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><w:body><w:p/>
<w:sectPr><w:headerReference w:type="default" r:id="rIdHdr1"/><w:headerReference w:type="first" r:id="rIdHdr2"/><w:footerReference w:type="default" r:id="rIdFtr1"/><w:footerReference w:type="first" r:id="rIdFtr2"/><w:pgSz w:w="11906" w:h="16838"/><w:pgMar w:top="%d" w:right="%d" w:bottom="%d" w:left="%d" w:header="567" w:footer="567" w:gutter="0"/><w:titlePg/></w:sectPr></w:body></w:document>`,
		int(2.5*cm), int(1*cm), int(2.5*cm), int(2.5*cm))
}

const hdrNS = `xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"`

func pageField(jc string) string {
	return `<w:p><w:pPr><w:jc w:val="` + jc + `"/><w:ind w:firstLine="0"/></w:pPr><w:r><w:fldChar w:fldCharType="begin"/></w:r><w:r><w:instrText xml:space="preserve"> PAGE </w:instrText></w:r><w:r><w:fldChar w:fldCharType="separate"/></w:r><w:r><w:t>2</w:t></w:r><w:r><w:fldChar w:fldCharType="end"/></w:r></w:p>`
}

// колонтитулы: номер страницы (на титуле — нет), текст справа — по стилю
func headerFooter(ds DocStyle) (hdr, ftr string) {
	empty := `<w:p><w:pPr><w:ind w:firstLine="0"/></w:pPr></w:p>`
	hdr, ftr = empty, empty
	ftr = pageField("center")
	if ds.HeaderText != "" {
		hdr = `<w:p><w:pPr><w:jc w:val="right"/><w:ind w:firstLine="0"/></w:pPr><w:r><w:t>` + ds.HeaderText + `</w:t></w:r></w:p>`
	}
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:hdr ` + hdrNS + `>` + hdr + `</w:hdr>`,
		`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:ftr ` + hdrNS + `>` + ftr + `</w:ftr>`
}

// ReferenceDocx — эталон стилей для pandoc: берём стандартный и заменяем стили и параметры страницы.
func ReferenceDocx(pandoc string, ds DocStyle) ([]byte, error) {
	def, err := exec.Command(pandoc, "--print-default-data-file", "reference.docx").Output()
	if err != nil {
		return nil, fmt.Errorf("pandoc: %w", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(def), int64(len(def)))
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	for _, f := range zr.File {
		w, err := zw.Create(f.Name)
		if err != nil {
			return nil, err
		}
		switch f.Name {
		case "word/styles.xml":
			io.WriteString(w, stylesXML(ds))
			continue
		case "[Content_Types].xml", "word/_rels/document.xml.rels":
			rc, _ := f.Open()
			b, _ := io.ReadAll(rc)
			rc.Close()
			x := string(b)
			if f.Name == "[Content_Types].xml" {
				var add string
				for _, n := range []string{"header1", "header2"} {
					add += `<Override PartName="/word/` + n + `.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.header+xml"/>`
				}
				for _, n := range []string{"footer1", "footer2"} {
					add += `<Override PartName="/word/` + n + `.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.footer+xml"/>`
				}
				x = strings.Replace(x, "</Types>", add+"</Types>", 1)
			} else {
				rel := func(id, typ, target string) string {
					return `<Relationship Id="` + id + `" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/` + typ + `" Target="` + target + `"/>`
				}
				x = strings.Replace(x, "</Relationships>", rel("rIdHdr1", "header", "header1.xml")+rel("rIdHdr2", "header", "header2.xml")+
					rel("rIdFtr1", "footer", "footer1.xml")+rel("rIdFtr2", "footer", "footer2.xml")+"</Relationships>", 1)
			}
			io.WriteString(w, x)
			continue
		case "word/document.xml":
			io.WriteString(w, documentXML())
			continue
		case "word/settings.xml":
			// обновить поля (оглавление) при открытии
			rc, _ := f.Open()
			b, _ := io.ReadAll(rc)
			rc.Close()
			io.WriteString(w, strings.Replace(string(b), "</w:settings>", `<w:updateFields w:val="true"/></w:settings>`, 1))
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		io.Copy(w, rc)
		rc.Close()
	}
	hdr, ftr := headerFooter(ds)
	emptyHdr := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:hdr ` + hdrNS + `><w:p/></w:hdr>`
	emptyFtr := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:ftr ` + hdrNS + `><w:p/></w:ftr>`
	for name, x := range map[string]string{"word/header1.xml": hdr, "word/header2.xml": emptyHdr, "word/footer1.xml": ftr, "word/footer2.xml": emptyFtr} {
		w, err := zw.Create(name)
		if err != nil {
			return nil, err
		}
		io.WriteString(w, x)
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
