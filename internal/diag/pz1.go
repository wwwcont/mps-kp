package diag

import (
	"fmt"
	"image"
)

// Structural показывает внешние узлы и их связи с микроконтроллером.
func Structural(fontTTF []byte, fontMM, stroke float64) (image.Image, error) {
	c, err := New(170, 68, fontTTF, fontMM, stroke)
	if err != nil {
		return nil, err
	}
	c.Box(5, 23, 38, 24, "Микро-\nконтроллер")
	c.Box(72, 3, 42, 15, "Кольцевой буфер")
	c.Box(72, 27, 42, 17, "Пульт управления")
	c.Box(72, 53, 42, 13, "Регистр Y2")
	c.Line(false, 43, 27, 55, 27, 55, 10, 72, 10)
	c.Line(false, 43, 35, 72, 35)
	c.Line(false, 43, 43, 55, 43, 55, 59, 72, 59)
	c.Text("Адрес, данные", 68, 7, 1, 0)
	c.Line(true, 148, 10, 114, 10)
	c.Text("X2", 147, 7, 1, 0)
	c.Line(true, 10, 20, 10, 23)
	c.Text("X2stb", 12, 18, 0, 0)
	c.Line(true, 148, 33, 114, 33)
	c.Text("X1", 147, 30, 1, 0)
	c.Line(true, 114, 59, 148, 59)
	c.Text("Y2", 147, 56, 1, 0)
	c.Line(true, 32, 47, 32, 64)
	c.Text("Y1stb, Y2stb", 35, 63, 0, 0)
	return c.Image(), nil
}

// FuncInfo — обозначения из сохранённой Э3 и параметры варианта.
type FuncInfo struct {
	MCU, Latch, Dec, Buf, KbCol, KbRow, IndReg, Ind, Y2Reg string
	CSBuf, CSKb, CSInd, CSY2                               string
	X2, Out, Keyboard                                      string
	BufAddrLast, Cols, Rows                                int
}

// Functional показывает функциональные связи. Одноимённые управляющие сигналы помечены у обоих концов.
func Functional(fi FuncInfo, fontTTF []byte, stroke float64) (image.Image, error) {
	c, err := New(252, 154, fontTTF, 3.8, stroke)
	if err != nil {
		return nil, err
	}
	bundle := func(x, y float64, count int) {
		c.Line(false, x-1.5, y+1.5, x+1.5, y-1.5)
		c.Text(fmt.Sprint(count), x+2, y-3.5, 0, 3)
	}
	label := func(s string, x, y, align float64) { c.Text(s, x, y, align, 3.1) }
	c.Box(25, 31, 36, 108, "Микро-\nконтроллер\n"+fi.MCU)
	c.Box(101, 9, 39, 18, "Защёлка адреса\n"+fi.Latch)
	c.Box(167, 21, 43, 32, "Двухпортовое ОЗУ\n"+fi.Buf)
	c.Box(101, 73, 39, 22, "Регистр столбцов\n"+fi.KbCol)
	c.Box(101, 102, 39, 20, "Буфер строк\n"+fi.KbRow)
	c.Box(167, 81, 43, 32, "Клавиатура\n"+fi.Keyboard)
	c.Box(167, 58, 43, 17, "Регистр индикатора\n"+fi.IndReg)
	c.Box(220, 58, 24, 17, "Индикатор\n"+fi.Ind)
	c.Box(167, 129, 43, 20, "Регистр Y2\n"+fi.Y2Reg)
	if fi.Dec != "" {
		c.Box(101, 47, 39, 19, "Дешифратор\n"+fi.Dec)
		c.Line(false, 61, 51, 101, 51)
		label("A13–A15", 64, 47, 0)
		bundle(91, 51, 3)
		c.Line(false, 61, 60, 101, 60)
		label("CS_EN", 64, 57, 0)
		for i, labelText := range []string{fi.CSBuf + " / CS_buf", fi.CSKb + " / CS_kbd", fi.CSInd + " / CS_ind", fi.CSY2 + " / CS_rgo"} {
			y := 50.0 + float64(i)*5
			c.Line(false, 140, y, 164, y)
			label(labelText, 142, y-2.5, 0)
		}
	}

	c.Line(false, 61, 36, 72, 36, 72, 14, 101, 14)
	label("AD0–AD7", 75, 10, 0)
	bundle(94, 14, 8)
	c.Line(false, 61, 42, 77, 42, 77, 22, 101, 22)
	label("ALE", 82, 19, 0)
	c.Line(false, 140, 14, 178, 14, 178, 21)
	label("A0–A7", 143, 10, 0)
	bundle(166, 14, 8)
	c.Line(false, 61, 33, 66, 33, 66, 5, 196, 5, 196, 21)
	label(fmt.Sprintf("A8–A%d", fi.BufAddrLast), 174, 2.5, 1)
	bundle(190, 5, fi.BufAddrLast-7)
	c.Line(false, 72, 36, 72, 139, 167, 139)
	for _, line := range [][2]float64{{36, 167}, {69, 167}, {84, 101}, {110, 101}} {
		c.Line(false, 72, line[0], line[1], line[0])
		c.dc.DrawCircle(72*c.s, line[0]*c.s, 0.65*c.s)
		c.dc.Fill()
	}
	c.Line(false, 210, 67, 220, 67)
	c.Line(false, 140, 87, 167, 87)
	c.Line(false, 140, 107, 167, 107)
	label("Col1…", 143, 83, 0)
	label("Row1…", 143, 103, 0)
	bundle(161, 87, fi.Cols)
	bundle(161, 107, fi.Rows)

	for _, w := range []struct {
		x1, x2, y float64
		name      string
	}{
		{210, 247, 30, "X2_0–X2_7 (" + fi.X2 + ")"},
		{2, 25, 94, "X2stb (" + fi.X2 + ")"},
		{4, 25, 84, "INT0"},
		{61, 96, 122, "RD"},
		{61, 96, 130, "WR"},
		{210, 246, 140, "Y2_0–Y2_7 (" + fi.Out + ")"},
		{4, 25, 112, "Y1stb"},
		{4, 25, 122, "Y2stb"},
		{210, 234, 43, "WR, RD, CS_buf"},
		{210, 240, 97, "INT0"},
		{210, 234, 132, "WR, CS_rgo"},
		{167, 148, 74, "WR, CS_ind"},
		{101, 78, 92, "WR, CS_kbd"},
		{101, 78, 119, "RD, CS_kbd"},
	} {
		c.Line(false, w.x1, w.y, w.x2, w.y)
		label(w.name, (w.x1+w.x2)/2, w.y-3.5, 0.5)
	}
	bundle(240, 30, 8)
	bundle(240, 140, 8)
	return c.Image(), nil
}

// Timing — временные диаграммы цикла movx при 12 МГц: а) чтение, б) запись. Параметры — из даташита AT89S53 (нс).
type Timing struct {
	TAVLL, TLLAX, TLLWL, TRLRH, TRLDV, TRHDX, TQVWH, TWHQX, TWLWH float64
	MemRead, MemWrite                                             string // подписи сигналов памяти в скобках, напр. «(OEL)»
}

func (t Timing) draw(c *Canvas, y0 float64, write bool) {
	// шкала времени: 1 нс = 0.12 мм
	k := 0.12
	x0 := 34.0
	ale := []float64{0, 120}                                    // ALE высокий до 120 нс
	strobe := []float64{120 + t.TLLWL, 120 + t.TLLWL + t.TRLRH} // RD/WR низкий
	end := strobe[1] + 100
	X := func(ns float64) float64 { return x0 + ns*k }
	rows := []string{"ALE", "P2", "P0", "RD"}
	if write {
		rows[3] = "WR"
		rows = append(rows, "X2_0–X2_7")
	}
	h := 7.0
	for i, r := range rows {
		y := y0 + float64(i)*14
		if r == "RD" || r == "WR" {
			c.Overbar(r, 2, y+h/2, 0)
			mem := map[bool]string{true: t.MemWrite, false: t.MemRead}[write]
			if mem != "" {
				c.Text("(", 3+c.Width(r), y+h/2, 0, 0)
				c.Overbar(mem, 3+c.Width(r)+c.Width("("), y+h/2, 0)
				c.Text(")", 3+c.Width(r)+c.Width("(")+c.Width(mem), y+h/2, 0, 0)
			}
		} else {
			c.Text(r, 2, y+h/2, 0, 0)
		}
		switch r {
		case "ALE":
			c.Line(false, X(-40), y+h, X(ale[0]), y+h, X(ale[0])+1, y, X(ale[1]), y, X(ale[1])+1, y+h, X(end), y+h)
		case "P2":
			bus(c, X(-40), X(-40)+1, X(end)-1, X(end), y, h, "A8–A15")
		case "P0":
			bus(c, X(-40), X(-40)+1, X(ale[1]+t.TLLAX), X(ale[1]+t.TLLAX)+1, y, h, "A0–A7")
			if write {
				dv := strobe[1] - t.TQVWH
				bus(c, X(dv), X(dv)+1, X(strobe[1]+t.TWHQX), X(strobe[1]+t.TWHQX)+1, y, h, "данные МК")
			} else {
				dv := strobe[0] + t.TRLDV
				bus(c, X(dv), X(dv)+1, X(strobe[1]+t.TRHDX+40), X(strobe[1]+t.TRHDX+40)+1, y, h, "данные ОЗУ")
			}
		case "X2_0–X2_7":
			bus(c, X(strobe[0]-35), X(strobe[0]-35)+1, X(end)-1, X(end), y, h, "данные с XS2")
		default:
			c.Line(false, X(-40), y, X(strobe[0]), y, X(strobe[0])+1, y+h, X(strobe[1]), y+h, X(strobe[1])+1, y, X(end), y)
		}
	}
	// размеры
	yd := y0 + float64(len(rows))*14 + 3
	c.Dim(X(ale[1])-X(0)+X(0), X(ale[1]+t.TLLAX), y0+28-3, fmt.Sprintf("tLLAX ≥ %.0f", t.TLLAX))
	c.Dim(X(ale[1]), X(strobe[0]), yd, fmt.Sprintf("tLLWL %.0f…%.0f", t.TLLWL, t.TLLWL+100))
	if write {
		c.Dim(X(strobe[0]), X(strobe[1]), yd+6, fmt.Sprintf("tWLWH ≥ %.0f", t.TWLWH))
		c.Dim(X(strobe[1]-t.TQVWH), X(strobe[1]), y0+28+h+2.5, fmt.Sprintf("tQVWH ≥ %.0f", t.TQVWH))
		c.Dim(X(strobe[1]), X(strobe[1]+t.TWHQX), yd, fmt.Sprintf("tWHQX ≥ %.0f", t.TWHQX))
	} else {
		c.Dim(X(strobe[0]), X(strobe[1]), yd+6, fmt.Sprintf("tRLRH ≥ %.0f", t.TRLRH))
		c.Dim(X(strobe[0]), X(strobe[0]+t.TRLDV), y0+28+h+2.5, fmt.Sprintf("tRLDV ≤ %.0f", t.TRLDV))
	}
	c.Text("время в нс, 12 МГц", X(end), yd+12, 1, c.fontMM*0.75)
}

// bus — шина: «ромбики» перехода в xa…xb и xc…xd, текст внутри.
func bus(c *Canvas, xa, xb, xc, xd, y, h float64, label string) {
	c.Line(false, xa, y+h/2, xb, y, xc, y, xd, y+h/2, xc, y+h, xb, y+h, xa, y+h/2)
	c.Text(label, (xb+xc)/2, y+h/2, 0.5, c.fontMM*0.8)
}

// TimingDiagrams — два рисунка: чтение и запись.
func TimingDiagrams(t Timing, fontTTF []byte, fontMM, stroke float64) (read, write image.Image, err error) {
	for i, w := range []bool{false, true} {
		c, e := New(165, 84+float64(i)*14, fontTTF, fontMM, stroke)
		if e != nil {
			return nil, nil, e
		}
		t.draw(c, 4, w)
		if i == 0 {
			read = c.Image()
		} else {
			write = c.Image()
		}
	}
	return read, write, nil
}
