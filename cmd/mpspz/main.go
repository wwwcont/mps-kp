// mpspz — пояснительные записки по варианту студента.
//
//	mpspz -student students/<ник>/variant.yaml -doc pz2          → build/<группа>/<M>/pz2/«Фамилия ИО ПЗ2.docx»
//	mpspz -student … -doc pz2 -init                              → students/<ник>/pz/pz2.md (места «ДОПИШИ») + flow/*.flow
//
// Генерируется всё, что можно посчитать и нарисовать точно; места для текста студента — в students/<ник>/pz/pzN.md.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"mpskp/internal/codecheck"
	"mpskp/internal/diag"
	"mpskp/internal/flow"
	"mpskp/internal/pz"
	"mpskp/internal/render"
	"mpskp/internal/schgen"
	"mpskp/internal/variant"
)

func main() {
	var (
		table   = flag.String("table", "data/table-2026.yaml", "таблица вариантов")
		student = flag.String("student", "", "students/<ник>/variant.yaml")
		year    = flag.String("year", "23", "год набора группы")
		doc     = flag.String("doc", "pz2", "pz1 | pz2")
		outroot = flag.String("outroot", "build", "корень результатов")
		initF   = flag.Bool("init", false, "создать/дополнить students/<ник>/pz/<doc>.md и заготовки схем алгоритмов")
		fontDir = flag.String("fonts", "fonts", "папка шрифтов")
		dstyle  = flag.String("style", "", "вид документа A | B | C (пусто — от ФИО)")
		plain   = flag.Bool("plain", false, "схемы алгоритмов в виде по умолчанию (GOST type A), без вариаций от ФИО")
	)
	flag.Parse()
	if *student == "" {
		die(fmt.Errorf("нужен -student students/<ник>/variant.yaml"))
	}
	tb, err := variant.LoadTable(*table)
	die(err)
	st, err := variant.LoadStudent(*student)
	die(err)
	st.Defaults(*year, time.Now())
	p, err := variant.Compute(tb, st)
	die(err)
	sdir := filepath.Dir(*student)
	pzdir := filepath.Join(sdir, "pz")
	sheetDir := filepath.Join(*outroot, st.Dir())
	out := filepath.Join(sheetDir, *doc)
	varsInc := render.Asm(p)
	var schema string
	if !*initF {
		p, varsInc, err = render.ArchivedVariant(sheetDir, p)
		die(err)
		b, err := os.ReadFile(filepath.Join(sheetDir, ".schema"))
		if err != nil && !os.IsNotExist(err) {
			die(err)
		}
		schema = strings.TrimSpace(string(b))
	}
	if !*initF {
		os.RemoveAll(out) // прошлая сборка (старые рисунки)
	}
	die(os.MkdirAll(out, 0o755))
	fst := flow.PickStyle(fmt.Sprintf("%s|%d", st.GroupFull, p.M))
	if *plain {
		fst = flow.Default
	}
	font, err := os.ReadFile(filepath.Join(*fontDir, fst.Font))
	die(err)

	textPath := filepath.Join(pzdir, *doc+".md")
	txt, _ := os.ReadFile(textPath)
	d := pz.NewDoc(out, pz.ParseStudent(string(txt)))
	d.FontDir = *fontDir
	if *dstyle == "" {
		*dstyle = st.PZStyle
	}
	d.Style = pz.PickDocStyle(*dstyle, fmt.Sprintf("%s|%d", st.GroupFull, p.M))

	var warns []string // предупреждения по файлам студента (в report.md и карточку Issue)
	switch *doc {
	case "pz2":
		if *initF {
			initFlows(p, filepath.Join(pzdir, "flow"))
		}
		in := pz.PZ2{P: p, GroupFull: st.GroupFull, Checker: st.Checker, Year: time.Now().Year(), VarsInc: varsInc, Schema: schema}
		if schema != "" {
			var sj struct{ Refs map[string][2]string }
			data, err := os.ReadFile(filepath.Join(sheetDir, "sheet.json"))
			die(err)
			die(json.Unmarshal(data, &sj))
			in.Refs = sj.Refs
		}
		for n := 1; n <= 3; n++ {
			f := filepath.Join(sdir, "code", fmt.Sprintf("prog%d.a51", n))
			src, err := os.ReadFile(f)
			if err != nil {
				fmt.Fprintf(os.Stderr, "нет %s — программа %d в ПЗ2 не попадёт\n", f, n)
				continue
			}
			r := codecheck.Check(codecheck.Input{Params: p, Prog: n, Src: string(src), Name: filepath.Base(f), VarsInc: varsInc, Seed: n})
			in.Programs = append(in.Programs, pz.Program{N: n, Src: string(src), Report: r})
		}
		flows, _ := filepath.Glob(filepath.Join(pzdir, "flow", "*.flow"))
		sort.Strings(flows)
		for _, f := range flows {
			src, err := os.ReadFile(f)
			die(err)
			// ошибка в схеме студента не роняет ПЗ2: схема пропускается, ошибка — в предупреждения (report.md, карточка Issue)
			c, err := flow.Parse(string(src))
			if err != nil {
				warns = append(warns, fmt.Sprintf("схема `%s` не разобрана и в ПЗ2 не вошла: %v — исправьте pz/flow/%s", filepath.Base(f), err, filepath.Base(f)))
				continue
			}
			img, err := flow.Render(c, flow.Options{Font: font, Style: fst})
			if err != nil {
				warns = append(warns, fmt.Sprintf("схема `%s` не нарисована: %v", filepath.Base(f), err))
				continue
			}
			id := strings.TrimSuffix(filepath.Base(f), ".flow")
			pngName := "flow-" + id + ".png"
			w, err := os.Create(filepath.Join(out, pngName))
			die(err)
			die(png.Encode(w, img))
			w.Close()
			title := c.Title
			if title == "" {
				title = id
			}
			// натуральный размер (8 px на мм), не шире поля страницы и не выше листа (23 см под рисунок с подписью):
			// длинную схему (вложенный цикл не режется на колонки) уменьшаем и предупреждаем — лучше разбить или упростить
			wcm := min(float64(img.Bounds().Dx())/8/10, 17)
			hcm := float64(img.Bounds().Dy()) / 8 / 10 * wcm / (float64(img.Bounds().Dx()) / 8 / 10)
			if hcm > 23 {
				wcm *= 23 / hcm
				warns = append(warns, fmt.Sprintf("схема `%s` выше листа (%.0f мм) — уменьшена до высоты страницы; лучше разбить её "+
					"на подпрограммы или упростить вложенные циклы (pz/flow/%s.flow)", id, hcm*10, id))
			}
			in.Flows = append(in.Flows, pz.FlowFig{ID: id, Title: title, File: pngName, WidthCm: wcm, Text: flow.Describe(c)})
		}
		pz.BuildPZ2(d, in)
	case "pz1":
		pz1(d, p, st, sheetDir, out, font, fst)
		if !*initF {
			bom, err := os.ReadFile(filepath.Join(sheetDir, "perechen.pdf"))
			if err == nil {
				die(pz.Save(filepath.Join(out, "perechen.pdf"), bom))
			} else if schema != "" || !os.IsNotExist(err) {
				die(err)
			}
		}
	default:
		die(fmt.Errorf("документ %q не поддержан (pz1 | pz2)", *doc))
	}

	if *initF {
		die(pz.Save(textPath, []byte(pz.StudentTemplate(string(txt), d.Fills, strings.Replace(*doc, "pz", "ПЗ", 1)))))
		fmt.Printf("%s — места «ДОПИШИ»: %d\n", textPath, len(d.Fills))
		return
	}
	name := docName(p.Student, strings.ToUpper(strings.Replace(*doc, "pz", "ПЗ", 1)))
	die(pz.Build(d, filepath.Join(out, name)))
	// номера страниц в содержании — по рендеру LibreOffice (если он есть; в CI ставится)
	if err := pz.FillTOCPages(d, filepath.Join(out, name)); err != nil {
		fmt.Fprintf(os.Stderr, "содержание без номеров страниц (%v) — Word проставит их при открытии\n", err)
	} else {
		die(pz.Build(d, filepath.Join(out, name)))
		// PDF «для просмотра» — как документ выглядит в LibreOffice (встроенные просмотрщики docx упрощённые);
		// свой PDF студент делает сам из docx, когда допишет
		pdf := filepath.Join(out, strings.TrimSuffix(name, ".docx")+" — просмотр.pdf")
		if err := pz.RenderPDF(filepath.Join(out, name), pdf); err != nil {
			fmt.Fprintf(os.Stderr, "PDF для просмотра не собран: %v\n", err)
		}
	}
	left := 0
	var todo strings.Builder
	for _, f := range d.Fills {
		if !f.Done {
			left++
			fmt.Fprintf(&todo, "- `%s` — %s\n", f.ID, f.Title)
		}
	}
	rep := fmt.Sprintf("# %s: %s, %s, вариант %d\n\nФайл: `%s`. Мест «ДОПИШИ»: %d, осталось заполнить: %d (students/<ник>/pz/%s.md).\n\n%s",
		strings.Replace(*doc, "pz", "ПЗ", 1), p.Student, st.GroupFull, p.M, name, len(d.Fills), left, *doc, todo.String())
	if len(warns) > 0 {
		rep += "\n## Предупреждения\n\n- " + strings.Join(warns, "\n- ") + "\n"
	}
	die(os.WriteFile(filepath.Join(out, "report.md"), []byte(rep), 0o644))
	fmt.Print(rep)
	if s := os.Getenv("GITHUB_STEP_SUMMARY"); s != "" {
		if f, err := os.OpenFile(s, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644); err == nil {
			f.WriteString(rep + "\n")
			f.Close()
		}
	}
}

// docName — «Рязанцев ИВ ПЗ2.docx» (как в ТЗ: «Михалин СН ПЗ2-v1.pdf»).
func docName(fio, doc string) string {
	f := strings.Fields(fio)
	name := "Фамилия ИО"
	if len(f) > 0 {
		name = f[0]
		if len(f) > 1 {
			name += " " + strings.NewReplacer(".", "", " ", "").Replace(strings.Join(f[1:], ""))
		}
	}
	return name + " " + doc + ".docx"
}

func die(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "mpspz:", err)
		os.Exit(2)
	}
}

// initFlows — заготовки схем алгоритмов, если папка пуста (существующие не трогаются).
func initFlows(p *variant.Params, dir string) {
	if old, _ := filepath.Glob(filepath.Join(dir, "*.flow")); len(old) > 0 {
		fmt.Printf("%s: схемы уже есть (%d) — не трогаю\n", dir, len(old))
		return
	}
	for name, b := range pz.DefaultFlows(p) {
		die(pz.Save(filepath.Join(dir, name), b))
		fmt.Printf("%s — заготовка схемы алгоритма\n", filepath.Join(dir, name))
	}
}

// pz1 — ПЗ1 по результатам КМ-1 (build/<группа>/<M>/: sheet.json, schematic.png — make student / джоба КМ-1).
func pz1(d *pz.Doc, p *variant.Params, st *variant.Student, sheetDir, out string, font []byte, fst flow.Style) {
	var sj struct {
		Regions map[string]schgen.Rect
		Refs    map[string][2]string
		BOM     []pz.BOMLine
		Paper   [2]float64 `json:"paper_mm"`
	}
	b, err := os.ReadFile(filepath.Join(sheetDir, "sheet.json"))
	if err != nil {
		die(fmt.Errorf("нет %s — сначала схема: make student S=<ник> (джоба КМ-1)", filepath.Join(sheetDir, "sheet.json")))
	}
	die(json.Unmarshal(b, &sj))
	src, err := readPNG(filepath.Join(sheetDir, "schematic.png"))
	if err != nil {
		die(fmt.Errorf("нет схемы PNG (%v) — сначала make student S=<ник> (нужен KiCad) или джоба КМ-1", err))
	}
	// дата на титуле — как в рамке схемы, по которой собирается ПЗ1 (meta.json из СХЕМА-N)
	date := st.Date
	if mt, err := variant.LoadMeta(filepath.Join(sheetDir, "meta.json")); err == nil && mt.Date != "" {
		date = mt.Date
	}
	in := pz.PZ1{P: p, GroupFull: st.GroupFull, Checker: st.Checker, Year: time.Now().Year(), Date: date, Refs: sj.Refs, BOM: sj.BOM,
		Figs: map[string]string{}, FigW: map[string]float64{}}
	W := float64(src.Bounds().Dx())
	pxmm := W / sj.Paper[0]
	var fragments map[string]string
	for name := range sj.Regions {
		fragment, err := readPNG(filepath.Join(sheetDir, "fragment-"+name+".png"))
		if os.IsNotExist(err) {
			if fragments == nil {
				saved, err := schgen.LoadSavedSheet(filepath.Join(sheetDir, "schematic.kicad_sch"))
				die(err)
				fragments = saved.Fragments(sj.Refs, sj.Regions)
			}
			fragment, err = renderFragment(sheetDir, name, fragments[name])
		}
		die(err)
		rect := inkBounds(fragment).Inset(-int(pxmm * 2)).Intersect(fragment.Bounds())
		sub := image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
		draw.Draw(sub, sub.Bounds(), fragment, rect.Min, draw.Src)
		f := "sch-" + name + ".png"
		die(writePNG(filepath.Join(out, f), sub))
		in.Figs[name] = f
		in.FigW[name] = math.Min(16.5, float64(rect.Dx())/pxmm/10*1.1)
	}
	die(writePNG(filepath.Join(out, "sheet.png"), rotate90(src)))
	in.Figs["sheet"] = "sheet.png"
	fmm, stroke := fst.FontMM, fst.Stroke
	s1, err := diag.Structural(font, fmm, stroke)
	die(err)
	die(writePNG(filepath.Join(out, "struct.png"), s1))
	in.Figs["struct"] = "struct.png"
	dev := func(name string) string { return p.Dev(name).CS }
	f1, err := diag.Functional(diag.FuncInfo{MCU: sj.Refs["mcu"][0], Latch: sj.Refs["latchA"][0], Dec: sj.Refs["dec"][0],
		Buf: sj.Refs["idt"][0], KbCol: sj.Refs["kb173"][0], KbRow: sj.Refs["buf"][0], IndReg: sj.Refs["latchInd"][0],
		Ind: sj.Refs["hg"][0], X2: sj.Refs["xsX2"][0], Out: sj.Refs["xsY"][0], Y2Reg: sj.Refs["latchY2"][0], CSBuf: dev("Буфер (IDT7005)"), CSKb: dev("Клавиатура"), CSInd: dev("Индикатор"), CSY2: dev("Регистр Y2"),
		BufAddrLast: int(math.Ceil(math.Log2(float64(p.V)))) - 1, Cols: p.Cols, Rows: p.Rows,
		Keyboard: strings.ReplaceAll(p.Keyboard, "x", "×")}, font, stroke)
	die(err)
	die(writePNG(filepath.Join(out, "func.png"), f1))
	in.Figs["func"] = "func.png"
	tr, tw, err := diag.TimingDiagrams(diag.Timing{TAVLL: 43, TLLAX: 48, TLLWL: 200, TRLRH: 400, TRLDV: 252, TRHDX: 0, TQVWH: 433, TWHQX: 33, TWLWH: 400,
		MemRead: "OEL", MemWrite: "R/WR"}, font, fmm, stroke)
	die(err)
	die(writePNG(filepath.Join(out, "tread.png"), tr))
	die(writePNG(filepath.Join(out, "twrite.png"), tw))
	in.Figs["tread"], in.Figs["twrite"] = "tread.png", "twrite.png"
	pz.BuildPZ1(d, in)
}

func renderFragment(sheetDir, name, drawing string) (image.Image, error) {
	if drawing == "" {
		return nil, fmt.Errorf("в исходнике Э3 нет элементов для рисунка %s", name)
	}
	dir, err := os.MkdirTemp("", "mps-fragment-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	base := filepath.Join(dir, "schematic")
	if err := os.WriteFile(base+".kicad_sch", []byte(drawing), 0o644); err != nil {
		return nil, err
	}
	pro, err := os.ReadFile(filepath.Join(sheetDir, "schematic.kicad_pro"))
	if err == nil {
		err = os.WriteFile(base+".kicad_pro", pro, 0o644)
	}
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	for _, cmd := range []*exec.Cmd{
		exec.Command("kicad-cli", "sch", "export", "pdf", "-b", "-e", "-o", base+".pdf", base+".kicad_sch"),
		exec.Command("pdftoppm", "-png", "-r", "300", "-singlefile", base+".pdf", base),
	} {
		if b, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("рисунок %s: %w\n%s", name, err, b)
		}
	}
	return readPNG(base + ".png")
}

// inkBounds обрезает белые поля отдельного рисунка без рамки.
func inkBounds(img image.Image) image.Rectangle {
	b := img.Bounds()
	ink := image.Rectangle{Min: b.Max, Max: b.Min}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, blue, _ := img.At(x, y).RGBA()
			if r < 0xf000 || g < 0xf000 || blue < 0xf000 {
				ink.Min.X, ink.Min.Y = min(ink.Min.X, x), min(ink.Min.Y, y)
				ink.Max.X, ink.Max.Y = max(ink.Max.X, x+1), max(ink.Max.Y, y+1)
			}
		}
	}
	if ink.Empty() {
		return b
	}
	return ink
}

func readPNG(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return png.Decode(f)
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// rotate90 — поворот на 90° против часовой (лист А3 альбомный → на книжную страницу).
func rotate90(src image.Image) image.Image {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dy(), b.Dx()))
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dst.Set(y-b.Min.Y, b.Max.X-1-x, src.At(x, y))
		}
	}
	return dst
}
