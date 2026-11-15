#!/usr/bin/env bash
# Готовит папку сборки студента для КМ-2/КМ-3: куда класть результаты (.dest) и схему, по которой собирать.
#   scripts/schema_fetch.sh <ник> [N] [--optional]
# Берёт из ветки results папку <группа>/<M> <ФИО>/СХЕМА-N (пусто — последнюю), распаковывает в build/<группа>/<M>/,
# сверяет её данные с students/<ник>/variant.yaml и проверяет цепи буфера (bin/mpsgen -verify) и пишет .schema = «СХЕМА-N».
# Нет схемы: ошибка «сначала КМ-1»; с --optional — только предупреждение (КМ-3 проверяет код и без схемы).
set -euo pipefail
nick="${1:?ник студента}"; want="${2:-}"; optional="${3:-}"
[ "$want" = "--optional" ] && { optional="--optional"; want=""; }
y="students/$nick/variant.yaml"
[ -f "$y" ] || { echo "::error::нет $y — ник это имя папки в students/ (make nick G=… FIO=…)"; exit 1; }
[ -x bin/mpsgen ] || go build -o bin/mpsgen ./cmd/mpsgen
info="$(bin/mpsgen -student "$y" -info)"
dest="$(python3 -c 'import json,sys; print(json.loads(sys.argv[1])["dest"])' "$info")"
build="$(python3 -c 'import json,sys; print(json.loads(sys.argv[1])["build"])' "$info")"
mkdir -p "$build"
echo "$dest" > "$build/.dest"
echo "$nick" > "$build/.nick"
rm -f "$build/.schema"

miss() {
  if [ "$optional" = "--optional" ] && [ -z "$want" ]; then echo "::warning::$nick: $1 — собираю без привязки к схеме"; exit 0; fi
  echo "::error::$nick: $1"; exit 1
}
git fetch -q --depth=1 origin results 2>/dev/null || miss "ветки results ещё нет — сначала «КМ-1 схема и перечень» для $nick"
runs="$(git -c core.quotePath=false ls-tree --name-only FETCH_HEAD "$dest/" 2>/dev/null | sed -nE 's|.*/СХЕМА-([0-9]+)$|\1|p' | sort -n)"
[ -n "$runs" ] || miss "схемы ещё нет (results/$dest/СХЕМА-N) — сначала «КМ-1 схема и перечень» для $nick"
n="${want:-$(echo "$runs" | tail -1)}"
echo "$runs" | grep -qx "$n" || miss "нет СХЕМА-$n (есть: $(echo $runs | sed 's/ /, /g'))"
src="$dest/СХЕМА-$n"
git cat-file -e "FETCH_HEAD:$src/meta.json" 2>/dev/null || miss "СХЕМА-$n старого формата (без meta.json)"
rm -f "$build"/fragment-*.png
# файлы схемы → папка сборки (как будто КМ-1 собрал её здесь)
git archive FETCH_HEAD "$src" | tar -x -C "$build" --strip-components=3
[ -f "$build/meta.json" ] || miss "СХЕМА-$n старого формата (без meta.json) — прогони «КМ-1 схема и перечень», будет новая"
bin/mpsgen -student "$y" -verify "$build/meta.json" || { echo "::error::$nick: СХЕМА-$n не подходит для сборки (см. выше)"; exit 3; }
echo "СХЕМА-$n" > "$build/.schema"
echo "$nick: $src → $build"
