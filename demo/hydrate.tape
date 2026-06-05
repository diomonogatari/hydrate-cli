# hydrate — demo GIF.  Generate from the repo root:  vhs demo/hydrate.tape
# Requires: vhs, ttyd, ffmpeg, and `hydrate` on PATH. Fully sandboxed (see seed.sh).

Output demo/hydrate.gif

Set Shell bash
Set FontSize 22
Set Width 1100
Set Height 720
Set Padding 26
Set Theme "Catppuccin Mocha"
Set TypingSpeed 60ms

# --- sandboxed setup (hidden) ---
Hide
Type "source demo/seed.sh"
Enter
Type "clear"
Enter
Show
Sleep 600ms

# --- 1) guided setup, walked quickly ---
Type "hydrate init"
Enter
Sleep 1500ms
Enter
Sleep 500ms
Enter
Sleep 500ms
Enter
Sleep 500ms
Enter
Sleep 500ms
Enter
Sleep 750ms
Enter
Sleep 750ms
Enter
Sleep 750ms
Enter
Sleep 2000ms
Type "clear"

# --- 2) the always-on tmux status segment ---
Hide
Type "clear"
Enter
Type "tmux -f demo/tmux.conf new-session -s demo"
Enter
Sleep 900ms
Type "demo/escalate.sh & disown"
Enter
Type "clear"
Enter
Show
Sleep 1300ms

# log a glass — calm blue
Type "hydrate log"
Enter
Sleep 2300ms
# (time-lapse: the segment escalates due -> overdue -> critical on its own)
Sleep 2000ms
Sleep 2000ms
Sleep 1600ms
# log again — back to calm
Type "hydrate log"
Enter
Sleep 1750ms
