package schgen

import (
	"math"
	"strings"
)

// Rect — область листа, мм (координаты KiCad: x вправо, y вниз от левого верхнего угла листа).
type Rect struct{ X0, Y0, X1, Y1 float64 }

// Regions — рамки узлов схемы для рисунков ПЗ1 (вырезки из листа Э3). Считать до вывода листа
// (String сдвигает чертёж на J.ShiftX/ShiftY и обнуляет сдвиг — здесь он учитывается).
func (s *Sheet) Regions() map[string]Rect {
	groups := map[string][]string{
		"mcu":   {"mcu", "latchA", "zq", "cX1", "cX2", "rRst", "cRst"},
		"dec":   {"dec"},
		"kb":    {"kb173", "buf", "and", "or", "vd", "sb", "pull", "ser", "ck"},
		"ind":   {"latchInd", "norInd", "rseg", "hg"},
		"idt":   {"idt", "xsX2", "regAL", "regAH", "norAL", "norAH"},
		"y2":    {"latchY2", "norY2", "xsY"},
		"power": {"xsPwr", "filter"},
	}
	out := map[string]Rect{}
	for g, prefixes := range groups {
		r := Rect{math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)}
		add := func(c *Comp) {
			pts := []Pt{c.At}
			for _, p := range c.pins {
				pts = append(pts, p)
			}
			for _, p := range pts {
				r.X0, r.Y0 = math.Min(r.X0, p.X), math.Min(r.Y0, p.Y)
				r.X1, r.Y1 = math.Max(r.X1, p.X), math.Max(r.Y1, p.Y)
			}
		}
		for role, c := range s.Roles {
			for _, pre := range prefixes {
				if role == pre || strings.HasPrefix(role, pre) && len(role) > len(pre) && (role[len(pre)] >= '0' && role[len(pre)] <= '9') {
					add(c)
				}
			}
		}
		if g == "power" {
			for _, c := range s.syms {
				if strings.HasPrefix(c.Sym, "C") && (c.Value == "470 мк" || strings.Contains(c.Value, " н") && c.At.Y > 230) {
					add(c)
				}
			}
		}
		if math.IsInf(r.X0, 1) {
			continue
		}
		// поле вокруг выводов (надписи, метки) и сдвиг «почерка»
		const pad = 8
		out[g] = Rect{r.X0 - pad + s.J.ShiftX, r.Y0 - pad + s.J.ShiftY, r.X1 + pad + s.J.ShiftX, r.Y1 + pad + s.J.ShiftY}
	}
	return out
}
