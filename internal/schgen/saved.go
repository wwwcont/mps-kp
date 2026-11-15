package schgen

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
)

// SavedSheet reads the symbols and pin locations from a published single-sheet E3.
type SavedSheet struct {
	root    *Node
	symbols []*Comp
}

func LoadSavedSheet(path string) (*SavedSheet, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	root, err := Parse(string(data))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	lib := root.Find("lib_symbols")
	if root.Head() != "kicad_sch" || lib == nil {
		return nil, fmt.Errorf("%s: нет библиотеки символов Э3", path)
	}
	pins := map[string][]Pin{}
	for _, sym := range lib.All("symbol") {
		local := sym.Clone()
		_, name, ok := strings.Cut(sym.Arg(0), ":")
		if ok {
			local.Kids[1] = Q(name)
		}
		pins[sym.Arg(0)] = Pins(local)
	}
	s := &SavedSheet{root: root}
	for _, node := range root.All("symbol") {
		at := node.Find("at")
		id := node.Find("lib_id").Arg(0)
		c := &Comp{At: Pt{at.Num(0), at.Num(1)}, Rot: int(at.Num(2)), Unit: int(node.Find("unit").Num(0)), node: node, pins: map[string]Pt{}}
		for _, prop := range node.All("property") {
			switch prop.Arg(0) {
			case "Reference":
				c.Ref = prop.Arg(1)
			case "Value":
				c.Value = prop.Arg(1)
			}
		}
		for _, pin := range pins[id] {
			if pin.Unit == 0 || pin.Unit == c.Unit {
				pt := xform(c.At, c.Rot, pin.X, pin.Y)
				c.pins[pin.Num] = Pt{round(pt.X), round(pt.Y)}
			}
		}
		s.symbols = append(s.symbols, c)
	}
	return s, nil
}

// Fragments uses the components and wires from the archived sheet, including releases before KM-1 exported figures.
func (s *SavedSheet) Fragments(refs map[string][2]string, regions map[string]Rect) map[string]string {
	groups := map[string][]*Comp{}
	for group, prefixes := range drawingRoles {
		for _, prefix := range prefixes {
			for role, ref := range refs {
				if !drawingRole(role, prefix) {
					continue
				}
				unit := map[string]int{"norY2": 1, "norInd": 2, "norAL": 3, "norAH": 4, "orBuf": 1}[role]
				if role == "or" && refs["orBuf"][0] == ref[0] {
					unit = 2
				}
				for _, c := range s.symbols {
					if c.Ref == ref[0] && (unit == 0 || unit == c.Unit) {
						groups[group] = append(groups[group], c)
					}
				}
			}
		}
	}
	if refs["filter0"][0] == "" {
		r := regions["power"]
		for _, c := range s.symbols {
			if strings.HasPrefix(c.Ref, "C") && c.At.X >= r.X0 && c.At.X <= r.X1 && c.At.Y >= r.Y0 && c.At.Y <= r.Y1 {
				groups["power"] = append(groups["power"], c)
			}
		}
	}
	return drawingFragments(s.root, groups, s.symbols, func(p Pt) Pt { return p })
}

// CheckBuffer verifies that the archived E3 supports the buffer described in the reports and used by the emulator.
func (s *SavedSheet) CheckBuffer(refs map[string][2]string, addressLines int) error {
	labels := map[Pt][]string{}
	for _, label := range s.root.All("label") {
		at := label.Find("at")
		p := Pt{at.Num(0), at.Num(1)}
		labels[p] = append(labels[p], label.Arg(0))
	}
	wires := map[Pt][]Pt{}
	for _, wire := range s.root.All("wire") {
		xy := wire.Find("pts").All("xy")
		a, b := Pt{xy[0].Num(0), xy[0].Num(1)}, Pt{xy[1].Num(0), xy[1].Num(1)}
		wires[a] = append(wires[a], b)
		wires[b] = append(wires[b], a)
		for p := range labels {
			if onSegInner(a, b, p) {
				wires[a] = append(wires[a], p)
			}
		}
	}
	for _, c := range s.symbols {
		if strings.HasPrefix(c.Ref, "#") {
			for _, p := range c.pins {
				labels[p] = append(labels[p], c.Value)
			}
		}
	}
	check := func(role, pin, want string) error {
		ref := refs[role][0]
		for _, c := range s.symbols {
			start, ok := c.pins[pin]
			if c.Ref != ref || !ok {
				continue
			}
			queue, visited := []Pt{start}, map[Pt]bool{start: true}
			for len(queue) > 0 {
				at := queue[0]
				queue = queue[1:]
				if slices.Contains(labels[at], want) {
					return nil
				}
				for _, next := range wires[at] {
					if !visited[next] {
						visited[next] = true
						queue = append(queue, next)
					}
				}
			}
		}
		return fmt.Errorf("схема несовместима с описанием и моделью буфера: %s (%s), вывод %s, ожидается цепь %s", ref, role, pin, want)
	}
	for _, conn := range [][3]string{
		{"mcu", "13", nX2stb}, {"xsX2", "1", nX2stb}, {"mcu", "16", nWR}, {"mcu", "17", nRD},
		{"idt", "20", nWR}, {"idt", "22", "~{CS_buf}"}, {"idt", "59", nRDbuf}, {"idt", "62", nRD},
		{"orBuf", "1", "~{CS_buf}"}, {"orBuf", "2", nRD}, {"orBuf", "3", nRDbuf},
		{"idt", "40", "+5V"}, {"idt", "61", "+5V"}, {"idt", "19", "+5V"},
	} {
		if err := check(conn[0], conn[1], conn[2]); err != nil {
			return err
		}
	}
	for i := 0; i < 13; i++ {
		want := "GND"
		if i < addressLines {
			want = fmt.Sprintf("A%d", i)
		}
		for _, pin := range []int{44 + i, 37 - i} {
			if err := check("idt", strconv.Itoa(pin), want); err != nil {
				return err
			}
		}
	}
	for i, pin := range []string{"63", "64", "1", "2", "3", "4", "6", "7"} {
		if err := check("idt", pin, fmt.Sprintf("AD%d", i)); err != nil {
			return err
		}
	}
	for i, pin := range []string{"10", "11", "12", "14", "15", "16", "17", "18"} {
		name := fmt.Sprintf("X2_%d", i)
		if err := check("idt", pin, name); err != nil {
			return err
		}
		if err := check("xsX2", strconv.Itoa(i+2), name); err != nil {
			return err
		}
	}
	return nil
}
