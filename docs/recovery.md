# File ownership, plans and interrupted writes

Generation is deliberately non-destructive. Existing business implementations are
preserved, existing unowned targets are conflicts, and managed content must match
its recorded previous hash. The checker also compares managed files against canonical
rendered output, so resetting a hash does not by itself make a changed file valid.

`--dry-run` does not create the project directory, lock, files or journal. The plan
lists create/update/keep/preserve actions and hashes. `--expect ID` binds application
to a reviewed plan. The complete preflight rechecks tracked files after the writer
lock is acquired, before it starts committing changes.

The writer obtains `.gsvc/write.lock` with exclusive creation. It creates a journal,
then uses temporary files and per-file replacement. Ordinary write failures trigger
rollback; an interrupted process can leave `.gsvc/transaction.json` for explicit
recovery. Changes made outside that transaction are not silently overwritten.

This is not a multi-file atomic filesystem transaction, a power-loss durability
protocol, or protection against a hostile process racing filesystem changes. Symlinks
inside the project are rejected; the project root's existing ancestor symlinks are
resolved to accommodate normal platform paths such as macOS /tmp.

## Procedure

Run from the project root, or pass the root explicitly:

    gsvc recover --root . --dry-run --json

If a write lock exists, first confirm that no gsvc writer is still running. Do not
remove an active writer's lock. A process crash can leave a stale lock; remove only
that stale `.gsvc/write.lock` after confirming the process is gone.

Then run:

    gsvc recover --root . --json
    gsvc check --json

If recovery reports an unrelated edit, preserve and reconcile that edit manually.
Do not use a hash reset, directory deletion, or forced regeneration as a shortcut.
Keep version control as the recovery source for developer-owned business files.

Recovery removes files created by an interrupted initial scaffold but may leave empty
directories. After recovering an interrupted `init`, restart in a fresh empty target
or remove only those verified-empty directories. Recovery never recursively removes
arbitrary project directories.
