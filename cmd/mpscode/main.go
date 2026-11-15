// mpscode — проверка программ курсовой (КМ-3) до отправки роботу.
//
//	mpscode -student students/<ник>/variant.yaml            — проверить students/<ник>/code/prog{1,2,3}.a51
//	mpscode -group А-12 -m 14 -name "Иванов И.И." prog1.a51  — любой файл под любой вариант (номер — из имени progN)
//	mpscode -student … -init                                 — положить заготовки prog1..3.a51 (существующие не трогает)
//	mpscode -asm prog.a51 -vars vars.inc [-hex out.hex | -cmp mide.hex] — только собрать / сверить байты с MCU 8051 IDE
//
// Результат — в build/<группа>/<M>/code/: файлы для робота «Фамилия ИО-код-n.txt», копии исходников, листинги, HEX,
// vars.inc, report.md.
// Код выхода 1 — есть ❌.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"mpskp/internal/asm51"
	"mpskp/internal/codecheck"
	"mpskp/internal/render"
	"mpskp/internal/variant"
)

func main() {
	var (
		table   = flag.String("table", "data/table-2026.yaml", "таблица вариантов")
		student = flag.String("student", "", "students/<ник>/variant.yaml")
		group   = flag.String("group", "", "группа (без -student)")
		m       = flag.Int("m", 0, "вариант M (без -student)")
		name    = flag.String("name", "", "Фамилия И.О. (без -student)")
		year    = flag.String("year", "23", "год набора группы")
		outroot = flag.String("outroot", "build", "корень результатов")
		initT   = flag.Bool("init", false, "положить заготовки программ в students/<ник>/code/")
		noSim   = flag.Bool("nosim", false, "без симулятора (только сборка и правила)")
		asmF    = flag.String("asm", "", "только собрать файл")
		vars    = flag.String("vars", "", "vars.inc для -asm")
		hexOut  = flag.String("hex", "", "куда записать HEX для -asm")
		cmpHex  = flag.String("cmp", "", "для -asm: сверить байты с этим HEX (MCU 8051 IDE)")
	)
	flag.Parse()
	if *asmF != "" {
		os.Exit(assembleOnly(*asmF, *vars, *hexOut, *cmpHex))
	}

	tb, err := variant.LoadTable(*table)
	die(err)
	var st *variant.Student
	var codeDir string
	if *student != "" {
		st, err = variant.LoadStudent(*student)
		die(err)
		codeDir = filepath.Join(filepath.Dir(*student), "code")
	} else {
		if *group == "" || *m == 0 {
			die(fmt.Errorf("нужен -student или -group и -m"))
		}
		st = &variant.Student{Group: *group, M: *m, Name: *name}
	}
	st.Defaults(*year, time.Now())
	p, err := variant.Compute(tb, st)
	die(err)
	varsInc := render.Asm(p)

	if *initT {
		if codeDir == "" {
			die(fmt.Errorf("-init только вместе с -student"))
		}
		die(os.MkdirAll(codeDir, 0o755))
		for n := 1; n <= 3; n++ {
			f := filepath.Join(codeDir, fmt.Sprintf("prog%d.a51", n))
			if _, err := os.Stat(f); err == nil {
				fmt.Printf("%s уже есть — не трогаю\n", f)
				continue
			}
			die(os.WriteFile(f, []byte(codecheck.Template(p, st.GroupFull, n)), 0o644))
			fmt.Printf("%s — заготовка: %s\n", f, codecheck.Title(p, n))
		}
		// vars.inc рядом — для MCU 8051 IDE / Keil при отладке руками (в git не нужен, генерируется)
		die(os.WriteFile(filepath.Join(codeDir, "vars.inc"), []byte(varsInc), 0o644))
		return
	}

	files := flag.Args()
	if len(files) == 0 && codeDir != "" {
		files, _ = filepath.Glob(filepath.Join(codeDir, "prog[123].a51"))
	}
	if len(files) == 0 {
		die(fmt.Errorf("нет файлов progN.a51 (заготовки: -init)"))
	}
	sort.Strings(files)
	dir := filepath.Join(*outroot, st.Dir())
	if *student != "" {
		p, varsInc, err = render.ArchivedVariant(dir, p)
		die(err)
	}
	out := filepath.Join(dir, "code")
	die(os.MkdirAll(out, 0o755))
	die(os.WriteFile(filepath.Join(out, "vars.inc"), []byte(varsInc), 0o644))

	var md strings.Builder
	fmt.Fprintf(&md, "# Проверка программ: %s, %s, вариант %d\n\n", p.Student, st.GroupFull, p.M)
	if schema, err := os.ReadFile(filepath.Join(dir, ".schema")); *student != "" && err == nil {
		fmt.Fprintf(&md, "Схема: %s. Адреса и константы взяты из её vars.inc.\n\n", strings.TrimSpace(string(schema)))
	}
	md.WriteString("Проверено mpscode: сборка (asm51, байты сверяются с MCU 8051 IDE), правила робота (ТЗ-2026, разд. 2.2, рис. 7), сценарии во встроенном эмуляторе 8051 (emu51, сверен с ucsim s51) с моделями устройств схемы. " +
		"Робот МЭИ может проверять иначе — «✅» здесь не гарантия «принято», но «❌» почти наверняка ошибка.\n\n")
	failed := false
	for _, f := range files {
		n := progNum(f)
		if n == 0 {
			fmt.Fprintf(os.Stderr, "%s: номер программы берётся из имени prog1/prog2/prog3\n", f)
			failed = true
			continue
		}
		src, err := os.ReadFile(f)
		die(err)
		r := codecheck.Check(codecheck.Input{Params: p, Prog: n, Src: string(src), Name: filepath.Base(f), VarsInc: varsInc, Seed: n, NoSim: *noSim})
		base := fmt.Sprintf("prog%d", n)
		die(os.WriteFile(filepath.Join(out, base+".a51"), src, 0o644))
		die(os.WriteFile(filepath.Join(out, base+".lst"), []byte(r.Listing), 0o644))
		if r.Worst() != codecheck.Fail || r.Robot != "" {
			die(os.WriteFile(filepath.Join(out, base+".hex"), []byte(r.HEX), 0o644))
		}
		if r.Robot != "" {
			die(os.WriteFile(filepath.Join(out, r.RobotName), []byte(r.Robot), 0o644))
		}
		md.WriteString(r.Markdown())
		md.WriteString("\n")
		if r.Worst() == codecheck.Fail {
			failed = true
		}
	}
	die(os.WriteFile(filepath.Join(out, "report.md"), []byte(md.String()), 0o644))
	fmt.Print(md.String())
	if s := os.Getenv("GITHUB_STEP_SUMMARY"); s != "" {
		if f, err := os.OpenFile(s, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644); err == nil {
			f.WriteString(md.String())
			f.Close()
		}
	}
	fmt.Fprintf(os.Stderr, "результаты: %s\n", out)
	if failed {
		os.Exit(1)
	}
}

var reProg = regexp.MustCompile(`prog([123])`)

func progNum(f string) int {
	if m := reProg.FindStringSubmatch(filepath.Base(f)); m != nil {
		n, _ := strconv.Atoi(m[1])
		return n
	}
	return 0
}

func assembleOnly(file, vars, hexOut, cmpHex string) int {
	src, err := os.ReadFile(file)
	die(err)
	read := func(name string) (string, error) {
		if vars != "" && strings.EqualFold(filepath.Base(name), "vars.inc") {
			b, err := os.ReadFile(vars)
			return string(b), err
		}
		b, err := os.ReadFile(filepath.Join(filepath.Dir(file), name))
		return string(b), err
	}
	r := asm51.Assemble(filepath.Base(file), string(src), read)
	for _, e := range r.Errors {
		fmt.Fprintln(os.Stderr, e)
	}
	if len(r.Errors) > 0 {
		return 1
	}
	if cmpHex != "" {
		b, err := os.ReadFile(cmpHex)
		die(err)
		want, err := asm51.ParseHEX(string(b))
		die(err)
		if d := asm51.Diff(r.Code, want, 20); len(d) > 0 {
			fmt.Fprintf(os.Stderr, "%s: байты отличаются от %s:\n  %s\n", file, cmpHex, strings.Join(d, "\n  "))
			return 1
		}
		fmt.Printf("%s: %d байт — совпадает с %s\n", file, len(r.Code), cmpHex)
		return 0
	}
	if hexOut != "" {
		die(os.WriteFile(hexOut, []byte(r.HEX()), 0o644))
	} else {
		fmt.Print(r.HEX())
	}
	return 0
}

func die(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "mpscode:", err)
		os.Exit(2)
	}
}
