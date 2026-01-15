#!/usr/bin/env bash

# Environment setup
set -o pipefail

# NO_COLOR support (https://no-color.org/)
# When NO_COLOR is set (to any value), disable all color output
if [[ -n "${NO_COLOR}" ]]; then
  export RED=''
  export BGRED=''
  export GREEN=''
  export MAGENTA=''
  export CYAN=''
  export BLUE=''
  export YELLOW=''
  export WHITE=''
  export NORMAL=''
  export BOLD=''
  export CLEARLINE=''
else
  # Define style constants
  export RED=$'\033[31m'
  export BGRED=$'\033[41m'
  export GREEN=$'\033[32m'
  export MAGENTA=$'\033[35m'
  export CYAN=$'\033[36m'
  export BLUE=$'\033[34m'
  export YELLOW=$'\033[33m'
  export WHITE=$'\033[37m'
  export NORMAL=$'\033[0m'
  export BOLD=$'\033[01m'
  export CLEARLINE=$'\033[K'
fi
