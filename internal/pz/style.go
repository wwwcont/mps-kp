package pz

import (
	"hash/fnv"
)

// DocStyle — варианты подписей и колонтитула при общем кафедральном оформлении.
type DocStyle struct {
	Name        string
	HasAbstract bool
	HeaderText  string // верхний колонтитул справа ("" — нет)
	H1Caps      bool   // прописными
	FigShort    bool   // «Рис. N. …» (иначе «Рисунок N — …»)
	CodeSz      int    // кегль листингов в полупунктах (0 — 10 pt); подбирает FitCode под самую длинную строку
}

var docStyles = map[string]DocStyle{
	"A": {Name: "A", HeaderText: "Курсовая работа, МПС ч2", H1Caps: true, FigShort: true},
	"B": {Name: "B", H1Caps: true, FigShort: true},
	"C": {Name: "C"},
}

// PickDocStyle — стиль по имени (A, B, C) или, если пусто, от строки-зерна (ФИО|группа|M).
func PickDocStyle(name, seed string) DocStyle {
	if s, ok := docStyles[name]; ok {
		return s
	}
	h := fnv.New32a()
	h.Write([]byte(seed + "|doc"))
	return docStyles[string(rune('A'+h.Sum32()%3))]
}

func (ds DocStyle) codeSz() int {
	if ds.CodeSz > 0 {
		return ds.CodeSz
	}
	return 20
}

// FitCode — кегль листинга (полупункты, 20…24 = 10…12 pt), при котором строка из maxCols символов Courier New
// (ширина знака 0,6 кегля) помещается в поле текста 175 мм; длиннее 10 pt — переносится.
func FitCode(maxCols int) int {
	if maxCols <= 0 {
		return 20
	}
	sz := int(2 * 175 / 25.4 * 72 / (0.6 * float64(maxCols)))
	return max(20, min(24, sz))
}
