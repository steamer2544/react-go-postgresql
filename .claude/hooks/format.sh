#!/usr/bin/env bash
# format.sh — PostToolUse hook (matcher: Write|Edit|MultiEdit)
# จัดฟอร์แมตไฟล์ที่เพิ่งถูกเขียน/แก้ ตามชนิดไฟล์:
#   .go                                   -> gofmt -w  (+ goimports ถ้ามี)
#   .js .jsx .ts .tsx .css .scss .json .md -> prettier ที่ติดตั้งใน frontend/ (ไม่ยิงออกเน็ต)
# ปลอดภัยเสมอ: ถ้าเครื่องมือไม่มี/ไฟล์ไม่เข้าเงื่อนไข -> exit 0 (ไม่บล็อก)
# หมายเหตุ: PostToolUse ยกเลิกการเขียนไม่ได้อยู่แล้ว หน้าที่คือ normalize หลังเขียน
set -euo pipefail

input="$(cat)"

extract_path() {
  if command -v jq >/dev/null 2>&1; then
    printf '%s' "$1" | jq -r '.tool_input.file_path // empty'
  elif command -v python3 >/dev/null 2>&1; then
    printf '%s' "$1" | python3 -c 'import sys,json;print(json.load(sys.stdin).get("tool_input",{}).get("file_path",""))' 2>/dev/null
  else
    printf ''
  fi
}

path="$(extract_path "$input")"
[ -z "$path" ] && exit 0
[ -f "$path" ] || exit 0

case "$path" in
  *.go)
    command -v gofmt    >/dev/null 2>&1 && gofmt -w "$path" || true
    command -v goimports >/dev/null 2>&1 && goimports -w "$path" || true
    ;;
  *.js|*.jsx|*.ts|*.tsx|*.css|*.scss|*.json|*.md)
    local_prettier="${CLAUDE_PROJECT_DIR:-.}/frontend/node_modules/.bin/prettier"
    [ -x "$local_prettier" ] && "$local_prettier" --write --log-level warn "$path" 2>/dev/null || true
    ;;
  *) : ;;
esac

exit 0
