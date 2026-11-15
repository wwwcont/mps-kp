package schgen

import (
	"strings"
)

// Rect — область листа, мм (координаты KiCad: x вправо, y вниз от левого верхнего угла листа).
type Rect struct{ X0, Y0, X1, Y1 float64 }

// DrawingGroups — состав узлов для рисунков ПЗ1.
var drawingRoles = map[string][]string{
	"mcu":   {"mcu", "latchA", "zq", "cX1", "cX2", "rRst", "cRst"},
	"dec":   {"dec"},
	"kb":    {"kb173", "buf", "and", "or", "vd", "sb", "pull", "ser", "ck"},
	"ind":   {"latchInd", "norInd", "rseg", "hg"},
	"idt":   {"idt", "xsX2", "rX2", "orBuf", "regAL", "regAH", "norAL", "norAH"},
	"y2":    {"latchY2", "norY2", "xsY"},
	"power": {"xsPwr", "filter"},
}

func drawingRole(role, prefix string) bool {
	return role == prefix || strings.HasPrefix(role, prefix) && len(role) > len(prefix) && role[len(prefix)] >= '0' && role[len(prefix)] <= '9'
}

func (s *Sheet) drawingGroups() map[string][]*Comp {
	out := make(map[string][]*Comp, len(drawingRoles))
	for group, prefixes := range drawingRoles {
		for role, c := range s.Roles {
			for _, prefix := range prefixes {
				if drawingRole(role, prefix) {
					out[group] = append(out[group], c)
				}
			}
		}
	}
	return out
}

// Regions — рамки узлов схемы на полном листе, мм. Считать до String/Drawings.
func (s *Sheet) Regions() map[string]Rect {
	out := map[string]Rect{}
	for group, comps := range s.drawingGroups() {
		var pts []Pt
		for _, c := range comps {
			pts = append(pts, c.At)
			for _, p := range c.pins {
				pts = append(pts, p)
			}
		}
		if len(pts) == 0 {
			continue
		}
		b := boxOf(pts...)
		const pad = 8
		out[group] = Rect{b.x0 - pad + s.J.ShiftX, b.y0 - pad + s.J.ShiftY, b.x1 + pad + s.J.ShiftX, b.y1 + pad + s.J.ShiftY}
	}
	return out
}
