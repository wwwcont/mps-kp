package pz

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Pandoc — путь к pandoc (переменная PANDOC, иначе из PATH).
func Pandoc() string {
	if p := os.Getenv("PANDOC"); p != "" {
		return p
	}
	return "pandoc"
}

// Build собирает docx: markdown → pandoc с эталоном стилей. Рисунки ищутся в d.Dir.
func Build(d *Doc, out string) error {
	ref, err := ReferenceDocx(Pandoc(), d.Style)
	if err != nil {
		return err
	}
	refPath := filepath.Join(d.Dir, "reference.docx")
	if err := os.WriteFile(refPath, ref, 0o644); err != nil {
		return err
	}
	md := filepath.Join(d.Dir, strings.TrimSuffix(filepath.Base(out), filepath.Ext(out))+".md")
	if err := os.WriteFile(md, []byte(d.Markdown()), 0o644); err != nil {
		return err
	}
	args := []string{md, "-f", "markdown-implicit_figures+implicit_figures", "-o", out, "--reference-doc", refPath, "--resource-path", d.Dir,
		"--columns=40"} // ширины колонок таблиц — по разделителям (иначе короткие таблицы верстаются «по содержимому»)
	cmd := exec.Command(Pandoc(), args...)
	if b, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("pandoc: %v\n%s", err, b)
	}
	os.Remove(refPath)
	// порядок элементов по схеме OOXML: Word строже LibreOffice (см. ooxml.go)
	return normalizeDocx(out, d.media, d.Style.HasAbstract)
}
