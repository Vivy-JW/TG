#!/usr/bin/env bash
# Regenerate corpus.manifest -- the pinned input set for the four batch drivers.
#
# Why this exists: batch.sh used to glob *.tzs without excluding the drivers' own scratch
# files, so its stored baseline included _ai_*.tzs left behind by earlier AddField runs.
# Those files appear and disappear between sessions, so the baseline was not reproducible and
# no refactor could be proven against it. The three write-side drivers already excluded their
# own scratch (_dw/_ed/_del/_ac); the read side did not.
#
# Two prefixes were added 2026-09-26, and neither is a driver's scratch:
#   _tdev*    the corpus test's own output (internal/dev/tzs/scratchPath), deleted after each
#             use but left behind when a run is killed;
#   _tt_dry*  the ENGINE's dry-run rollback snapshot (DryRun.cs), which is written beside the
#             source package because TzpManager only loads from inside the workspace. It is
#             normally deleted too -- but a daemon that dies mid-rollback leaves one, and a
#             scratch package that lands in the pinned set is exactly the unreproducible
#             baseline this script exists to prevent.
#
#   ./make-manifest.sh [root]      default root: /d/t100_wrok_dir
#
# The manifest is a record, not (yet) the discovery mechanism -- the four drivers still glob,
# with the exclusions now aligned across all four. Wiring discovery to the manifest is a
# Wave 3 change, because doing it now would shift the baselines again.

set -u
ROOT="${1:-/d/t100_wrok_dir}"
HERE_POSIX="$(cd "$(dirname "$0")" && pwd)"

{
  echo "# Corpus manifest -- the pinned input set for the four batch drivers."
  echo "# Regenerate: ./make-manifest.sh [root]"
  echo "# Format: <sha256-16>  <bytes>  <path>"
  echo "#"
  echo "# batch.sh used to glob *.tzs without excluding the drivers' own scratch files, so its"
  echo "# stored baseline included _ai_*.tzs from earlier AddField runs. Those can appear and"
  echo "# disappear between runs, which made the baseline unreproducible. This list is the fix."
  echo
  find "$ROOT" -iname '*.tzs' \
       ! -name '_ai*' ! -name '_dw*' ! -name '_ed*' ! -name '_del*' ! -name '_ac*' \
       ! -name '_tdev*' ! -name '_tt_dry*' \
       -print0 \
    | sort -z \
    | while IFS= read -r -d '' f; do
        printf '%s  %8d  %s\n' "$(sha256sum "$f" | cut -c1-16)" "$(stat -c%s "$f")" "$f"
      done
} > "$HERE_POSIX/corpus.manifest"

echo "wrote $HERE_POSIX/corpus.manifest: $(grep -vce '^#\|^$' "$HERE_POSIX/corpus.manifest") files"
