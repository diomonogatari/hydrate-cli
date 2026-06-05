#!/usr/bin/env bash
# Demo helper: age the sandbox log so the segment renders a given urgency level,
# then refresh the bar. Keeps the count at 4/8. Usage: age.sh <due|overdue|critical>
# (interval ~2.875h for goal 2000 / glass 250 over a 0–23 window.)
case "$1" in
  due)      ago=12000 ;;   # ~1.16x interval
  overdue)  ago=19000 ;;   # ~1.84x
  critical) ago=28800 ;;   # ~2.78x
  *)        ago=60 ;;
esac
now=$(date +%s)
printf '%s' "$now" > "$XDG_STATE_HOME/hydrate/last_activity"   # mark "in use" → no notify
{
  printf '{"ts":%d,"ml":250}\n' "$((now - ago - 600))"
  printf '{"ts":%d,"ml":250}\n' "$((now - ago - 400))"
  printf '{"ts":%d,"ml":250}\n' "$((now - ago - 200))"
  printf '{"ts":%d,"ml":250}\n' "$((now - ago))"
} > "$XDG_STATE_HOME/hydrate/log.jsonl"
hydrate tick >/dev/null 2>&1
