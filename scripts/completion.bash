# Bash tab completion for game names in scripts.
#
# Source this in your shell:
#   source scripts/completion.bash
#
# Then tab-complete works:
#   scripts/build.sh mon<TAB>  →  scripts/build.sh monkey1

_scumm_game_completion() {
    local cur="${COMP_WORDS[COMP_CWORD]}"

    # Only complete the first non-flag argument
    local i arg_count=0
    for ((i=1; i<COMP_CWORD; i++)); do
        [[ "${COMP_WORDS[i]}" != --* ]] && ((arg_count++))
    done
    [[ "$cur" == --* ]] && return
    ((arg_count > 0)) && return

    # Find repo root from the script path
    local script="${COMP_WORDS[0]}"
    local script_dir
    script_dir="$(cd "$(dirname "$script")" 2>/dev/null && pwd)"

    local repo_root
    if [[ "$(basename "$script_dir")" == "scripts" ]]; then
        repo_root="$(dirname "$script_dir")"
    else
        repo_root="$script_dir"
    fi

    [[ -d "$repo_root/games" ]] || return

    local games
    games=$(cd "$repo_root/games" && ls -1d */ 2>/dev/null | sed 's|/$||')
    COMPREPLY=($(compgen -W "$games" -- "$cur"))
}

for _script in build.sh clean.sh clean_assets.sh test.sh extract.sh extract_pak.sh extract_assets.sh init_translation.sh; do
    complete -F _scumm_game_completion "$_script"
    complete -F _scumm_game_completion "./$_script"
    complete -F _scumm_game_completion "scripts/$_script"
    complete -F _scumm_game_completion "./scripts/$_script"
    complete -F _scumm_game_completion "bash scripts/$_script"
    complete -F _scumm_game_completion "bash ./scripts/$_script"
done
unset _script
