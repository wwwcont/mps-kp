package variant

import "testing"

func load(t *testing.T) *Table {
	t.Helper()
	tb, err := LoadTable("../../data/table-2025.yaml")
	if err != nil {
		t.Fatal(err)
	}
	return tb
}

// Эталон — вариант, посчитанный руками в ПЗ1: А-12, M=14.
func TestA12M14(t *testing.T) {
	p, err := Compute(load(t), &Student{Group: "А-12", M: 14})
	if err != nil {
		t.Fatal(err)
	}
	check := func(name string, got, want any) {
		t.Helper()
		if got != want {
			t.Errorf("%s: got %v, want %v", name, got, want)
		}
	}
	check("k", p.K, 0)
	check("Y1", p.Y1Pin, "P1.0")
	check("Y2", p.Y2Pin, "P1.1")
	check("T1", p.T1ms, 352)
	check("V", p.V, 1540)
	check("T2", p.T2us, 1745)
	check("T3", p.T3ms, 2830)
	check("T2 reload", Hex(p.T2Reload), "0F92Fh")
	check("empty bit", p.Empty.Asm, "23h.2")
	check("ovf bit", p.Ovf.Asm, "21h.6")
	check("keyboard", p.Keyboard, "4x3")
	check("kb int", p.KbIntPin, "P3.2")
	check("ind", Hex(p.Dev("Индикатор").Base), "7800h")
	check("kb", Hex(p.Dev("Клавиатура").Base), "0B800h")
	check("buf", Hex(p.Dev("Буфер (IDT7005)").Base), "0D800h")
	check("y2", Hex(p.Dev("Регистр Y2").Base), "0E800h")
	check("unused", p.UnusedCS, "P2.3")
	check("fill", p.FillTimeS, 308.0)
	check("tick", p.Y1.TickMs, 2)
	check("y1 ticks", p.Y1.Ticks, 176)
	check("t3 ticks", p.T3.Ticks, 1415)
}

// Все группы и все M считаются без ошибок и без конфликтов CS.
func TestAllVariants(t *testing.T) {
	tb := load(t)
	for g := range tb.Groups {
		for m := 1; m <= 30; m++ {
			if _, err := Compute(tb, &Student{Group: g, M: m}); err != nil {
				t.Errorf("%s M=%d: %v", g, m, err)
			}
		}
	}
}

// ТЗ-2026: CS — выходы дешифратора 74HC138 (A13..A15), базовый адрес = n·2000h.
// Эталон посчитан руками по табл. 1 ТЗ-2026 (docs/research/tz-2026.md).
func TestA12M14_2026(t *testing.T) {
	tb, err := LoadTable("../../data/table-2026.yaml")
	if err != nil {
		t.Fatal(err)
	}
	p, err := Compute(tb, &Student{Group: "А-12", M: 14})
	if err != nil {
		t.Fatal(err)
	}
	check := func(name string, got, want any) {
		t.Helper()
		if got != want {
			t.Errorf("%s: got %v, want %v", name, got, want)
		}
	}
	check("k", p.K, 0)
	check("Y1", p.Y1Pin, "P1.0")
	check("Y2", p.Y2Pin, "P1.1")
	check("T1", p.T1ms, 331)
	check("V", p.V, 1744)
	check("T2", p.T2us, 2332)
	check("T2 reload", Hex(p.T2Reload), "0F6E4h")
	check("T3", p.T3ms, 3040)
	check("G", p.G, 12)
	check("empty bit", p.Empty.Asm, "21h.6")
	check("ovf bit", p.Ovf.Asm, "23h.2")
	check("head", p.Head, 0x47)
	check("keyboard", p.Keyboard, "4x3")
	check("kb int", p.KbIntPin, "P3.2")
	check("x2 int", p.X2IntPin, "P3.3")
	check("timers", [3]int{p.Y1Timer, p.Y2Timer, p.T3Timer}, [3]int{0, 1, 0})
	check("indicator", p.Indicator, "anode")
	check("cs mode", p.CSMode, "decoder")
	check("cs_en", p.CSEnPin, "P3.4")
	check("buf", p.Dev("Буфер (IDT7005)").CS+" "+Hex(p.Dev("Буфер (IDT7005)").Base), "Y5 0A000h")
	check("y2", p.Dev("Регистр Y2").CS+" "+Hex(p.Dev("Регистр Y2").Base), "Y6 0C000h")
	check("ind", p.Dev("Индикатор").CS+" "+Hex(p.Dev("Индикатор").Base), "Y7 0E000h")
	check("kb", p.Dev("Клавиатура").CS+" "+Hex(p.Dev("Клавиатура").Base), "Y1 2000h")
	check("unused", p.UnusedCS, "Y0, Y2, Y3, Y4")
	check("fill", p.FillTimeS, 436.0)
	check("prog3", p.M%3, 2)
}

// Нечётный M меняет CS регистра Y2 и индикатора местами (А-12: 6/7 и 7/6).
func TestParity2026(t *testing.T) {
	tb, err := LoadTable("../../data/table-2026.yaml")
	if err != nil {
		t.Fatal(err)
	}
	p, err := Compute(tb, &Student{Group: "А-12", M: 15})
	if err != nil {
		t.Fatal(err)
	}
	if p.Dev("Регистр Y2").CS != "Y7" || p.Dev("Индикатор").CS != "Y6" || p.Keyboard != "3x4" {
		t.Errorf("M=15: rgo %s, ind %s, kb %s", p.Dev("Регистр Y2").CS, p.Dev("Индикатор").CS, p.Keyboard)
	}
}

func TestAllVariants2026(t *testing.T) {
	tb, err := LoadTable("../../data/table-2026.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for g := range tb.Groups {
		for m := 1; m <= 30; m++ {
			if _, err := Compute(tb, &Student{Group: g, M: m}); err != nil {
				t.Errorf("%s M=%d: %v", g, m, err)
			}
		}
	}
}
