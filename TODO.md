# TODO

Things that are on my list to knock off.

## Show elapsed time per session in the picker

Each session row in the fzf picker could show how long the session has been in its current state (e.g. `12m` since last state change). This is useful for spotting stuck sessions at a glance.

The hard part is defining "start": the session JSON needs a reliable `last_status_changed_at` timestamp that gets written whenever status transitions (idle→working, working→completed, etc.). Currently `last_updated` is updated on every hook event, not just status changes, so it can't be used directly. Needs a schema change + hook handler update before the picker render can compute elapsed time correctly.

## Test with Codex

I don't actually use Codex so none of the Codex paths are actually tested at this point. Only the Claude paths are known to work. I should rectify that at some point and actually test with Codex.

## Add a screen recording to the README

I ended up dropping the one that came from the original repo because it was for the original plugin and not this version.
