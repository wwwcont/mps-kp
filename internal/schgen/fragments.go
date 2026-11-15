package schgen

import "strings"

// Drawings возвращает полный лист и рисунки узлов из тех же символов и проводников.
func (s *Sheet) Drawings() (full string, fragments map[string]string) {
	dx, dy := s.J.ShiftX, s.J.ShiftY
	root := s.drawing()
	shifted := func(p Pt) Pt { return Pt{round(p.X + dx), round(p.Y + dy)} }
	return root.String(), drawingFragments(root, s.drawingGroups(), s.syms, shifted)
}

func drawingFragments(root *Node, groups map[string][]*Comp, symbols []*Comp, pinPoint func(Pt) Pt) map[string]string {
	fragments := make(map[string]string, len(groups))
	point := func(n *Node) Pt { return Pt{n.Num(0), n.Num(1)} }
	type wire struct {
		node *Node
		a, b Pt
	}
	var wires []wire
	edges := map[Pt][]int{}
	for _, node := range root.All("wire") {
		pts := node.Find("pts").All("xy")
		a, b := point(pts[0]), point(pts[1])
		edges[a] = append(edges[a], len(wires))
		edges[b] = append(edges[b], len(wires))
		wires = append(wires, wire{node, a, b})
	}

	for name, comps := range groups {
		keep := map[*Node]bool{}
		reached := map[Pt]bool{}
		var pending []Pt
		for _, c := range comps {
			keep[c.node] = true
			for _, p := range c.pins {
				pending = append(pending, pinPoint(p))
			}
		}
		// Идём по настоящим проводам от выводов узла. Переходы через шины и
		// одноимённые метки не нужны: метка уже подписывает конец проводника.
		for len(pending) > 0 {
			p := pending[len(pending)-1]
			pending = pending[:len(pending)-1]
			if reached[p] {
				continue
			}
			reached[p] = true
			for _, i := range edges[p] {
				w := wires[i]
				keep[w.node] = true
				pending = append(pending, w.a, w.b)
			}
		}
		for _, c := range symbols {
			if strings.HasPrefix(c.Ref, "#") {
				for _, p := range c.pins {
					if reached[pinPoint(p)] {
						keep[c.node] = true
					}
				}
			}
		}
		fragment := L("kicad_sch")
		for _, node := range root.Kids[1:] {
			switch node.Head() {
			case "wire", "symbol":
				if !keep[node] {
					continue
				}
			case "junction":
				if !reached[point(node.Find("at"))] {
					continue
				}
			case "label":
				p := point(node.Find("at"))
				onWire := reached[p]
				for _, w := range wires {
					if keep[w.node] && onSegInner(w.a, w.b, p) {
						onWire = true
						break
					}
				}
				if !onWire {
					continue
				}
			case "bus", "bus_entry", "text":
				continue
			}
			fragment.Kids = append(fragment.Kids, node)
		}
		fragments[name] = fragment.String()
	}
	return fragments
}
