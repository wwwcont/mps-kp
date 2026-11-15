// Package pz — пояснительные записки (ПЗ1 — КМ-2, ПЗ2 — КМ-3): генерируемая часть (таблицы, расчёты, рисунки,
// листинги) + текст студента из students/<ник>/pz/pzN.md по местам «ДОПИШИ». Сборка — pandoc → docx.
package pz

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Fill — место, которое пишет студент.
type Fill struct {
	ID, Title, Hint string
	Done            bool
}

// Doc — markdown для pandoc с нумерацией разделов, рисунков, таблиц.
type Doc struct {
	b          strings.Builder
	h1, h2, h3 int
	fig, tab   int
	formula    int
	Dir        string            // папка сборки (рисунки кладутся сюда)
	Student    map[string]string // текст студента по ид
	Fills      []Fill
	appendixNo int
	Style      DocStyle
	toc        []tocEntry
	tocPages   []int             // номера страниц заголовков (FillTOCPages), nil — без номеров
	pages      int               // число листов по рендеру (FillTOCPages), 0 — неизвестно
	FontDir    string            // папка шрифтов (fonts/): для картинок титула; пусто — без них
	media      map[string][]byte // картинки сырого OOXML (word/media/…), кладутся в docx после pandoc
}

type tocEntry struct {
	level int
	text  string
}

// NewDoc — документ; student — разобранный students/<ник>/pz/pzN.md (может быть nil).
func NewDoc(dir string, student map[string]string) *Doc {
	return &Doc{Dir: dir, Student: student, Style: docStyles["C"], media: map[string][]byte{}}
}

func esc(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `*`, `\*`, `_`, `\_`, "`", "\\`", `<`, `\<`, `>`, `\>`, `[`, `\[`, `]`, `\]`, `#`, `\#`, `|`, `\|`, `$`, `\$`, `~`, `\~`, `^`, `\^`)
	return r.Replace(s)
}

// Esc — экранирование текста для markdown (номера, формулы с «*» и «_»).
func Esc(s string) string { return esc(s) }

func (d *Doc) w(format string, a ...any) { fmt.Fprintf(&d.b, format, a...) }

// Styled — абзацы в пользовательском стиле (TitlePage, Center, Fill…).
func (d *Doc) Styled(style string, lines ...string) {
	d.w("\n::: {custom-style=\"%s\"}\n", style)
	for _, l := range lines {
		if l == "" {
			l = " "
		}
		d.w("%s\n\n", l)
	}
	d.w(":::\n\n")
}

// PageBreak — разрыв страницы.
func (d *Doc) PageBreak() {
	d.w("\n```{=openxml}\n<w:p><w:r><w:br w:type=\"page\"/></w:r></w:p>\n```\n\n")
}

// H1 — раздел «N Название» с новой страницы; num = false — без номера (Аннотация, Приложения).
func (d *Doc) H1(title string, num bool) {
	if num {
		d.h1++
		d.h2 = 0
		d.w("\n# %d. %s {-}\n\n", d.h1, esc(title))
		d.toc = append(d.toc, tocEntry{1, fmt.Sprintf("%d. %s", d.h1, title)})
		return
	}
	d.Styled("SectionHeading", esc(title))
	if title != "АННОТАЦИЯ" {
		d.toc = append(d.toc, tocEntry{1, title})
	}
}

// H2 — подраздел «N.M Название».
func (d *Doc) H2(title string) {
	d.h2++
	d.h3 = 0
	d.w("\n## %d.%d. %s {-}\n\n", d.h1, d.h2, esc(title))
	d.toc = append(d.toc, tocEntry{2, fmt.Sprintf("%d.%d. %s", d.h1, d.h2, title)})
}

// H3 — пункт «N.M.K Название».
func (d *Doc) H3(title string) {
	d.h3++
	d.w("\n### %d.%d.%d. %s {-}\n\n", d.h1, d.h2, d.h3, esc(title))
}

// P — абзац (markdown разрешён: **жирный**, *курсив*).
func (d *Doc) P(format string, a ...any) { d.w("%s\n\n", fmt.Sprintf(format, a...)) }

// Bullets — маркированный список.
func (d *Doc) Bullets(items ...string) {
	for _, it := range items {
		d.w("- %s\n", it)
	}
	d.w("\n")
}

// Numbered — нумерованный список «1. …» (литература).
func (d *Doc) Numbered(items ...string) {
	for i, it := range items {
		d.w("%d. %s\n", i+1, esc(it))
	}
	d.w("\n")
}

// NextFig / NextTab — номер, который получит следующий рисунок / таблица (для ссылок в тексте до них).
func (d *Doc) NextFig() string { return d.label(d.fig + 1) }
func (d *Doc) NextTab() string { return d.label(d.tab + 1) }

func (d *Doc) label(n int) string {
	if d.appendixNo > 0 {
		return fmt.Sprintf("%s.%d", appLetters[d.appendixNo-1], n)
	}
	return fmt.Sprint(n)
}

// Table — таблица с номером справа и названием по центру. Возвращает номер.
func (d *Doc) Table(caption string, head []string, rows [][]string) string {
	return d.TableW(caption, head, rows, nil)
}

// TableW — таблица с относительными ширинами колонок (например 20/110/10/45 для перечня по ГОСТ 2.701).
func (d *Doc) TableW(caption string, head []string, rows [][]string, widths []int) string {
	d.tab++
	n := d.label(d.tab)
	d.Styled("TableNum", "Таблица "+n)
	d.Styled("TableCaption", esc(caption))
	d.w("\n")
	line := func(cells []string) {
		d.w("|")
		for _, c := range cells {
			d.w(" %s |", strings.ReplaceAll(c, "|", `\|`))
		}
		d.w("\n")
	}
	line(head)
	d.w("|")
	for i := range head {
		n := 10
		if i < len(widths) {
			n = widths[i]
		}
		d.w("%s|", strings.Repeat("-", max(3, n)))
	}
	d.w("\n")
	for _, r := range rows {
		line(r)
	}
	d.w("\n")
	return n
}

// Figure — рисунок из файла (в Dir) по центру с подписью под ним («Рисунок N — …» или «Рис. N. …»). widthCm = 0 — 16 см.
func (d *Doc) Figure(caption, file string, widthCm float64) string {
	d.fig++
	n := d.label(d.fig)
	if widthCm == 0 {
		widthCm = 16
	}
	d.w("\n::: {custom-style=\"Figure\"}\n![](%s){width=%.1fcm}\n:::\n\n", filepath.ToSlash(file), widthCm)
	if d.Style.FigShort {
		d.Styled("ImageCaption", "Рис. "+n+". "+esc(caption))
	} else {
		d.Styled("ImageCaption", "Рисунок "+n+" — "+esc(caption))
	}
	return n
}

// Formula uses a center tab for the expression and a right tab for its number.
func (d *Doc) Formula(s string) {
	d.formula++
	d.w("\n```{=openxml}\n<w:p><w:pPr><w:pStyle w:val=\"Formula\"/></w:pPr><w:r><w:tab/><w:t>%s</w:t><w:tab/><w:t>(%d)</w:t></w:r></w:p>\n```\n\n", xmlEsc(s), d.formula)
}

// Code — листинг моноширинным шрифтом.
func (d *Doc) Code(text string) {
	fence := "```"
	for strings.Contains(text, fence) {
		fence += "`"
	}
	d.w("\n%s\n%s\n%s\n\n", fence, strings.TrimRight(text, "\n"), fence)
}

// Fill — место для текста студента: если в students/<ник>/pz/pzN.md раздел «## id» заполнен — вставляется он,
// иначе — подсвеченная подсказка «✍ ДОПИШИ: …».
var (
	reRawBlock = regexp.MustCompile("(?s)(?:```+|~~~+)[ \\t]*\\{=[^}]*\\}.*?\\n[ \\t]*(?:```+|~~~+)")
	reRawSpan  = regexp.MustCompile("`[^`]*`\\{=[^}]*\\}")
	reHeading  = regexp.MustCompile(`(?m)^\s{0,3}#{1,6}\s+(.+?)\s*#*\s*$`)
)

// studentText — текст студента для вставки в ПЗ: обычная разметка (жирный, курсив, списки) работает, но
// служебные вставки pandoc ({=openxml} и т.п.) вырезаются (могут сломать docx), свои заголовки «# …» — жирным абзацем
// (иначе сбивают нумерацию разделов), обратные слэши — буквально (C:\Keil не превращается в «C:»).
func studentText(s string) string {
	s = reRawBlock.ReplaceAllString(s, "")
	s = reRawSpan.ReplaceAllString(s, "")
	s = reHeading.ReplaceAllString(s, "**$1**")
	s = strings.ReplaceAll(s, `\`, `\\`)
	return strings.TrimSpace(s)
}

func (d *Doc) Fill(id, title, hint string) {
	txt := studentText(d.Student[id])
	f := Fill{ID: id, Title: title, Hint: hint, Done: txt != ""}
	d.Fills = append(d.Fills, f)
	if f.Done {
		d.w("\n%s\n\n", txt)
		return
	}
	d.Styled("Fill", "✍ ДОПИШИ ("+id+" в students/…/pz/): "+esc(hint))
}

// TOC — оглавление: поле Word TOC (Word пересчитает с номерами страниц при открытии), внутри — готовый список
// заголовков (его покажут LibreOffice и МойОфис, которые поля не обновляют). Список подставляется в Markdown().
func (d *Doc) TOC() { d.w("\n```{=openxml}\n%s\n```\n\n", tocMarker) }

const tocMarker = "<!--TOC-->"

func (d *Doc) tocXML() string {
	// таблица без рамок «название | страница»: в Word, LibreOffice и упрощённых просмотрщиках выглядит одинаково
	// (табуляция с точками до номера внутри ячейки названия — где не поддерживается, просто пробел). Номера — FillTOCPages.
	var b strings.Builder
	b.WriteString(`<w:p><w:pPr><w:pStyle w:val="TOCHeading"/></w:pPr><w:r><w:t>СОДЕРЖАНИЕ</w:t></w:r></w:p>`)
	numW := twip(12)
	txtW := twip(textW) - numW
	fmt.Fprintf(&b, `<w:tbl><w:tblPr><w:tblW w:w="%d" w:type="dxa"/><w:tblBorders><w:top w:val="nil"/><w:left w:val="nil"/><w:bottom w:val="nil"/><w:right w:val="nil"/><w:insideH w:val="nil"/><w:insideV w:val="nil"/></w:tblBorders><w:tblLayout w:type="fixed"/>`+
		`<w:tblCellMar><w:left w:w="0" w:type="dxa"/><w:right w:w="0" w:type="dxa"/></w:tblCellMar></w:tblPr><w:tblGrid><w:gridCol w:w="%d"/><w:gridCol w:w="%d"/></w:tblGrid>`, txtW+numW, txtW, numW)
	for i, e := range d.toc {
		num := ""
		if i < len(d.tocPages) {
			num = fmt.Sprint(d.tocPages[i])
		}
		fmt.Fprintf(&b, `<w:tr><w:trPr><w:cantSplit/></w:trPr><w:tc><w:tcPr><w:tcW w:w="%d" w:type="dxa"/><w:vAlign w:val="bottom"/></w:tcPr>`+
			`<w:p><w:pPr><w:pStyle w:val="TOC%d"/><w:tabs><w:tab w:val="right" w:leader="dot" w:pos="%d"/></w:tabs></w:pPr><w:r><w:t xml:space="preserve">%s</w:t></w:r><w:r><w:tab/></w:r></w:p></w:tc>`+
			`<w:tc><w:tcPr><w:tcW w:w="%d" w:type="dxa"/><w:vAlign w:val="bottom"/></w:tcPr><w:p><w:pPr><w:pStyle w:val="TOC%d"/><w:ind w:left="0" w:firstLine="0"/><w:jc w:val="right"/></w:pPr><w:r><w:t>%s</w:t></w:r></w:p></w:tc></w:tr>`,
			txtW, e.level, txtW-20, xmlEsc(e.text), numW, e.level, num)
	}
	b.WriteString(`</w:tbl>`)
	return b.String()
}

// Num — число с десятичной запятой (ГОСТ): Num(3.19, 3) → «3,190».
func Num(v float64, prec int) string {
	return strings.Replace(fmt.Sprintf("%.*f", prec, v), ".", ",", 1)
}

// FillOr — место с готовым текстом по умолчанию: если студент написал свой раздел — берётся он, иначе — generated
// (не подсвечивается и не считается незаполненным; в файле студента раздел остаётся, чтобы заменить при желании).
func (d *Doc) FillOr(id, title, hint, generated string) {
	txt := studentText(d.Student[id])
	d.Fills = append(d.Fills, Fill{ID: id, Title: title, Hint: hint + " (необязательно: без текста в документ идёт пересказ схемы)", Done: true})
	if txt == "" {
		txt = esc(generated)
	}
	d.w("\n%s\n\n", txt)
}

// Appendix — «ПРИЛОЖЕНИЕ А» с новой страницы.
func (d *Doc) Appendix(title string) string {
	letter := appLetters[d.appendixNo]
	d.appendixNo++
	d.fig, d.tab = 0, 0
	d.Styled("SectionHeading", "ПРИЛОЖЕНИЕ "+letter)
	d.toc = append(d.toc, tocEntry{1, "ПРИЛОЖЕНИЕ " + letter + ". " + title})
	d.Styled("Center", "**"+esc(title)+"**")
	return letter
}

var appLetters = []string{"А", "Б", "В", "Г", "Д", "Е", "Ж", "И", "К"}

// Markdown — итоговый текст.
func (d *Doc) Markdown() string {
	n := "—"
	if d.pages > 0 {
		n = fmt.Sprint(d.pages)
	}
	return strings.ReplaceAll(strings.Replace(d.b.String(), tocMarker, d.tocXML(), 1), pagesMarker, n)
}

// pagesMarker — место числа листов в поле NUMPAGES титула (заполняется по рендеру; Word пересчитает поле сам).
const pagesMarker = "@@NUMPAGES@@"

// ------------------------------------------------------------------ текст студента

var reSection = regexp.MustCompile(`(?m)^##\s+([A-Za-z0-9_.-]+)`)

// ParseStudent разбирает pzN.md: разделы «## ид …», подсказки «> ✍ …» и комментарии <!-- --> не входят в текст.
func ParseStudent(src string) map[string]string {
	src = regexp.MustCompile(`(?s)<!--.*?-->`).ReplaceAllString(src, "")
	out := map[string]string{}
	idx := reSection.FindAllStringSubmatchIndex(src, -1)
	for i, m := range idx {
		id := src[m[2]:m[3]]
		end := len(src)
		if i+1 < len(idx) {
			end = idx[i+1][0]
		}
		body := src[m[1]:end]
		if j := strings.IndexByte(body, '\n'); j >= 0 {
			body = body[j+1:] // остаток строки заголовка — название раздела
		} else {
			body = ""
		}
		var keep []string
		for _, l := range strings.Split(body, "\n") {
			if strings.HasPrefix(strings.TrimSpace(l), "> ✍") {
				continue
			}
			keep = append(keep, l)
		}
		out[id] = strings.TrimSpace(strings.Join(keep, "\n"))
	}
	return out
}

// StudentTemplate — файл для студента: все места «ДОПИШИ» в порядке документа; уже написанный текст сохраняется,
// разделы, которых в документе больше нет, уходят в конец с пометкой (текст не теряется).
func StudentTemplate(existing string, fills []Fill, doc string) string {
	old := ParseStudent(existing)
	var b strings.Builder
	fmt.Fprintf(&b, "<!-- Текст студента для %s. Каждый раздел начинается строкой «## ид — название»: ид не менять.\n", doc)
	b.WriteString("     Строки «> ✍ …» — подсказки, в документ не попадают. Пиши под подсказкой обычными абзацами;\n")
	b.WriteString("     markdown можно: **жирный**, *курсив*, списки через «- ». Пустой раздел → в документе жёлтое «ДОПИШИ».\n")
	b.WriteString("     Файл можно пересоздать (make pz-init): написанный текст сохранится. -->\n")
	used := map[string]bool{}
	for _, f := range fills {
		used[f.ID] = true
		fmt.Fprintf(&b, "\n## %s — %s\n> ✍ %s\n\n", f.ID, f.Title, f.Hint)
		if t := old[f.ID]; t != "" {
			b.WriteString(t + "\n")
		}
	}
	var orphans []string
	for id, t := range old {
		if !used[id] && t != "" {
			orphans = append(orphans, id)
		}
	}
	sort.Strings(orphans)
	for _, id := range orphans {
		fmt.Fprintf(&b, "\n## %s — НЕ ИСПОЛЬЗУЕТСЯ (такого места в документе больше нет — перенеси текст или удали)\n\n%s\n", id, old[id])
	}
	return b.String()
}

// Save пишет файл, создавая папки.
func Save(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
