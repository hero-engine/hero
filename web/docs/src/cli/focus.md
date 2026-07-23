# Personal Focus

Focus is a private, user-global list of small intentions backed by exact saved
prompts. A Focus item is not a spec, Intake item, tracker issue, or harness todo,
and Hero never synchronizes it with those systems.

```sh
# Prompts are read from a file or stdin, never a command-line argument.
hero focus add --title "Finish the parser" --prompt-file prompt.txt --project hero --state today
printf 'Resume the parser work.\n' | hero focus add --title "Parser" --prompt-file -

hero focus list                         # active items; excludes done
hero focus list --state all --json
hero focus show focus_abcd --json
hero focus move focus_abcd later --revision 123
hero focus done focus_abcd --revision 456
hero focus launch focus_abcd --json
```

When `--project` is omitted inside a workspace registered with Hero, Focus
captures that workspace's canonical peer ID and current registry name. Outside
a registered workspace, the item remains unbound. Pass `--project` to select a
different registered project explicitly.

The lifecycle states are `inbox`, `today`, `later`, and `done`. Move and done
commands require the revision returned by the previous read or mutation; a
stale revision fails instead of overwriting another client's change. Repeating
the same move with the current revision is idempotent.

`launch` returns the saved prompt and an absolute registered-project target. It
does not invoke a harness, create a session, or mark the item done. If the
project is unavailable—or the item is intentionally unbound—launch fails rather
than falling back to the current directory.

Focus files live below the user's Hero state directory with private permissions.
Treat saved prompts as sensitive. There is intentionally no destructive delete
command in v1; move an item to `done` instead.
