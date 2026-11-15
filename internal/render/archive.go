package render

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"mpskp/internal/variant"
)

// ArchivedVariant provides both inputs from the schema selected by schema_fetch.sh.
func ArchivedVariant(dir string, current *variant.Params) (*variant.Params, string, error) {
	if _, err := os.Stat(filepath.Join(dir, ".schema")); os.IsNotExist(err) {
		return current, Asm(current), nil
	} else if err != nil {
		return nil, "", err
	}
	data, err := os.ReadFile(filepath.Join(dir, "params.json"))
	if err != nil {
		return nil, "", err
	}
	var saved variant.Params
	if err := json.Unmarshal(data, &saved); err != nil {
		return nil, "", fmt.Errorf("params.json: %w", err)
	}
	if saved.Year != current.Year || saved.Group != current.Group || saved.M != current.M || saved.Student != current.Student {
		return nil, "", fmt.Errorf("params.json не соответствует выбранному студенту или году ТЗ")
	}
	asm, err := os.ReadFile(filepath.Join(dir, "vars.inc"))
	if err != nil {
		return nil, "", err
	}
	return &saved, string(asm), nil
}
