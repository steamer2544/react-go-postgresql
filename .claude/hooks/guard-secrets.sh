#!/usr/bin/env bash
# guard-secrets.sh — PreToolUse hook (matcher: Read|Edit|Write|MultiEdit)
# บล็อกการอ่าน/แก้ไฟล์ที่เป็นความลับจริง (.env, *.key, *.pem, secrets/)
# แต่ยอมให้แตะ .env.example / .env.sample / .env.template ได้ (ไฟล์ตัวอย่างที่ commit ได้)
#
# กลไก: อ่าน JSON จาก stdin -> ดึง tool_input.file_path -> ถ้า match secret ให้ exit 2 (บล็อก)
#   exit 2 = บล็อก tool call + ส่ง stderr กลับให้ Claude เห็นเหตุผล
#   exit 0 = ปล่อยผ่าน
set -euo pipefail

input="$(cat)"

# ดึง file_path: ใช้ jq ถ้ามี ไม่งั้น fallback python3 (เครื่อง dev ส่วนใหญ่มีอย่างน้อยหนึ่งอย่าง)
extract_path() {
  if command -v jq >/dev/null 2>&1; then
    printf '%s' "$1" | jq -r '.tool_input.file_path // empty'
  elif command -v python3 >/dev/null 2>&1; then
    printf '%s' "$1" | python3 -c 'import sys,json;print(json.load(sys.stdin).get("tool_input",{}).get("file_path",""))' 2>/dev/null
  else
    printf ''   # ไม่มีตัว parse -> ไม่บล็อก (fail-open เพื่อไม่ให้ session พัง; permissions.deny ยังคุ้ม Read อยู่)
  fi
}

path="$(extract_path "$input")"
[ -z "$path" ] && exit 0
base="$(basename "$path")"

# อนุญาตไฟล์ตัวอย่างเสมอ
case "$base" in
  .env.example|.env.sample|.env.template) exit 0 ;;
esac

# บล็อก secret จริง
if [[ "$base" == ".env" || "$base" == .env.* || "$base" == *.pem || "$base" == *.key ]] \
   || [[ "$path" == *"/secrets/"* ]]; then
  echo "ถูกบล็อกโดย guard-secrets: '$path' เป็นไฟล์ความลับ ห้ามอ่าน/แก้ผ่าน agent — แก้ด้วยมือ และ commit เฉพาะ .env.example" >&2
  exit 2
fi

exit 0
