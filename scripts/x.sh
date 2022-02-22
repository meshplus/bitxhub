RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

function print_blue() {
  printf "${BLUE}%s${NC}\n" "$1"
}

function print_green() {
  printf "${GREEN}%s${NC}\n" "$1"
}

function print_red() {
  printf "${RED}%s${NC}\n" "$1"
}

# The sed commend with system judging
# Examples:
# sed -i 's/a/b/g' bob.txt => x_replace 's/a/b/g' bob.txt
function x_replace() {
  system=$(uname)

  if [ "${system}" = "Linux" ] || [[ "${system}" =~ "MINGW" ]]; then
    sed -i "$@"
  else
    sed -i '' "$@"
  fi
}
