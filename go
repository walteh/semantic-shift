#!/usr/bin/env bash
set -euo pipefail

# downloads folder
dl=$HOME/Downloads/logd

# Create logs directory if it doesn't exist
mkdir -p "$dl"

# Log file for detailed debugging
LOG_FILE="$dl/go-wrapper.log"
STDERR_LOG="$dl/go-stderr.log"
TIMESTAMP=$(date +"%Y-%m-%d %H:%M:%S")
echo "[$TIMESTAMP] Running go with args: $@" | tee -a "$LOG_FILE"

# Special handling for taskmcp to debug stdio interaction with Cursor
if [[ "$*" == *"taskmcp"* && "$*" != *"-http"* ]]; then
	echo "[$TIMESTAMP] Detected taskmcp in stdio mode, adding special logging" >>"$LOG_FILE"

	# Log environment variables
	echo "[$TIMESTAMP] Environment:" >>"$LOG_FILE"
	env | sort >>"$LOG_FILE"

	# Files to store diagnostic info
	TASKMCP_LOG="$dl/taskmcp-debug.log"
	STDIN_LOG="$dl/taskmcp-stdin.log"
	STDOUT_LOG="$dl/taskmcp-stdout.log"

	# Log that we're starting
	echo "[$TIMESTAMP] Starting taskmcp in direct stdio mode. Debug log: $TASKMCP_LOG" >>"$LOG_FILE"

	# Record stdin if possible (non-blocking)
	timeout 0.1s cat >"$STDIN_LOG" || true &

	# Extract the arguments for taskmcp and add our debug flags
	# We need to be careful not to interpret the debug flag as part of the "go" command
	ARGS=()
	for ARG in "$@"; do
		ARGS+=("$ARG")
	done
	ARGS+=("-log" "$TASKMCP_LOG" "-log-level" "debug")

	# Run taskmcp with stderr redirected to our log
	# Note: we pass arguments as separate array elements to avoid shell interpretation issues
	echo "[$TIMESTAMP] Running: go ${ARGS[*]}" >>"$LOG_FILE"
	go "${ARGS[@]}" 2> >(tee -a "$STDERR_LOG" >&2)
	EXIT_CODE=$?

	# Log completion
	TIMESTAMP=$(date +"%Y-%m-%d %H:%M:%S")
	echo "[$TIMESTAMP] taskmcp completed with exit code $EXIT_CODE" >>"$LOG_FILE"

	# If it failed, capture the failure
	if [[ $EXIT_CODE -ne 0 ]]; then
		echo "[$TIMESTAMP] taskmcp FAILED with exit code $EXIT_CODE" >>"$LOG_FILE"
		echo "[$TIMESTAMP] See $TASKMCP_LOG and $STDERR_LOG for details" >>"$LOG_FILE"
	fi

	exit $EXIT_CODE
fi

# if first argument is "test", use gotestsum
if [ "${1:-}" == "test" ]; then
	shift

	cc=0
	ff=0
	real_args=()
	extra_args=""
	max_lines=1000 # Default to 1000 lines

	# Handle each argument
	for arg in "$@"; do
		if [ "$arg" = "-custom-coverage" ]; then
			cc=1
		elif [ "$arg" = "-force" ]; then
			ff=1
		elif [[ "$arg" =~ ^-max-lines=(.*)$ ]]; then
			max_lines="${BASH_REMATCH[1]}"
		else
			real_args+=("$arg")
		fi
	done

	if [[ "$cc" == "1" ]]; then
		tmpcoverdir=$(mktemp -d)
		function print_coverage() {
			echo "================================================"
			echo "Function Coverage"
			echo "------------------------------------------------"
			go tool cover -func=$tmpcoverdir/coverage.out
			echo "================================================"

		}
		extra_args=" -coverprofile=$tmpcoverdir/coverage.out -covermode=atomic "
		trap "print_coverage" EXIT
	fi

	if [[ "$ff" == "1" ]]; then
		extra_args="$extra_args -count=1 "
	fi

	# Use our truncation wrapper
	./scripts/truncate-test-logs.sh "$max_lines" -- ./go tool gotest.tools/gotestsum \
		--format pkgname \
		--format-icons hivis \
		--hide-summary=skipped \
		--raw-command -- go test -v -vet=all -json -cover $extra_args "${real_args[@]}"
	exit $?
fi

if [ "${1:-}" == "tool" ]; then
	shift
	escape_regex() {
		printf '%s\n' "$1" | sed 's/[][(){}.*+?^$|\\]/\\&/g'
	}
	errors_to_suppress=(
		# https://github.com/protocolbuffers/protobuf-javascript/issues/148
		"plugin.proto#L122"
	)
	# 🔧 Build regex for suppressing errors
	errors_to_suppress_regex=""
	for phrase in "${errors_to_suppress[@]}"; do
		escaped_phrase=$(escape_regex "$phrase")
		if [[ -n "$errors_to_suppress_regex" ]]; then
			errors_to_suppress_regex+="|"
		fi
		errors_to_suppress_regex+="$escaped_phrase"
	done

	# Log the tool command being executed
	echo "[$TIMESTAMP] Running go tool: go tool $@" >>"$LOG_FILE"
	go tool "$@" <&0 >&1 2> >(tee -a "$STDERR_LOG" | grep -Ev "$errors_to_suppress_regex" >&2)

	# Log completion
	TIMESTAMP=$(date +"%Y-%m-%d %H:%M:%S")
	echo "[$TIMESTAMP] go tool completed with exit code $?" >>"$LOG_FILE"
	exit $?
fi

# For standard go commands, add logging
echo "[$TIMESTAMP] Running go command: go $@" >>"$LOG_FILE"

# Run go with stderr redirected to our log
go "$@" 2> >(tee -a "$STDERR_LOG" >&2)
EXIT_CODE=$?

# Log completion
TIMESTAMP=$(date +"%Y-%m-%d %H:%M:%S")
echo "[$TIMESTAMP] go command completed with exit code $EXIT_CODE" >>"$LOG_FILE"
exit $EXIT_CODE
