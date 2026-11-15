package schgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDrawings(t *testing.T) {
	lib, err := LoadLib("../../masters/lib/mps.kicad_sym")
	if err != nil {
		t.Fatal(err)
	}
	v := variants[3]
	v.Style = PickStyle("B", "fragments")
	j := MakeJitter("fragments", v.Rows)
	v.Jitter = &j
	sheet := Build(lib, v, "fragments")
	full, fragments := sheet.Drawings()
	if want := Build(lib, v, "fragments").String(); full != want {
		t.Fatal("построение рисунков изменило исходный лист")
	}
	for _, name := range []string{"mcu", "dec", "kb", "ind", "idt", "y2", "power"} {
		root, err := Parse(fragments[name])
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(root.All("bus"))+len(root.All("bus_entry"))+len(root.All("text")) != 0 {
			t.Errorf("%s: остались шины или надписи полного листа", name)
		}
		if len(root.All("symbol")) == 0 || len(root.All("wire")) == 0 {
			t.Errorf("%s: пропали компоненты или проводники", name)
		}
	}
	idt := fragments["idt"]
	root, err := Parse(idt)
	if err != nil {
		t.Fatal(err)
	}
	symbols := map[string]bool{}
	for _, node := range root.All("symbol") {
		symbols[node.Find("uuid").Arg(0)] = true
	}
	for _, role := range []string{"idt", "orBuf", "xsX2", "rX2"} {
		if !symbols[sheet.Roles[role].node.Find("uuid").Arg(0)] {
			t.Errorf("на рисунке буфера нет %s", role)
		}
	}
	if symbols[sheet.Roles["latchY2"].node.Find("uuid").Arg(0)] {
		t.Error("на рисунок буфера попал регистр Y2")
	}
	for _, name := range []string{nCSbuf, nWR, "X2_0", nX2stb} {
		if !strings.Contains(idt, "(label "+Q(name).String()) {
			t.Errorf("на рисунке буфера нет метки %s", name)
		}
	}
}

func TestSavedFragments(t *testing.T) {
	lib, err := LoadLib("../../masters/lib/mps.kicad_sym")
	if err != nil {
		t.Fatal(err)
	}
	v := variants[3]
	v.Style = PickStyle("B", "fragments")
	j := MakeJitter("fragments", v.Rows)
	v.Jitter = &j
	sheet := Build(lib, v, "saved-fragments")
	regions := sheet.Regions()
	refs := map[string][2]string{}
	for role, c := range sheet.Roles {
		if !strings.HasPrefix(role, "filter") {
			refs[role] = [2]string{c.Ref, c.Value}
		}
	}
	full, want := sheet.Drawings()
	path := filepath.Join(t.TempDir(), "schematic.kicad_sch")
	write := func(drawing string) *SavedSheet {
		t.Helper()
		if err := os.WriteFile(path, []byte(drawing), 0o644); err != nil {
			t.Fatal(err)
		}
		saved, err := LoadSavedSheet(path)
		if err != nil {
			t.Fatal(err)
		}
		return saved
	}

	saved := write(full)
	for group, got := range saved.Fragments(refs, regions) {
		if got != want[group] {
			t.Errorf("рисунок %s из архива отличается от оригинала", group)
		}
		delete(want, group)
	}
	if len(want) != 0 {
		t.Errorf("не все рисунки построены: осталось %d", len(want))
	}
	if !v.Decoder {
		t.Fatal("нужен вариант с дешифратором")
	}
	if err := saved.CheckBuffer(refs, v.BufLines); err != nil {
		t.Fatal(err)
	}
	// Подмена строба на входе записи DD меняет цепь, но сохраняет позиционные обозначения и метаданные.
	changed := strings.ReplaceAll(full, `(label "~{WR}"`, `(label "~{X2stb}"`)
	if err := write(changed).CheckBuffer(refs, v.BufLines); err == nil {
		t.Fatal("архив со стробом X2 вместо WR принят")
	}
}
