package schgen

import (
	"fmt"
	"sort"
	"strings"
)

// BOMLine — строка перечня элементов (ГОСТ 2.701): поз. обозначение, наименование, кол., примечание.
// Group != "" — строка-заголовок группы («Конденсаторы»).
type BOMLine struct {
	Refs, Name, Note, Group string
	Qty                     int
}

// Наименования — тип без видового слова (вид — заголовок группы или «Примечание»; замечание Гольцова: «розетка — в графу примечание»), по образцу принятого ПЭ3 (Осипова, 2025) и ТЗ-2026 «В работе применять…».
var bomGroups = map[string]string{
	"C": "Конденсаторы", "DD": "Микросхемы", "HG": "Индикаторы", "R": "Резисторы",
	"SB": "Кнопки", "VD": "Диоды", "XS": "Разъёмы", "ZQ": "Резонаторы",
}

// unit: «10 к» → «10 кОм», «33 н» → «33 нФ», «330» → «330 Ом».
func withUnit(prefix, v string) string {
	v = strings.TrimSpace(v)
	switch prefix {
	case "R":
		if strings.HasSuffix(v, "к") || strings.HasSuffix(v, "М") {
			return v + "Ом"
		}
		return v + " Ом"
	case "C":
		return v + "Ф"
	}
	return v
}

func bomName(c *Comp, contacts int) (name, note string) {
	sym := strings.TrimSuffix(c.Sym, "_G")
	switch sym {
	case "AT89S53":
		return "AT89S53-24PU", "Микроконтроллер"
	case "74HC573":
		return c.Value, "Регистр 8-разрядный"
	case "74HC173":
		return c.Value, "Регистр 4-разрядный"
	case "74HC244":
		return "IN74AC244", "Буферный формирователь" // короче: в графу «Примечание» помещается в одну строку
	case "74HC02":
		return c.Value, "4 элемента 2ИЛИ-НЕ"
	case "74HC32":
		return c.Value, "4 элемента 2ИЛИ"
	case "74HC11":
		return c.Value, "3 элемента 3И"
	case "74HC21":
		return c.Value, "2 элемента 4И"
	case "74HC138":
		return c.Value, "Дешифратор 3 на 8"
	case "IDT7005":
		return "IDT7005S55PF", "Двухпортовое ОЗУ 8К×8"
	case "FYS-5612AX":
		return "FYS-5612AX", "Индикатор, общий катод"
	case "FYS-5612BX":
		return "FYS-5612BX", "Индикатор, общий анод"
	case "SW_Push":
		return "DTSM-62N-V", ""
	case "D":
		return "КД521А", ""
	case "Crystal":
		return "HC-49S 12 МГц", "Кварцевый"
	case "R":
		return "МЛТ-0,125 " + withUnit("R", c.Value) + " ±5 %", ""
	case "C_Polarized":
		return "К50-35 " + withUnit("C", c.Value) + " ±20 % 16 В", ""
	case "C":
		return "К10-17Б " + withUnit("C", c.Value) + " ±10 % 50 В", ""
	}
	if strings.HasPrefix(sym, "CONN_") {
		// «Розетка» — в примечание, не в наименование (замечание Гольцова 08.10.2026)
		return fmt.Sprintf("PBS-%d", contacts), fmt.Sprintf("Розетка, %d конт.", contacts)
	}
	return c.Value, ""
}

// bomLine — наименование и примечание с правками студента (Fixes.BOMNames / BOMNotes).
func (s *Sheet) bomLine(c *Comp, contacts int) (name, note string) {
	name, note = bomName(c, contacts)
	if s.Fixes != nil {
		if v, ok := s.Fixes.BOMNames[c.Ref]; ok {
			name = v
		}
		if v, ok := s.Fixes.BOMNotes[c.Ref]; ok {
			note = v
		}
	}
	return
}

// BOM — перечень по листу. full = false — черновик для КМ-1: только микросхемы и разъёмы (ТЗ, разд. 3).
func (s *Sheet) BOM(full bool) []BOMLine {
	type pkg struct {
		c        *Comp
		contacts int
	}
	pk := map[string]*pkg{}
	for _, c := range s.syms {
		if strings.HasPrefix(c.Ref, "#") {
			continue
		}
		p, ok := pk[c.Ref]
		if !ok {
			p = &pkg{c: c}
			pk[c.Ref] = p
		}
		if strings.HasPrefix(c.Sym, "CONN_") {
			p.contacts += len(c.pins)
		}
	}
	var refs []string
	for r := range pk {
		refs = append(refs, r)
	}
	sort.Slice(refs, func(i, j int) bool {
		pi, pj := refPrefix(refs[i]), refPrefix(refs[j])
		if pi != pj {
			return pi < pj
		}
		return refNum(refs[i]) < refNum(refs[j])
	})
	// одинаковые элементы (наименование и примечание) — одной строкой с количеством и перечнем обозначений, даже если номера не подряд
	// (общие требования Гольцова 19.10.2026: «в перечне элементов одинаковые элементы пишутся одной строкой с указанием количества
	// и перечня позиционных обозначений»; так же у принятого листа Казарина: «DD5, DD6, DD11 74AC573 3»)
	var out []BOMLine
	group := ""
	done := map[string]bool{}
	for i := 0; i < len(refs); i++ {
		if done[refs[i]] {
			continue
		}
		p := pk[refs[i]]
		prefix := refPrefix(refs[i])
		if !full && prefix != "DD" && prefix != "XS" {
			continue
		}
		name, note := s.bomLine(p.c, p.contacts)
		var nums []int
		for j := i; j < len(refs); j++ {
			if done[refs[j]] || refPrefix(refs[j]) != prefix {
				continue
			}
			if n2, nt2 := s.bomLine(pk[refs[j]].c, pk[refs[j]].contacts); n2 == name && nt2 == note {
				done[refs[j]] = true
				nums = append(nums, refNum(refs[j]))
			}
		}
		if prefix != group {
			group = prefix
			out = append(out, BOMLine{Group: bomGroups[prefix]})
		}
		out = append(out, BOMLine{Refs: refList(prefix, nums), Name: name, Note: note, Qty: len(nums)})
	}
	return out
}

// BOMMarkdown — перечень таблицей markdown (для просмотра и вставки в ПЗ).
func BOMMarkdown(lines []BOMLine, title string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n| Поз. обозначение | Наименование | Кол. | Примечание |\n| --- | --- | --- | --- |\n", title)
	for _, l := range lines {
		if l.Group != "" {
			fmt.Fprintf(&b, "|  | **%s** |  |  |\n", l.Group)
			continue
		}
		fmt.Fprintf(&b, "| %s | %s | %d | %s |\n", l.Refs, l.Name, l.Qty, l.Note)
	}
	return b.String()
}

// refList — «DD3, DD8, DD11», «C5–C15», «R2, R3»: подряд три и больше — через тире.
func refList(prefix string, nums []int) string {
	var parts []string
	for i := 0; i < len(nums); {
		j := i
		for j+1 < len(nums) && nums[j+1] == nums[j]+1 {
			j++
		}
		switch {
		case j == i:
			parts = append(parts, fmt.Sprintf("%s%d", prefix, nums[i]))
		case j == i+1:
			parts = append(parts, fmt.Sprintf("%s%d", prefix, nums[i]), fmt.Sprintf("%s%d", prefix, nums[j]))
		default:
			parts = append(parts, fmt.Sprintf("%s%d–%s%d", prefix, nums[i], prefix, nums[j]))
		}
		i = j + 1
	}
	return strings.Join(parts, ", ")
}
