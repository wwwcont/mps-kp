package schgen

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Renumber присваивает позиционные обозначения по ГОСТ 2.710: в пределах буквенного
// кода — по расположению на листе сверху вниз, столбцами слева направо.
// Столбец — элементы, перекрывающиеся по горизонтали (по корпусам, без ножек выводов).
// Части одного корпуса (DD3.1, DD3.2) получают номер по первой встреченной части.
func (s *Sheet) Renumber() {
	s.numCols = map[string][][]colItem{}
	groups := map[string][]*Comp{}
	for _, c := range s.syms {
		if strings.HasPrefix(c.Ref, "#") {
			continue
		}
		p := refPrefix(c.Ref)
		groups[p] = append(groups[p], c)
	}
	for prefix, cs := range groups {
		type box struct {
			c         *Comp
			x0, x1, y float64
		}
		var bs []box
		for _, c := range cs {
			x0, x1, ok := s.bodyX(c)
			if !ok {
				x0, x1 = c.At.X, c.At.X
				for _, p := range c.pins {
					x0, x1 = min(x0, p.X), max(x1, p.X)
				}
				if x1-x0 > 6 {
					x0, x1 = x0+2.54, x1-2.54
				}
			}
			bs = append(bs, box{c, x0, x1, c.At.Y})
		}
		// столбец — элементы с одним центром по горизонтали (±2,54 мм); столбцы — слева направо по центру, в столбце — сверху вниз.
		// Так читает руководитель: «у вас принято „сверху вниз и слева направо“, тогда DD2 — это должно быть DD1» (Михалин 08.10.2026:
		// регистр столбцов левее МК — отдельный, более левый столбец, хотя корпуса перекрываются по горизонтали).
		// Михалин 28.10.2026: «нумерация элементов — см. DD8.1 и DD9, „корпус“ которой левей; R10 правее R11»: столбец — по левому
		// краю корпуса (рисунок символа, без выводов), левее корпус — раньше номер; в столбце (левый край тот же) — сверху вниз
		cx := func(b box) float64 { return b.x0 }
		sort.SliceStable(bs, func(i, j int) bool { return cx(bs[i]) < cx(bs[j]) })
		var cols [][]box
		for _, b := range bs {
			if len(cols) == 0 || cx(b)-cx(cols[len(cols)-1][0]) > 1.905 { // C у кварца и сброса (вертикальный и горизонтальный) — один столбец
				cols = append(cols, nil)
			}
			cols[len(cols)-1] = append(cols[len(cols)-1], b)
		}
		s.numCols[prefix] = nil
		for _, col := range cols {
			var refs []colItem
			for _, b := range col {
				refs = append(refs, colItem{b.c, b.x0, b.x1, b.y})
			}
			s.numCols[prefix] = append(s.numCols[prefix], refs)
		}
		newRef := map[string]string{}
		n := 0
		for _, col := range cols {
			sort.SliceStable(col, func(i, j int) bool { return col[i].y < col[j].y })
			for _, b := range col {
				if _, ok := newRef[b.c.Ref]; !ok {
					n++
					newRef[b.c.Ref] = fmt.Sprintf("%s%d", prefix, n)
				}
			}
		}
		for _, c := range cs {
			c.setRef(newRef[c.Ref])
		}
	}
}

type colItem struct {
	c         *Comp
	x0, x1, y float64
}

// NumberingDoubts — места, где порядок «по столбцам» неочевиден глазу: соседние столбцы почти касаются (зазор < 2,54 мм),
// или в столбце, слитом по цепочке перекрытий, нижний элемент правее и не перекрывается с верхним, а верхний левее.
// Такие места правят раскладкой (элементы разносят), а не правилом.
func (s *Sheet) NumberingDoubts() []string {
	var out []string
	cx := func(b colItem) float64 { return b.x0 } // как в Renumber: по левому краю корпуса
	// порядок при «широких» столбцах: центры ближе 7,62 мм — один столбец (так может прочитать глаз)
	order := func(items []colItem, tol float64) []string {
		sort.SliceStable(items, func(i, j int) bool { return cx(items[i]) < cx(items[j]) })
		var cols [][]colItem
		for _, it := range items {
			if len(cols) == 0 || cx(it)-cx(cols[len(cols)-1][len(cols[len(cols)-1])-1]) > tol {
				cols = append(cols, nil)
			}
			cols[len(cols)-1] = append(cols[len(cols)-1], it)
		}
		var refs []string
		seen := map[string]bool{}
		for _, col := range cols {
			sort.SliceStable(col, func(i, j int) bool { return col[i].y < col[j].y })
			for _, it := range col {
				if !seen[it.c.Ref] {
					seen[it.c.Ref] = true
					refs = append(refs, it.c.Ref)
				}
			}
		}
		return refs
	}
	for prefix, cols := range s.numCols {
		var all []colItem
		for _, c := range cols {
			all = append(all, c...)
		}
		a := order(append([]colItem(nil), all...), 1.905)
		b := order(append([]colItem(nil), all...), 2.4)
		for i := range a {
			if a[i] != b[i] {
				out = append(out, fmt.Sprintf("%s: номера зависят от того, как читать столбцы: %v или %v", prefix, a, b))
				break
			}
		}
	}
	sort.Strings(out)
	return dedup(out)
}

func (c *Comp) setRef(ref string) {
	c.Ref = ref
	c.node.Walk(func(n *Node) {
		switch {
		case n.Head() == "property" && n.Arg(0) == "Reference":
			n.Kids[2] = Q(ref)
		case n.Head() == "reference":
			n.Kids[1] = Q(ref)
		}
	})
}

func refPrefix(ref string) string { return strings.TrimRight(ref, "0123456789") }

func refNum(ref string) int {
	n, _ := strconv.Atoi(ref[len(refPrefix(ref)):])
	return n
}

// bodyX — левый и правый край рисунка символа (прямоугольники, линии, окружности; без выводов и надписей) на листе.
func (s *Sheet) bodyX(c *Comp) (x0, x1 float64, ok bool) {
	if s.lib == nil {
		return 0, 0, false
	}
	def, found := s.lib.Syms[c.Sym]
	if !found {
		return 0, 0, false
	}
	x0, x1 = 1e9, -1e9
	add := func(x, y float64) {
		p := xform(c.At, c.Rot, x, y)
		x0, x1 = min(x0, p.X), max(x1, p.X)
		ok = true
	}
	for _, sub := range def.All("symbol") {
		parts := strings.Split(sub.Arg(0), "_")
		if len(parts) < 2 {
			continue
		}
		u, _ := strconv.Atoi(parts[len(parts)-2])
		st, _ := strconv.Atoi(parts[len(parts)-1])
		if (u != 0 && u != c.Unit && !(c.Unit == 0 && u == 1)) || st > 1 {
			continue
		}
		for _, g := range sub.Kids {
			switch g.Head() {
			case "rectangle":
				a, e := g.Find("start"), g.Find("end")
				add(a.Num(0), a.Num(1))
				add(e.Num(0), e.Num(1))
			case "polyline":
				for _, xy := range g.Find("pts").All("xy") {
					add(xy.Num(0), xy.Num(1))
				}
			case "circle":
				ce, r := g.Find("center"), g.Find("radius").Num(0)
				add(ce.Num(0)-r, ce.Num(1))
				add(ce.Num(0)+r, ce.Num(1))
			}
		}
	}
	return x0, x1, ok
}
