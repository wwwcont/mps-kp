package pz

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// Проверка порядка элементов OOXML. Word требует порядок дочерних элементов из схемы (ECMA-376, wml.xsd) — при нарушении он
// «восстанавливает» документ (выкидывает куски) или раскладывает криво; LibreOffice такие ошибки прощает, поэтому превью
// их не показывает. Здесь — последовательности для элементов, которые мы пишем руками (стили, титул, таблицы, колонтитулы).
var ooxmlOrder = map[string][]string{
	"pPr": {"pStyle", "keepNext", "keepLines", "pageBreakBefore", "framePr", "widowControl", "numPr", "suppressLineNumbers", "pBdr", "shd",
		"tabs", "suppressAutoHyphens", "kinsoku", "wordWrap", "overflowPunct", "topLinePunct", "autoSpaceDE", "autoSpaceDN", "bidi",
		"adjustRightInd", "snapToGrid", "spacing", "ind", "contextualSpacing", "mirrorIndents", "suppressOverlap", "jc", "textDirection",
		"textAlignment", "textboxTightWrap", "outlineLvl", "divId", "cnfStyle", "rPr", "sectPr", "pPrChange"},
	"rPr": {"ins", "del", "moveFrom", "moveTo", "rStyle", "rFonts", "b", "bCs", "i", "iCs", "caps", "smallCaps", "strike", "dstrike", "outline", "shadow",
		"emboss", "imprint", "noProof", "snapToGrid", "vanish", "webHidden", "color", "spacing", "w", "kern", "position", "sz", "szCs",
		"highlight", "u", "effect", "bdr", "shd", "fitText", "vertAlign", "rtl", "cs", "em", "lang", "eastAsianLayout", "specVanish", "oMath"},
	"tblPr": {"tblStyle", "tblpPr", "tblOverlap", "bidiVisual", "tblStyleRowBandSize", "tblStyleColBandSize", "tblW", "jc",
		"tblCellSpacing", "tblInd", "tblBorders", "shd", "tblLayout", "tblCellMar", "tblLook", "tblCaption", "tblDescription"},
	"tcPr": {"cnfStyle", "tcW", "gridSpan", "hMerge", "vMerge", "tcBorders", "shd", "noWrap", "tcMar", "textDirection", "tcFitText",
		"vAlign", "hideMark"},
	"tblBorders": {"top", "left", "start", "bottom", "right", "end", "insideH", "insideV"},
	"tcBorders":  {"top", "left", "start", "bottom", "right", "end", "insideH", "insideV", "tl2br", "tr2bl"},
	"pBdr":       {"top", "left", "bottom", "right", "between", "bar"},
	"tblCellMar": {"top", "left", "start", "bottom", "right", "end"},
	"tcMar":      {"top", "left", "start", "bottom", "right", "end"},
	"style": {"name", "aliases", "basedOn", "next", "link", "autoRedefine", "hidden", "uiPriority", "semiHidden", "unhideWhenUsed",
		"qFormat", "locked", "personal", "personalCompose", "personalReply", "rsid", "pPr", "rPr", "tblPr", "trPr", "tcPr", "tblStylePr"},
	"sectPr": {"headerReference", "footnotePr", "endnotePr", "type", "pgSz", "pgMar", "paperSrc", "pgBorders", "lnNumType",
		"pgNumType", "cols", "formProt", "vAlign", "noEndnote", "titlePg", "textDirection", "bidi", "rtlGutter", "docGrid", "printerSettings"},
	"tbl": {"bookmarkStart", "tblPr", "tblGrid", "tr"},
	"tr":  {"tblPrEx", "trPr", "tc"},
	"tc":  {"tcPr", "p", "tbl"},
}

// checkOrder — нарушения порядка в одной XML-части.
func checkOrder(part string, r io.Reader) []string {
	const w = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"
	type frame struct {
		name string
		last int
		prev string
	}
	var stack []*frame
	var errs []string
	d := xml.NewDecoder(r)
	for {
		tok, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return append(errs, fmt.Sprintf("%s: XML не разбирается: %v", part, err))
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if len(stack) > 0 && t.Name.Space == w {
				top := stack[len(stack)-1]
				if seq, ok := ooxmlOrder[top.name]; ok {
					name := t.Name.Local
					if name == "footerReference" {
						name = "headerReference" // ссылки на колонтитулы идут одной группой
					}
					idx := -1
					for i, s := range seq {
						if s == name {
							idx = i
							break
						}
					}
					if idx >= 0 {
						if idx < top.last {
							errs = append(errs, fmt.Sprintf("%s: в <%s> элемент <%s> стоит после <%s>", part, top.name, t.Name.Local, top.prev))
						} else {
							top.last, top.prev = idx, t.Name.Local
						}
					}
				}
			}
			name := t.Name.Local
			if t.Name.Space != w {
				name = "~" + name
			}
			stack = append(stack, &frame{name: name, last: -1})
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}
	return errs
}

// CheckOOXML — нарушения порядка элементов в docx (документ, стили, колонтитулы). Пусто — Word откроет без «восстановления».
func CheckOOXML(docx string) ([]string, error) {
	z, err := zip.OpenReader(docx)
	if err != nil {
		return nil, err
	}
	defer z.Close()
	var errs []string
	seen := map[string]bool{}
	for _, f := range z.File {
		if !strings.HasPrefix(f.Name, "word/") || !strings.HasSuffix(f.Name, ".xml") || strings.Contains(f.Name, "_rels") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		for _, e := range checkOrder(f.Name, rc) {
			if !seen[e] {
				seen[e] = true
				errs = append(errs, e)
			}
		}
		rc.Close()
	}
	return errs, nil
}

// ---------------------------------------------------------------- нормализация порядка

// children делит содержимое контейнера на дочерние элементы верхнего уровня (вместе с их содержимым).
// Текст между элементами (пробелы) отбрасывается. ok = false — разобрать не удалось (содержимое оставляем как есть).
func children(s string) (out []string, ok bool) {
	i := 0
	for i < len(s) {
		for i < len(s) && (s[i] == ' ' || s[i] == '\n' || s[i] == '\r' || s[i] == '\t') {
			i++
		}
		if i >= len(s) {
			break
		}
		if s[i] != '<' {
			return nil, false
		}
		// имя тега
		j := i + 1
		for j < len(s) && s[j] != ' ' && s[j] != '>' && s[j] != '/' {
			j++
		}
		name := s[i+1 : j]
		end := strings.IndexByte(s[i:], '>')
		if end < 0 {
			return nil, false
		}
		end += i
		if s[end-1] == '/' { // <w:x …/>
			out = append(out, s[i:end+1])
			i = end + 1
			continue
		}
		// <w:x …> … </w:x> с учётом вложенных одноимённых
		depth, k := 1, end+1
		for depth > 0 {
			o := strings.Index(s[k:], "<"+name)
			c := strings.Index(s[k:], "</"+name+">")
			if c < 0 {
				return nil, false
			}
			if o >= 0 && o < c {
				// вложенный одноимённый открывающий (не самозакрывающийся)
				e := strings.IndexByte(s[k+o:], '>')
				if e > 0 && s[k+o+e-1] != '/' {
					nb := s[k+o+len(name)+1]
					if nb == ' ' || nb == '>' {
						depth++
					}
				}
				k += o + 1
				continue
			}
			depth--
			k += c + len(name) + 3
		}
		out = append(out, s[i:k])
		i = k
	}
	return out, true
}

func localName(el string) string {
	j := 1
	for j < len(el) && el[j] != ' ' && el[j] != '>' && el[j] != '/' {
		j++
	}
	n := el[1:j]
	if k := strings.IndexByte(n, ':'); k >= 0 {
		n = n[k+1:]
	}
	return n
}

// orderContainer переставляет детей <w:tag>…</w:tag> по схеме (неизвестные — в конец, в исходном порядке).
func orderContainer(xmlStr, tag string) string {
	seq := ooxmlOrder[tag]
	rank := map[string]int{}
	for i, s := range seq {
		rank[s] = i
	}
	if tag == "sectPr" {
		rank["footerReference"] = rank["headerReference"]
	}
	open, close := "<w:"+tag+">", "</w:"+tag+">"
	var b strings.Builder
	rest := xmlStr
	for {
		i := strings.Index(rest, open)
		if i < 0 {
			b.WriteString(rest)
			break
		}
		j := strings.Index(rest[i:], close)
		if j < 0 {
			b.WriteString(rest)
			break
		}
		inner := rest[i+len(open) : i+j]
		b.WriteString(rest[:i+len(open)])
		if strings.Contains(inner, open) { // вложенный такой же контейнер — не трогаем
			b.WriteString(inner)
		} else if ch, ok := children(inner); ok {
			idx := make([]int, len(ch))
			for k := range ch {
				idx[k] = k
			}
			r := func(k int) int {
				if v, ok := rank[localName(ch[k])]; ok {
					return v
				}
				return 1 << 20
			}
			sortStable(idx, func(a, c int) bool { return r(a) < r(c) })
			for _, k := range idx {
				b.WriteString(ch[k])
			}
		} else {
			b.WriteString(inner)
		}
		b.WriteString(close)
		rest = rest[i+j+len(close):]
	}
	return b.String()
}

func sortStable(idx []int, less func(a, b int) bool) {
	for i := 1; i < len(idx); i++ {
		for j := i; j > 0 && less(idx[j], idx[j-1]); j-- {
			idx[j], idx[j-1] = idx[j-1], idx[j]
		}
	}
}

// NormalizeOOXML — порядок детей в rPr, pPr, tcPr, tblPr, рамках и стилях (сначала внутренние, потом внешние).
func NormalizeOOXML(x string) string {
	for _, tag := range []string{"rPr", "pBdr", "tcBorders", "tblBorders", "tblCellMar", "tcMar", "pPr", "tcPr", "tblPr", "sectPr"} {
		x = orderContainer(x, tag)
	}
	return x
}

// pageBreaks — «с новой страницы» у заголовков разделов (Heading1) и «СОДЕРЖАНИЯ» (TOCHeading): свойство стиля pageBreakBefore
// простые просмотрщики docx игнорируют (всё идёт сплошняком), поэтому — явный разрыв первым элементом самого заголовка
// (понимают все; в отличие от отдельного абзаца с разрывом, не даёт пустого листа в Word). В стилях pageBreakBefore убирается.
func pageBreaks(name, x string, hasAbstract bool) string {
	if name == "word/styles.xml" {
		return strings.NewReplacer("<w:pageBreakBefore/>", "", "<w:pageBreakBefore />", "").Replace(x)
	}
	if name != "word/document.xml" {
		return x
	}
	var front string
	if hasAbstract {
		i := strings.LastIndex(x, "<w:sectPr")
		toc := `<w:p><w:pPr><w:pStyle w:val="TOCHeading"/>`
		if i >= 0 && strings.Contains(x, toc) {
			if end := strings.Index(x[i:], "</w:sectPr>"); end >= 0 {
				section := x[i : i+end+len("</w:sectPr>")]
				refs := regexp.MustCompile(`<w:footerReference[^>]*?/>`)
				front = refs.ReplaceAllStringFunc(section, func(ref string) string {
					if strings.Contains(ref, `w:type="first"`) {
						return strings.Replace(ref, `w:type="first"`, `w:type="default"`, 1)
					}
					return ""
				})
				x = x[:i] + regexp.MustCompile(`<w:titlePg[^>]*?/>`).ReplaceAllString(x[i:], "")
				x = strings.Replace(x, toc, `<w:p><w:pPr><w:spacing w:line="20" w:lineRule="exact"/>`+front+`</w:pPr></w:p>`+toc, 1)
			}
		}
	}
	var b strings.Builder
	for {
		i := -1
		for _, st := range []string{`<w:pStyle w:val="Heading1"`, `<w:pStyle w:val="TOCHeading"`, `<w:pStyle w:val="SectionHeading"`} {
			if j := strings.Index(x, st); j >= 0 && (i < 0 || j < i) {
				i = j
			}
		}
		if i < 0 {
			b.WriteString(x)
			return b.String()
		}
		k := strings.Index(x[i:], "</w:pPr>")
		if k < 0 {
			b.WriteString(x)
			return b.String()
		}
		k += i + len("</w:pPr>")
		b.WriteString(x[:k])
		if front == "" || !strings.HasPrefix(x[i:], `<w:pStyle w:val="TOCHeading"`) {
			b.WriteString(`<w:r><w:br w:type="page"/></w:r>`)
		}
		x = x[k:]
	}
}

// normalizeDocx переписывает word/*.xml в docx с упорядоченными элементами.
func normalizeDocx(path string, media map[string][]byte, hasAbstract bool) error {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			zr.Close()
			return err
		}
		b, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			zr.Close()
			return err
		}
		if strings.HasPrefix(f.Name, "word/") && strings.HasSuffix(f.Name, ".xml") && !strings.Contains(f.Name, "_rels") {
			b = []byte(NormalizeOOXML(pageBreaks(f.Name, string(b), hasAbstract)))
		}
		// картинки сырого OOXML: ссылки rId<Имя> → media/<файл>; тип png объявлен
		if f.Name == "word/_rels/document.xml.rels" {
			x := string(b)
			for name := range media {
				id := mediaID(name)
				if !strings.Contains(x, `Id="`+id+`"`) {
					x = strings.Replace(x, "</Relationships>", `<Relationship Id="`+id+`" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/`+name+`"/></Relationships>`, 1)
				}
			}
			b = []byte(x)
		}
		if f.Name == "[Content_Types].xml" && len(media) > 0 && !strings.Contains(string(b), `Extension="png"`) {
			b = []byte(strings.Replace(string(b), "<Default ", `<Default Extension="png" ContentType="image/png"/><Default `, 1))
		}
		w, err := zw.CreateHeader(&zip.FileHeader{Name: f.Name, Method: f.Method, Modified: f.Modified})
		if err != nil {
			zr.Close()
			return err
		}
		w.Write(b)
	}
	zr.Close()
	for name, data := range media {
		w, err := zw.Create("word/media/" + name)
		if err != nil {
			return err
		}
		w.Write(data)
	}
	if err := zw.Close(); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

// mediaID — rId картинки сырого OOXML по имени файла (sidegraphs.png → rIdSideGraph).
func mediaID(name string) string {
	if name == "sidegraphs.png" {
		return "rIdSideGraph"
	}
	return "rIdMedia_" + strings.TrimSuffix(name, ".png")
}
