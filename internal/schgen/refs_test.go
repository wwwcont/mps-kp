package schgen

import (
	"sort"
	"testing"
)

// Позиционные обозначения: в каждой группе номера 1..n без пропусков, у разных корпусов разные.
func TestRefs(t *testing.T) {
	lib, err := LoadLib("../../masters/lib/mps.kicad_sym")
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range variants {
		sh := Build(lib, v, "test")
		pkgs := map[string]map[string]bool{} // префикс → обозначения
		for _, c := range sh.syms {
			if c.Ref[0] == '#' {
				continue
			}
			p := refPrefix(c.Ref)
			if pkgs[p] == nil {
				pkgs[p] = map[string]bool{}
			}
			pkgs[p][c.Ref] = true
		}
		for p, refs := range pkgs {
			var nums []int
			for r := range refs {
				nums = append(nums, refNum(r))
			}
			sort.Ints(nums)
			for i, n := range nums {
				if n != i+1 {
					t.Errorf("%dx%d: %s — номера %v, ждём 1..%d подряд", v.Cols, v.Rows, p, nums, len(nums))
					break
				}
			}
		}
		want := 10
		if v.Decoder {
			want = 11
		}
		if len(pkgs["DD"]) != want {
			t.Errorf("корпусов DD %d, ждём %d", len(pkgs["DD"]), want)
		}
	}
}
