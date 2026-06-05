#!/usr/bin/env bash
# Demo driver: run in the background so the segment escalates on its own (a
# time-lapse), without typing anything into the visible pane.
sleep 4
demo/age.sh due
sleep 2
demo/age.sh overdue
sleep 2
demo/age.sh critical
