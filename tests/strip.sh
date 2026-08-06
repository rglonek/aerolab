#!/bin/bash

# strip.sh reduces the tier logs written by everything.sh to their summaries:
# the test structure lines plus each file's first two and last two lines. What it
# drops is the per-command detail -- every `$ aerolab ...` invocation and its
# output -- which is the bulk of the bytes and useless once a run is green.
#
# Usage:
#   ./tests/strip.sh                  # summarise every tests/log/*.log to stdout
#   ./tests/strip.sh log/gcp.log      # summarise the named files instead
#   ./tests/strip.sh -i               # rewrite the files in place
#
# Stripping is idempotent: a stripped file keeps the same first/last two lines
# and every line the filter matched, so re-running changes nothing.
#
# In-place is not the default on purpose. The detail is what a failure is
# diagnosed from, and it cannot be recovered without re-running the tier -- an
# hour and, for the cloud tiers, real money. Strip after a run is understood,
# not before.

set -uo pipefail

IN_PLACE=0
if [ "${1:-}" = "-i" ] || [ "${1:-}" = "--in-place" ]; then
	IN_PLACE=1
	shift
fi

TESTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

FILES=("$@")
if [ ${#FILES[@]} -eq 0 ]; then
	# Nothing named: every tier log. nullglob so an empty log dir is an error
	# with a message rather than a literal '*.log' passed to awk.
	shopt -s nullglob
	FILES=("${TESTS_DIR}"/log/*.log)
	shopt -u nullglob
	if [ ${#FILES[@]} -eq 0 ]; then
		echo "strip.sh: no logs found in ${TESTS_DIR}/log" >&2
		exit 1
	fi
fi

# summarise prints the wanted lines of $1, in file order and without repeats.
#
# The line set:
#   - the first two lines: the run's timestamp and the command it ran
#   - the last two lines: the exit code and the finish timestamp
#   - `=== RUN` / `=== PAUSE` / `=== CONT` / `=== NAME`: which test is running
#   - `--- PASS` / `--- FAIL` / `--- SKIP` at any indent: the per-test verdicts.
#     Indented ones are subtest verdicts (`--- FAIL: TestDockerOSMatrix/ubuntu-22.04`)
#     and matter most -- naming which entry of a matrix failed is the whole point
#     of a summary, so the filter cannot be anchored to column 0.
#   - `ok` / `FAIL` / `PASS` at column 0: the per-package and overall verdicts
#   - panics, fatal errors and `make: ***`: how a tier died when it did not fail
#     through a test, which no test line would show
#   - the messages tests emit: testify's assertion blocks (`Error:` and the lines
#     under it, `Test:`, `Error Trace:`) and `some_test.go:NN: message` lines,
#     which are the skip reasons and failure explanations. A verdict without its
#     reason is not enough to act on, and after an in-place strip the reason is
#     not recoverable, so it belongs in the summary rather than only in the
#     detail. The one exception is a `$ aerolab ...` line: that heads a command
#     and its output, which is exactly the detail being dropped.
summarise() {
	local file=$1
	local total
	total=$(wc -l <"${file}")
	awk -v total="${total}" '
		FNR <= 2 { print; inerr = 0; next }
		FNR > total - 2 { print; inerr = 0; next }
		/^=== (RUN|PAUSE|CONT|NAME)/ { print; inerr = 0; next }
		/^[[:space:]]*--- (PASS|FAIL|SKIP):/ { print; inerr = 0; next }
		/^(ok|FAIL|PASS)([ \t]|$)/ { print; inerr = 0; next }
		/^(panic:|fatal error:|\[signal |make: \*\*\*)/ { print; inerr = 0; next }
		/^[[:space:]]+(Error Trace|Messages|Test):/ { print; inerr = 0; next }
		/^[[:space:]]+Error:/ { print; inerr = 1; next }
		# The text of an assertion failure sits on continuation lines under
		# Error:, distinguishable only by being indented further, so it is
		# collected by state rather than by pattern.
		inerr && /^[[:space:]]/ { print; next }
		inerr { inerr = 0 }
		/^[[:space:]]+[A-Za-z0-9_.-]+_test\.go:[0-9]+:/ {
			if ($0 ~ /: \$ aerolab /) { next }
			print; next
		}
	' "${file}"
}

human() {
	# wc -c rather than du, which rounds every small file up to a block and
	# would report a 3KB summary as the same size as the 4KB one before it.
	local bytes
	bytes=$(wc -c <"$1")
	if [ "${bytes}" -lt 1024 ]; then
		# Arithmetic expansion, because wc pads its output with spaces.
		echo "$((bytes))B"
	else
		echo "$((bytes / 1024))K"
	fi
}

RET=0

for file in "${FILES[@]}"; do
	if [ ! -f "${file}" ]; then
		echo "strip.sh: ${file}: no such file" >&2
		RET=1
		continue
	fi
	if [ ${IN_PLACE} -eq 0 ]; then
		if [ ${#FILES[@]} -gt 1 ]; then
			printf '==> %s <==\n' "${file}"
		fi
		summarise "${file}"
		if [ ${#FILES[@]} -gt 1 ]; then
			printf '\n'
		fi
		continue
	fi
	before=$(human "${file}")
	tmp="${file}.stripped.$$"
	if ! summarise "${file}" >"${tmp}"; then
		rm -f "${tmp}"
		echo "strip.sh: ${file}: failed to summarise, left untouched" >&2
		RET=1
		continue
	fi
	mv "${tmp}" "${file}"
	echo "${file}: ${before} -> $(human "${file}")"
done

exit ${RET}
