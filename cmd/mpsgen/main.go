// mpsgen — генератор материалов курсовой «МПС ч.2» под вариант.
//
//	mpsgen -student students/a12_ryazantsev_iv/variant.yaml -render
//	mpsgen -group А-17 -m 16 -name "Иванов И.И." -render   # любой вариант без файла студента
//
// Результат — в build/<группа>/<вариант>/ (например build/А-17-23/16/): params.md, params.json, vars.inc,
// schematic.kicad_sch (+ .kicad_pro, ramka.kicad_wks); с -render ещё PDF/PNG и ERC (scripts/render.sh, kicad-cli).
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"mpskp/internal/render"
	"mpskp/internal/schgen"
	"mpskp/internal/variant"
)

func main() {
	var (
		table  = flag.String("table", "data/table-2026.yaml", "таблица вариантов из ТЗ")
		stud   = flag.String("student", "", "students/<ник>/variant.yaml")
		group  = flag.String("group", "", "группа (вместо -student)")
		m      = flag.Int("m", 0, "номер варианта (вместо -student)")
		out    = flag.String("out", "", "папка для результатов (по умолчанию <outroot>/<группа>/<вариант>)")
		root   = flag.String("outroot", "build", "корень для результатов")
		name   = flag.String("name", "", "Фамилия И.О. студента в рамку (вместо name в variant.yaml)")
		chk    = flag.String("checker", "", "Фамилия И.О. преподавателя (по умолчанию Михалин С.Н.)")
		year   = flag.String("year", "23", "год набора группы: А-12 → А-12-<year>")
		doRend = flag.Bool("render", false, "после генерации запустить scripts/render.sh (PDF, PNG, ERC)")
		style  = flag.String("style", "", "стиль листа A|B|C|D; пусто или auto — собирается из признаков по группе и варианту (internal/schgen/style.go)")
		nick   = flag.Bool("nick", false, "только напечатать ник студента по -group и -name (имя папки students/<ник>/)")
		info   = flag.Bool("info", false, "только напечатать JSON {nick, dest, build} для -student (куда класть результаты)")
		verify = flag.String("verify", "", "сверить meta.json готовой схемы с -student: расходится — код 3 и что именно")
		remark = flag.String("remark", "", "записать замечание руководителя в students/<ник>/remarks.md (с -student, -km, -by)")
		km     = flag.String("km", "", "к какому КМ замечание: 1…4 (для -remark)")
		by     = flag.String("by", "", "кто сделал замечание (для -remark; пусто — проверяющий из variant.yaml)")
		plain  = flag.Bool("plain", false, "без «почерка» (сдвигов и вариаций шрифтов) — как эталон")
		lib    = flag.String("lib", "masters/lib/mps.kicad_sym", "библиотека символов (пусто — без схемы)")
		wks    = flag.String("wks", "masters/gost_ramka.kicad_wks", "рамка ГОСТ")
	)
	flag.Parse()
	if *nick {
		n := variant.Nick(*group, *name)
		if *group == "" || n == "" {
			fmt.Fprintln(os.Stderr, "нужны -group и -name: make nick G=А-12 FIO=\"Рязанцев И.В.\"")
			os.Exit(2)
		}
		fmt.Println(n)
		return
	}
	if *remark != "" {
		st, err := loadStudent(*stud, *group, *m, *name, *chk, *style, *year)
		if err == nil && st.Path == "" {
			err = fmt.Errorf("нужен -student students/<ник>/variant.yaml")
		}
		if err == nil {
			err = addRemark(st, *km, *by, *remark, time.Now())
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "ошибка:", err)
			os.Exit(1)
		}
		return
	}
	if *info || *verify != "" {
		st, err := loadStudent(*stud, *group, *m, *name, *chk, *style, *year)
		if err != nil {
			fmt.Fprintln(os.Stderr, "ошибка:", err)
			os.Exit(1)
		}
		if *info {
			js, _ := json.Marshal(map[string]string{"nick": variant.Nick(st.Group, st.Name), "dest": st.Dest(), "build": filepath.Join(*root, st.Dir())})
			fmt.Println(string(js))
			return
		}
		mt, err := variant.LoadMeta(*verify)
		if err != nil {
			fmt.Fprintln(os.Stderr, "нет данных схемы:", err)
			os.Exit(1)
		}
		if d := mt.Stale(st.Meta("")); len(d) > 0 {
			fmt.Fprintln(os.Stderr, "схема собрана с другими данными, чем сейчас в variant.yaml:\n  "+strings.Join(d, "\n  ")+
				"\nпрогони «КМ-1 схема и перечень» для этого ника — будет новая СХЕМА-N")
			os.Exit(3)
		}
		return
	}
	plainSheet = *plain
	dir, err := run(*table, *stud, *group, *m, *name, *chk, *style, *year, *root, *out, *lib, *wks)
	if err == nil && *doRend {
		cmd := exec.Command("scripts/render.sh", dir)
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		err = cmd.Run()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "ошибка:", err)
		os.Exit(1)
	}
}

// loadStudent — студент из variant.yaml или из флагов (-group/-m/-name), с умолчаниями.
func loadStudent(studPath, group string, m int, name, checker, style, year string) (*variant.Student, error) {
	var st *variant.Student
	if studPath != "" {
		var err error
		if st, err = variant.LoadStudent(studPath); err != nil {
			return nil, err
		}
	} else {
		st = &variant.Student{Group: group, M: m}
	}
	if name != "" {
		st.Name = name
	}
	if checker != "" {
		st.Checker = checker
	}
	if style != "" {
		st.Style = style
	}
	st.Defaults(year, time.Now())
	return st, nil
}

func run(tablePath, studPath, group string, m int, name, checker, style, year, root, out, libPath, wksPath string) (string, error) {
	tb, err := variant.LoadTable(tablePath)
	if err != nil {
		return "", err
	}
	st, err := loadStudent(studPath, group, m, name, checker, style, year)
	if err != nil {
		return "", err
	}
	if out == "" {
		out = filepath.Join(root, st.Dir())
	}
	p, err := variant.Compute(tb, st)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return "", err
	}
	js, _ := json.MarshalIndent(p, "", "  ")
	files := map[string]string{
		"params.md":   render.Markdown(p),
		"params.json": string(js) + "\n",
		"vars.inc":    render.Asm(p),
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(out, name), []byte(body), 0o644); err != nil {
			return "", err
		}
	}
	fmt.Printf("%s, вариант %d, %s → %s: params.md, params.json, vars.inc\n", st.GroupFull, p.M, st.Name, out)
	for _, w := range p.Warnings {
		fmt.Println("  ⚠", w)
	}
	if libPath == "" {
		return out, nil
	}
	return out, schematic(p, st, out, libPath, wksPath)
}

var plainSheet bool

// schematic рисует схему Э3 с нуля построителем (internal/schgen).
func schematic(p *variant.Params, st *variant.Student, out, libPath, wksPath string) error {
	lib, err := schgen.LoadLib(libPath)
	if err != nil {
		return err
	}
	v := schgen.Variant{
		Cols: p.Cols, Rows: p.Rows, Anode: p.Indicator == "anode",
		CS: schgen.CSPins{
			Buf: p.Dev("Буфер (IDT7005)").CS, Y2: p.Dev("Регистр Y2").CS,
			Kb: p.Dev("Клавиатура").CS, Ind: p.Dev("Индикатор").CS,
		},
		Y1: p.Y1Pin, Y2: p.Y2Pin, KbInt: p.KbInt, X2Int: p.X2Int,
		Decoder: p.CSMode == "decoder", CSEn: p.CSEnPin, Filter: p.FilterCap,
		BufLines: int(math.Ceil(math.Log2(float64(p.V)))), // адресных линий у IDT7005 по объёму буфера V
		Style:    schgen.PickStyle(st.Style, fmt.Sprintf("%s|%d", st.GroupFull, p.M)),
		Date:     st.Date, Student: st.Name, Checker: st.Checker,
	}
	if !plainSheet {
		j := schgen.MakeJitter(fmt.Sprintf("%s|%d", st.GroupFull, p.M), p.Rows)
		v.Jitter = &j
	}
	// правки по замечаниям: students/<ник>/schema/fixes.yaml
	if fp := st.FixesPath(); fp != "" {
		if b, err := os.ReadFile(fp); err == nil {
			var fx schgen.Fixes
			if err := yaml.Unmarshal(b, &fx); err != nil {
				return fmt.Errorf("%s: %w", fp, err)
			}
			v.Fixes = &fx
			fmt.Println("  правки схемы:", fp)
		}
	}
	sh := schgen.Build(lib, v, fmt.Sprintf("%s-%d", p.Group, p.M))
	if len(sh.FixErrs) > 0 {
		return fmt.Errorf("%s:\n  %s", st.FixesPath(), strings.Join(sh.FixErrs, "\n  "))
	}
	regions := sh.Regions() // до String(): тот сдвигает лист и обнуляет сдвиг
	jit := sh.J
	writes := map[string]string{
		"schematic.kicad_sch": sh.String(),
		"schematic.kicad_pro": schgen.Project,
	}
	if v.Style.FullFrame {
		wksPath = strings.TrimSuffix(wksPath, ".kicad_wks") + "_full.kicad_wks"
	}
	w, err := os.ReadFile(wksPath)
	if err != nil {
		return err
	}
	wks, ok := schgen.PatchWks(string(w), render.VariantLine(st.GroupFull, p.M))
	if !ok {
		fmt.Println("  ⚠ в рамке не нашлась строка «Группа …, Э3» — вариант в рамку не вписан")
	}
	writes["ramka.kicad_wks"] = wks
	// перечень элементов (ГОСТ 2.701): полный — для ПЗ1, черновик КМ-1 — только микросхемы и разъёмы (ТЗ, разд. 3)
	peLine := strings.Replace(render.VariantLine(st.GroupFull, p.M), "Э3", "ПЭ3", 1)
	full, km1 := sh.BOM(true), sh.BOM(false)
	for _, set := range []struct {
		name  string
		lines []schgen.BOMLine
	}{{"perechen", full}, {"perechen-km1", km1}} {
		pages := schgen.PE3(set.lines, fmt.Sprintf("%s-%d-%s", p.Group, p.M, set.name), sh.Title)
		for i, pg := range pages {
			writes[fmt.Sprintf("%s-%d.kicad_sch", set.name, i+1)] = pg.String()
			writes[fmt.Sprintf("%s-%d.kicad_wks", set.name, i+1)] = schgen.PE3Frame(i == 0, i+1, len(pages), peLine, v.Style.FullFrame)
		}
	}
	// для ПЗ1 (mpspz -doc pz1): рамки узлов для вырезок, позиционные обозначения по ролям, полный перечень
	refs := map[string][2]string{}
	for role, c := range sh.Roles {
		refs[role] = [2]string{c.Ref, c.Value}
	}
	// style и jitter — какой вид листа вышел (для правок move в fixes.yaml видно, где что стоит)
	sj, _ := json.MarshalIndent(map[string]any{"regions": regions, "refs": refs, "bom": full, "paper_mm": [2]float64{420, 297},
		"style": v.Style, "jitter": jit}, "", "  ")
	writes["sheet.json"] = string(sj)
	mj, _ := json.MarshalIndent(st.Meta(v.Style.Name), "", "  ")
	writes["meta.json"] = string(mj) + "\n"
	writes["perechen.md"] = schgen.BOMMarkdown(full, "Перечень элементов — "+peLine) + "\n" +
		schgen.BOMMarkdown(km1, "Черновик для КМ-1 (только микросхемы и разъёмы)")
	for name, body := range writes {
		if err := os.WriteFile(filepath.Join(out, name), []byte(body), 0o644); err != nil {
			return err
		}
	}
	ind := map[bool]string{true: "общий анод", false: "общий катод"}[v.Anode]
	fmt.Printf("  схема: %s (стиль %s, клавиатура %dx%d, %s)\n", filepath.Join(out, "schematic.kicad_sch"), v.Style.Name, p.Cols, p.Rows, ind)
	return nil
}

// addRemark дописывает замечание руководителя в students/<ник>/remarks.md (файл создаётся с шапкой: ФИО, группа, вариант).
func addRemark(st *variant.Student, km, by, text string, now time.Time) error {
	path := filepath.Join(filepath.Dir(st.Path), "remarks.md")
	if by == "" {
		by = st.Checker
	}
	km = strings.TrimPrefix(strings.TrimSpace(km), "КМ-")
	var b strings.Builder
	if _, err := os.Stat(path); err != nil {
		fmt.Fprintf(&b, "# Замечания руководителя — %s, %s, вариант %d\n\n", st.Name, st.GroupFull, st.M)
		b.WriteString("Записи — сверху вниз по дате. Как работать: AGENTS.md, раздел «Замечания руководителя».\n")
	}
	kmS := "КМ-?"
	if km != "" {
		kmS = "КМ-" + km
	}
	fmt.Fprintf(&b, "\n## %s — %s — %s\n\n", now.Format("2006-01-02"), kmS, by)
	fmt.Fprintf(&b, "- **Студент:** %s, %s, вариант %d\n", st.Name, st.GroupFull, st.M)
	fmt.Fprintf(&b, "- **Замечание:** %s\n", strings.TrimSpace(text))
	b.WriteString("- **Что сделали:** — (дописать: что поправили и где — schema/fixes.yaml, текст ПЗ, код; какой прогон: СХЕМА-N / ПЗ1-K / ПЗ2-K)\n")
	b.WriteString("- **Статус:** открыто\n")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(b.String())
	if err == nil {
		fmt.Println("записано:", path)
	}
	return err
}
