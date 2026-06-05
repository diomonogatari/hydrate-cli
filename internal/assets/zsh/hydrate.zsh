# hydrate — stamp shell activity for the "terminal in use" notification gate.
#
# Source this from your interactive zsh (e.g. add `hydrate` to your module list,
# or `source /path/to/hydrate.zsh`). It is deliberately output-silent so it is
# safe with Powerlevel10k's instant prompt, and fork-free on the hot path: it
# uses zsh's EPOCHSECONDS parameter instead of spawning date(1).

# Load only the EPOCHSECONDS parameter from zsh/datetime (cheapest possible).
zmodload -F zsh/datetime p:EPOCHSECONDS 2>/dev/null

typeset -g _HYDRATE_DIR="${XDG_STATE_HOME:-$HOME/.local/state}/hydrate"
[[ -d "$_HYDRATE_DIR" ]] || mkdir -p "$_HYDRATE_DIR" 2>/dev/null

_hydrate_mark_activity() {
  # No subprocess, no output. EPOCHSECONDS comes from zsh/datetime.
  print -rn -- "$EPOCHSECONDS" >| "$_HYDRATE_DIR/last_activity" 2>/dev/null
}

autoload -Uz add-zsh-hook
add-zsh-hook preexec _hydrate_mark_activity
add-zsh-hook precmd  _hydrate_mark_activity

# Ergonomics: the most common action is the shortest command.
alias w='hydrate log'      # log one default glass
alias ww='hydrate status'  # today at a glance
