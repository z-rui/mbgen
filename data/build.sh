#!/bin/bash

set -o errexit
set -o xtrace

MBTOOL=~/go/bin/mbtool
MBGEN=../mbgen

SHORTFILE="short"

# 根据需要增减词库
WORDFILES=(
	words0
	words1
	words2
	words3
	THUOCL_lishimingren
	emoji
)

BOOTFILES=(
	1 2 3	# 《规范字表》
	gb2312 gbk ext
)

CHARFILES=("${BOOTFILES[@]}")

ALLFILES=(
	"${SHORTFILE}"
	words0 1 2
	words2
	words1
	words3
	3 gb2312 gbk ext
	emoji
	THUOCL_lishimingren
)

# ==================== FCITX ====================
FCITX_TEMPLATE="fcitx.tmpl"
output_fcitx() {
	local -r outfile="zm-all.txt"
	cat "$FCITX_TEMPLATE" > "$outfile"
	$MBTOOL -l -s=' ' "${ALLFILES[@]/%/.mb}" >> "$outfile"
}

# ==================== RIME ====================
RIME_TEMPLATE="rime.tmpl"
output_rime() {
	local -r outfile="zm.dict.yaml"
	cat "$RIME_TEMPLATE" > "$outfile"
	$MBTOOL -l -I "${ALLFILES[@]/%/.mb}" >> "$outfile"
}
# ============END OF FORMAT-SPECIFIC============

if [[ ! -z $BOOTSTRAP ]]; then
	if [[ -f "${SHORTFILE}.mb" ]]; then
		mv "${SHORTFILE}.mb" "${SHORTFILE}.mb~"
	fi
	$MBGEN -d -short="${SHORTFILE}.mbi" -l3boot="${SHORTFILE}.txt" "${BOOTFILES[@]/%/.txt}" > l3.txt
else
	if [[ ! -f "${SHORTFILE}.mb" ]]; then
		echo "${SHORTFILE}.mb not found.  Please run L3 bootstrap."
		exit 1
	fi
	$MBGEN -d -short="${SHORTFILE}.mb" "${CHARFILES[@]/%/.txt}"
	for wordfile in "${WORDFILES[@]}"; do
		$MBTOOL -w=zm "${CHARFILES[@]/%/.mb}" < "${wordfile/%/.txt}" > "${wordfile/%/.mb}"
	done
	case "$1" in
		fcitx) output_fcitx ;;
		rime) output_rime ;;
		*) echo "no format-specific files were generated"
	esac
fi
