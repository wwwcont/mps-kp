package schgen

import (
	"fmt"
	"strings"
)

// Перечень элементов (ПЭ3) по ГОСТ 2.701 на листах А4: графы 20 / 110 / 10 / 45 мм, шапка 15 мм, строка 8 мм.
// Рисуется тем же KiCad-листом (линии и текст), рамка — gost_ramka с внешним прямоугольником А4 (PE3Wks).
const (
	peX0, peX1, peX2, peX3, peX4 = 20.0, 40.0, 150.0, 160.0, 205.0
	peTop, peHead, peRow         = 5.0, 15.0, 8.0
	peBottom1, peBottomN         = 252.0, 277.0 // верх основной надписи: форма 2 (40 мм) на первом листе, 2а (15 мм) — на остальных
	peFont                       = 2.5
	peNameMax, peNoteMax         = 44, 24 // символов в строке графы (GOST type A 2,5 мм); «×», «→» в шрифте нет — не использовать
)

type peItem struct {
	ref, name, qty string
	note           []string // 1–2 строки в одной клетке (вторая — мельче, как в принятом ПЭ3)
	group          bool
}

// wrap режет текст по словам на строки не длиннее n символов.
func wrap(s string, n int) []string {
	var out []string
	cur := ""
	for _, w := range strings.Fields(s) {
		if cur != "" && len([]rune(cur+" "+w)) > n {
			out = append(out, cur)
			cur = w
			continue
		}
		if cur == "" {
			cur = w
		} else {
			cur += " " + w
		}
	}
	if cur != "" || len(out) == 0 {
		out = append(out, cur)
	}
	return out
}

func peRows(lines []BOMLine) []peItem {
	var rows []peItem
	for i, l := range lines {
		if l.Group != "" {
			if i > 0 {
				rows = append(rows, peItem{}) // пустая строка между группами
			}
			rows = append(rows, peItem{name: l.Group, group: true})
			continue
		}
		names := wrap(l.Name, peNameMax)
		note := []string{l.Note}
		if len([]rune(l.Note)) > peNoteMax {
			note = wrap(l.Note, peNoteMax+6) // две строчки мелким шрифтом в одной клетке
		}
		refs := wrapRefs(l.Refs, 11) // «DD3, DD8,» / «DD11» — в графе обозначения не теснится (как у принятого листа Казарина)
		for k := 0; k < max(len(names), len(refs)); k++ {
			r := peItem{}
			if k < len(names) {
				r.name = names[k]
			}
			if k < len(refs) {
				r.ref = refs[k]
			}
			if k == 0 {
				r.qty, r.note = fmt.Sprint(l.Qty), note
			}
			rows = append(rows, r)
		}
	}
	return rows
}

// PE3 раскладывает перечень по страницам. Возвращает листы (А4) в порядке страниц.
func PE3(lines []BOMLine, seed string, tb TitleBlock) []*Sheet {
	rows := peRows(lines)
	var pages []*Sheet
	for p, start := 0, 0; start < len(rows) || p == 0; p++ {
		bottom := peBottomN
		if p == 0 {
			bottom = peBottom1
		}
		avail := bottom - peTop - peHead
		per := int(avail / peRow)
		s := NewSheet(nil, fmt.Sprintf("%s/pe3-%d", seed, p+1))
		s.A4 = true
		s.Title = tb
		s.peGrid(bottom)
		for start < len(rows) && start > 0 && !rows[start].group && rows[start].name == "" && rows[start].ref == "" {
			start++ // пустая строка-разделитель в начале страницы не нужна
		}
		end := min(len(rows), start+per)
		// заголовок группы и пустая строка не остаются последними на странице — переносятся к своим строкам
		for end < len(rows) && end > start+1 && (rows[end-1].group || rows[end-1].name == "") {
			end--
		}
		for i, r := range rows[start:end] {
			y := peTop + peHead + float64(i)*peRow + peRow/2
			if r.group {
				s.peText(r.name, Pt{peX1 + 1.5, y}, "left", true)
				continue
			}
			s.peText(r.ref, Pt{(peX0 + peX1) / 2, y}, "center", false)
			s.peText(r.name, Pt{peX1 + 1.5, y}, "left", false)
			s.peText(r.qty, Pt{(peX2 + peX3) / 2, y}, "center", false)
			switch len(r.note) {
			case 1:
				s.peText(r.note[0], Pt{peX3 + 1.5, y}, "left", false)
			case 2:
				s.peTextSize(r.note[0], Pt{peX3 + 1.5, y - 1.6}, "left", 1.8)
				s.peTextSize(r.note[1], Pt{peX3 + 1.5, y + 1.6}, "left", 1.8)
			}
		}
		pages = append(pages, s)
		start = end
	}
	return pages
}

func (s *Sheet) peLine(a, b Pt) {
	s.Graphic(L("polyline",
		L("pts", L("xy", F(a.X), F(a.Y)), L("xy", F(b.X), F(b.Y))),
		L("stroke", L("width", F(0.3)), L("type", A("default"))),
		L("fill", L("type", A("none"))),
		L("uuid", Q(s.uuid()))))
}

func (s *Sheet) peText(t string, at Pt, just string, underline bool) {
	s.peTextSize(t, at, just, peFont)
	if underline && t != "" {
		w := float64(len([]rune(t))) * peFont * 0.62
		x0 := at.X
		if just == "center" {
			x0 -= w / 2
		}
		s.peLine(Pt{x0, at.Y + 1.8}, Pt{x0 + w, at.Y + 1.8})
	}
}

func (s *Sheet) peTextSize(t string, at Pt, just string, size float64) {
	if t == "" {
		return
	}
	eff := L("effects", L("font", L("face", Q("GOST type A")), L("size", F(size), F(size)), A("italic")))
	if just != "center" {
		eff.Kids = append(eff.Kids, L("justify", A(just)))
	}
	s.items = append(s.items, L("text", Q(t),
		L("exclude_from_sim", A("no")),
		L("at", F(at.X), F(at.Y), F(0)),
		eff,
		L("uuid", Q(s.uuid()))))
}

// peGrid — шапка и вертикальные графы до основной надписи.
func (s *Sheet) peGrid(peBottom float64) {
	top, head := peTop, peTop+peHead
	s.peLine(Pt{peX0, head}, Pt{peX4, head})
	for _, x := range []float64{peX1, peX2, peX3} {
		s.peLine(Pt{x, top}, Pt{x, peBottom})
	}
	for y := head + peRow; y < peBottom-0.1; y += peRow {
		s.peLine(Pt{peX0, y}, Pt{peX4, y})
	}
	s.peText("Поз.", Pt{(peX0 + peX1) / 2, top + 4.5}, "center", false)
	s.peText("обозна-", Pt{(peX0 + peX1) / 2, top + 8}, "center", false)
	s.peText("чение", Pt{(peX0 + peX1) / 2, top + 11.5}, "center", false)
	s.peText("Наименование", Pt{(peX1 + peX2) / 2, top + peHead/2}, "center", false)
	s.peText("Кол.", Pt{(peX2 + peX3) / 2, top + peHead/2}, "center", false)
	s.peText("Примечание", Pt{(peX3 + peX4) / 2, top + peHead/2}, "center", false)
}

// wrapRefs — перечень обозначений по строкам не длиннее n знаков, перенос после запятой.
func wrapRefs(refs string, n int) []string {
	parts := strings.Split(refs, ", ")
	var out []string
	cur := ""
	for i, p := range parts {
		t := p
		if i < len(parts)-1 {
			t += ","
		}
		switch {
		case cur == "":
			cur = t
		case len([]rune(cur))+1+len([]rune(t)) <= n:
			cur += " " + t
		default:
			out = append(out, cur)
			cur = t
		}
	}
	return append(out, cur)
}
