#!/bin/sh
# Builds ../build/toh-ir-<version>-<release>.aarch64.rpm (Go binary cross-compiled, app is pure QML).
# Needs go, python3 + Pillow (icon), rpmbuild (Arch: pacman -S rpm-tools), gh (GitHub CLI, for the database index).
# Sailfish has rpm 4.16, so the package is written in rpm v4 format with a gzip payload.
set -eu
cd "$(dirname "$0")"
VERSION=0.1.0
IRDB_REPO=UberGuidoZ/Flipper-IRDB
IRDB_COMMIT=c33f37e6f75836b5d1894a95c04778d93bae51e7   # pinned: downloads use this commit, so paths stay valid

OUT=$(cd .. && pwd)/build
TOP=$OUT/rpm
rm -rf "$TOP" && mkdir -p "$TOP/SOURCES"
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o "$TOP/SOURCES/toh-ird" ./cmd/toh-ird
python3 app/make_icon.py "$TOP/SOURCES/icons" >/dev/null
cp devkit-ir.yaml app/toh-ir.qml app/toh-ir.desktop app/io.github.lorenz_graf.TohIr.service "$TOP/SOURCES/"

# database index: every .ir file of Flipper-IRDB (CC0) except the auto-converted folder
INDEX="$OUT/irdb-index-$IRDB_COMMIT.json"
if [ ! -s "$INDEX" ]; then
  gh api "repos/$IRDB_REPO/git/trees/$IRDB_COMMIT?recursive=1" | python3 -c '
import json, sys
repo, commit = sys.argv[1], sys.argv[2]
t = json.load(sys.stdin)
assert not t["truncated"]
files = sorted(e["path"] for e in t["tree"] if e["type"] == "blob" and e["path"].endswith(".ir")
               and "/" in e["path"] and not e["path"].startswith(("_", ".")))
json.dump({"repo": repo, "commit": commit, "files": files}, sys.stdout, separators=(",", ":"))
' "$IRDB_REPO" "$IRDB_COMMIT" > "$INDEX"
fi
cp "$INDEX" "$TOP/SOURCES/irdb-index.json"

rpmbuild -bb --quiet --target aarch64-unknown-linux \
    --define "_topdir $TOP" \
    --define "tir_version $VERSION" \
    --define "_binary_payload w9.gzdio" \
    --define "_rpmformat 4" \
    rpm/toh-ir.spec
rm -f "$OUT"/toh-ir-*.rpm
mv "$TOP"/RPMS/aarch64/*.rpm "$OUT/"
rm -rf "$TOP"
echo "built: $(ls "$OUT"/toh-ir-*.rpm)"
