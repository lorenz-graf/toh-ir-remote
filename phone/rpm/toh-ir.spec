# TOH IR phone package: toh-ird (session D-Bus service + test CLI), the IR Remote app (pure QML), the bundled
# Flipper-IRDB index and the symbiosis config for the TOH DevKit. Built by ../build-rpm.sh from a prebuilt static
# Go binary.
%global debug_package %{nil}
%global _build_id_links none
%global __os_install_post %{nil}

Name:       toh-ir
Version:    %{tir_version}
Release:    2
Summary:    IR Remote: universal remote for the TOH IR cover
License:    BSD-3-Clause
Requires:   symbiosis
Requires:   toh-devkit
Requires:   libsailfishapp-launcher
Requires:   nemo-qml-plugin-dbus-qt5

%description
Phone side of the TOH IR cover (prototype on the TOH DevKit):
- IR Remote app: remotes from the Flipper-IRDB database, universal remotes, learning buttons from a remote
- toh-ird: session D-Bus service io.github.lorenz_graf.TohIr (started on demand)
- symbiosis config that allows toh-ird on the DevKit's bus

%prep

%build

%install
S=%{_sourcedir}
install -D -m 0755 $S/toh-ird %{buildroot}/usr/libexec/toh-ir/toh-ird
install -D -m 0644 $S/devkit-ir.yaml %{buildroot}/usr/share/tohd-1/tohs/0000/0000/devkit-ir.yaml
install -D -m 0644 $S/io.github.lorenz_graf.TohIr.service %{buildroot}/usr/share/dbus-1/services/io.github.lorenz_graf.TohIr.service
install -D -m 0644 $S/toh-ir.qml %{buildroot}/usr/share/toh-ir/qml/toh-ir.qml
install -D -m 0644 $S/irdb-index.json %{buildroot}/usr/share/toh-ir/irdb-index.json
install -D -m 0644 $S/toh-ir.desktop %{buildroot}/usr/share/applications/toh-ir.desktop
for s in 86 108 128 172; do
  install -D -m 0644 $S/icons/${s}x${s}/toh-ir.png %{buildroot}/usr/share/icons/hicolor/${s}x${s}/apps/toh-ir.png
done

%post
# A toh-ird still running from the previous version keeps its deleted binary, and symbiosis refuses deleted
# executables. Stop it; D-Bus activation starts the new one on the next call.
p=$(pidof toh-ird) && kill $p || :
# symbiosis reads the config on attach: take the cover off and put it back on after the first install

%preun
if [ "$1" = 0 ]; then
  p=$(pidof toh-ird) && kill $p || :
fi

%files
%dir /usr/libexec/toh-ir
/usr/libexec/toh-ir/toh-ird
/usr/share/tohd-1/tohs/0000/0000/devkit-ir.yaml
/usr/share/dbus-1/services/io.github.lorenz_graf.TohIr.service
%dir /usr/share/toh-ir
%dir /usr/share/toh-ir/qml
/usr/share/toh-ir/qml/toh-ir.qml
/usr/share/toh-ir/irdb-index.json
/usr/share/applications/toh-ir.desktop
/usr/share/icons/hicolor/86x86/apps/toh-ir.png
/usr/share/icons/hicolor/108x108/apps/toh-ir.png
/usr/share/icons/hicolor/128x128/apps/toh-ir.png
/usr/share/icons/hicolor/172x172/apps/toh-ir.png
