package schgen

import (
	"math"
	"sort"
	"strconv"
	"strings"
)

// busMarks — обозначение подключений к шинам по ГОСТ 2.702 (замечания Михалина 06.10 и Гольцова 08.10.2026):
// у каждого входа в шину — условный номер линии арабской цифрой (один и тот же номер у одного сигнала во всех местах листа),
// а имя сигнала (AD0, A8, ~{WR}) — «дополнительная информация для читаемости» — на середине проводника, а не у входа.
// Работает по готовым отводам (после перевода их в Т-образные): метка цепи на проводе отвода переносится на середину
// самого длинного участка провода, у точки на шине ставится номер.
func (s *Sheet) busMarks() {
	type lab struct {
		node *Node
		name string
		at   Pt
	}
	var labs []*lab
	for _, it := range s.items {
		if it.Head() == "label" {
			a := it.Find("at")
			labs = append(labs, &lab{it, it.Arg(0), Pt{a.Num(0), a.Num(1)}})
		}
	}
	onSeg := func(a, b, p Pt) bool { return p.eq(a) || p.eq(b) || onSegInner(a, b, p) }
	// цепочка проводов от конца отвода (до двух звеньев: отвод → [излом] → провод к выводу)
	chain := func(w Pt) [][2]Pt {
		var out [][2]Pt
		cur := w
		for hop := 0; hop < 3; hop++ {
			var next *[2]Pt
			for i := range s.wires {
				ww := s.wires[i]
				if (ww[0].eq(cur) || ww[1].eq(cur)) && !containsSeg(out, ww) {
					next = &s.wires[i]
					break
				}
			}
			if next == nil {
				break
			}
			out = append(out, *next)
			if next[0].eq(cur) {
				cur = next[1]
			} else {
				cur = next[0]
			}
		}
		return out
	}
	type mark struct {
		name   string
		on, w  Pt
		lab    *lab
		chainW [][2]Pt
	}
	var marks []mark
	names := map[string]bool{}
	for _, e := range s.entries {
		w, nb := e[0], e[1]
		if !s.onAnyBus(nb) {
			w, nb = nb, w
		}
		ch := chain(w)
		var l *lab
		for _, lb := range labs {
			for _, seg := range ch {
				if onSeg(seg[0], seg[1], lb.at) {
					l = lb
				}
			}
			if lb.at.eq(w) {
				l = lb
			}
		}
		if l == nil {
			continue
		}
		marks = append(marks, mark{l.name, nb, w, l, ch})
		names[l.name] = true
	}
	// номера — в каждой шине свои, с 1 (замечание Михалина 14.10.2026: «нумерация контактов в шинах не сквозная, в каждой шине
	// начинаем с 1»), по порядку, как элементы: столбцами слева направо, в столбце сверху вниз («нумерация проводников подчиняется
	// правилу нумерации элементов»). Номер линии — по первому её входу в этом порядке.
	_ = names
	group := s.busGroups()
	type key struct {
		g    int
		name string
	}
	num := map[key]int{}
	order := append([]mark(nil), marks...)
	sort.SliceStable(order, func(i, j int) bool {
		a, b := order[i].on, order[j].on
		if math.Abs(a.X-b.X) > 2.54 {
			return a.X < b.X
		}
		return a.Y < b.Y
	})
	next := map[int]int{}
	for _, m := range order {
		k := key{group(m.on), m.name}
		if _, ok := num[k]; !ok {
			next[k.g]++
			num[k] = next[k.g]
		}
	}
	moved := map[*lab]bool{}
	fs := s.labelFont()
	for _, m := range marks {
		// номер у точки входа, со стороны провода, над ним
		n := strconv.Itoa(num[key{group(m.on), m.name}])
		vertBus := math.Abs(m.w.X-m.on.X) > math.Abs(m.w.Y-m.on.Y)
		var at Pt
		just := "left"
		// над проводом; если вход сдвинут (излом) и провод к выводу идёт выше — под проводом
		up := true
		for _, seg := range m.chainW {
			for _, p := range seg {
				if p.Y < m.on.Y-0.01 && p.Y > m.on.Y-1.6 && math.Abs(p.X-m.on.X) < 3.3 { // провод над местом номера
					up = false
				}
			}
		}
		dy := -0.3
		vj := "bottom"
		if !up {
			dy, vj = 0.3, "top"
		}
		switch {
		case vertBus && m.w.X > m.on.X:
			at = Pt{m.on.X + 0.5, m.on.Y + dy}
		case vertBus:
			at, just = Pt{m.on.X - 0.5, m.on.Y + dy}, "right"
		default: // горизонтальная шина, провод сверху или снизу
			at = Pt{m.on.X + 0.4, m.on.Y + map[bool]float64{true: -0.6, false: 2.2}[m.w.Y < m.on.Y]} // под шиной — не касаясь её толстой линии
		}
		s.items = append(s.items, L("text", Q(n), L("exclude_from_sim", A("no")),
			L("at", F(round(at.X)), F(round(at.Y)), F(0)),
			L("effects", s.numFontExpr(), L("justify", A(just), A(vj))),
			L("uuid", Q(s.uuid()))))
		// имя — на середину самого длинного участка цепочки
		if moved[m.lab] {
			continue
		}
		moved[m.lab] = true
		best, bl := [2]Pt{}, 0.0
		for _, seg := range m.chainW {
			if l := dist(seg[0], seg[1]); l > bl {
				best, bl = seg, l
			}
		}
		tw := float64(len([]rune(strings.NewReplacer("~{", "", "}", "").Replace(m.name)))) * fs * charW
		if bl < tw+3 { // до середины места нет: имя — сразу за номером линии, а не поверх него
			nw := float64(len(n))*fs*charW + 0.8
			a := m.lab.node.Find("at")
			if vertBus && math.Abs(m.w.Y-m.on.Y) < 0.01 && math.Abs(a.Num(1)-m.on.Y) < 0.01 {
				x := snap(m.on.X + 0.5 + nw)
				if m.w.X < m.on.X {
					x = snap(m.on.X - 0.5 - nw - tw)
				}
				onWire := false // точка метки обязана лежать на проводе, иначе метка «висит» (ERC)
				for _, seg := range m.chainW {
					if onSeg(seg[0], seg[1], Pt{x, m.on.Y}) {
						onWire = true
					}
				}
				if onWire {
					a.Kids[1], a.Kids[3] = F(x), F(0)
					setJustify(m.lab.node, "left")
				}
			}
			continue
		}
		// «где-то посередине проводника»: ближе к шине, за номером (у вывода тесно — номера выводов, обозначения элементов)
		near, far := best[0], best[1]
		if dist(best[1], m.w) < dist(best[0], m.w) {
			near, far = best[1], best[0]
		}
		if s.LabelAtPin[m.name] {
			near, far = far, near
		}
		dx, dy := far.X-near.X, far.Y-near.Y
		k := math.Min(0.5, (3.81+tw/2)/bl)
		mid := Pt{near.X + dx*k, near.Y + dy*k}
		a := m.lab.node.Find("at")
		if math.Abs(best[0].Y-best[1].Y) < 0.01 { // горизонтальный участок: текст слева направо
			a.Kids[1], a.Kids[2], a.Kids[3] = F(snap(mid.X-tw/2)), F(round(mid.Y)), F(0)
			setJustify(m.lab.node, "left")
		} else { // вертикальный: снизу вверх
			a.Kids[1], a.Kids[2], a.Kids[3] = F(round(mid.X)), F(snap(mid.Y+tw/2)), F(90)
			setJustify(m.lab.node, "left")
		}
	}
}

// snap — на сетку 0,635: точка метки должна лежать на проводе (провода на сетке 1,27).
func snap(v float64) float64 { return math.Round(v/0.635) * 0.635 }

func setJustify(n *Node, just string) {
	if ef := n.Find("effects"); ef != nil {
		if j := ef.Find("justify"); j != nil {
			j.Kids = []*Node{A("justify"), A(just), A("bottom")}
		}
	}
}

func containsSeg(list [][2]Pt, w [2]Pt) bool {
	for _, x := range list {
		if x[0].eq(w[0]) && x[1].eq(w[1]) {
			return true
		}
	}
	return false
}

func (s *Sheet) onAnyBus(p Pt) bool {
	for _, b := range s.buses {
		if p.eq(b[0]) || p.eq(b[1]) || onSegInner(b[0], b[1], p) {
			return true
		}
	}
	return false
}

func (s *Sheet) numFontExpr() *Node {
	return L("font", L("size", F(s.labelFont()), F(s.labelFont())))
}

// busNumbers — условные номера линий: сначала данные/адрес (AD0…AD7, A0…A15), потом остальные группы по имени с индексом.
func busNumbers(names map[string]bool) map[string]int {
	var list []string
	for n := range names {
		list = append(list, n)
	}
	rank := func(n string) (int, string, int) {
		base := strings.NewReplacer("~{", "", "}", "").Replace(n)
		pre := strings.TrimRight(base, "0123456789")
		idx, _ := strconv.Atoi(base[len(pre):])
		order := map[string]int{"AD": 0, "A": 1, "X2_": 3, "X2_A": 3, "Y2_": 4, "Col": 5, "Row": 6, "Q": 7}
		if seg := strings.Index("abcdefg", base); len(base) == 1 && seg >= 0 { // сегменты — в порядке a…g, dp
			return 8, "", seg
		}
		if base == "dp" {
			return 8, "", 7
		}
		r, ok := order[pre]
		if !ok {
			r = 2 // управляющие: ALE, RD, WR, CS_…, INT…, RST — между адресом и данными X2/Y2
		}
		return r, pre, idx
	}
	sort.Slice(list, func(i, j int) bool {
		ri, pi, ii := rank(list[i])
		rj, pj, ij := rank(list[j])
		if ri != rj {
			return ri < rj
		}
		if pi != pj {
			return pi < pj
		}
		return ii < ij
	})
	out := map[string]int{}
	for i, n := range list {
		out[n] = i + 1
	}
	return out
}

// busGroups — номер связной шины (отрезки, касающиеся друг друга концами или концом середины) для точки на шине.
func (s *Sheet) busGroups() func(Pt) int {
	par := make([]int, len(s.buses))
	for i := range par {
		par[i] = i
	}
	var find func(int) int
	find = func(i int) int {
		for par[i] != i {
			par[i] = par[par[i]]
			i = par[i]
		}
		return i
	}
	touch := func(a, b [2]Pt) bool {
		for _, p := range a {
			if p.eq(b[0]) || p.eq(b[1]) || onSegInner(b[0], b[1], p) {
				return true
			}
		}
		for _, p := range b {
			if onSegInner(a[0], a[1], p) {
				return true
			}
		}
		return false
	}
	for i := range s.buses {
		for j := i + 1; j < len(s.buses); j++ {
			if touch(s.buses[i], s.buses[j]) {
				par[find(i)] = find(j)
			}
		}
	}
	return func(p Pt) int {
		for i, b := range s.buses {
			if p.eq(b[0]) || p.eq(b[1]) || onSegInner(b[0], b[1], p) {
				return find(i)
			}
		}
		return -1
	}
}
