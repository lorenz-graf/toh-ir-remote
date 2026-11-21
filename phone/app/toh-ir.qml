// TOH IR app: remotes for the IR cover. Talks to toh-ird (session D-Bus, started on demand).
// Pure QML (ES5 JavaScript: Qt 5.6), run by sailfish-qml.
import QtQuick 2.0
import Sailfish.Silica 1.0
import Nemo.DBus 2.0

ApplicationWindow {
    id: app
    property string coverError: ""
    property var remotesPage: null

    DBusInterface {
        id: ir
        bus: DBus.SessionBus
        service: "io.github.lorenz_graf.TohIr"
        path: "/io/github/lorenz_graf/TohIr"
        iface: "io.github.lorenz_graf.TohIr"
    }

    function strArgs(list) {   // strings, and whole numbers as int32
        var a = []
        for (var i = 0; i < list.length; i++)
            a.push(typeof list[i] === "number" ? { type: "i", value: list[i] } : { type: "s", value: String(list[i]) })
        return a
    }
    // call(method, [string args], onResult(value), onError(text))
    function call(method, args, ok, err) {
        ir.typedCall(method, strArgs(args || []), function(r) { if (ok) ok(r) }, function(e) {
            var s = String(e)
            var i = s.lastIndexOf(":")
            if (err) err(s)
            else app.toast(s)
        })
    }
    function json(method, args, ok, err) {
        call(method, args, function(r) { ok(JSON.parse(r)) }, err)
    }
    function probe() {
        call("Probe", [], function(r) { app.coverError = r })
    }

    // ---------------- short messages at the bottom ----------------
    function toast(text) { toastLabel.text = text; toastBox.opacity = 1; toastTimer.restart() }
    Rectangle {
        id: toastBox
        z: 100
        anchors { left: parent.left; right: parent.right; bottom: parent.bottom; margins: Theme.paddingLarge }
        height: toastLabel.height + 2 * Theme.paddingMedium
        radius: Theme.paddingSmall
        color: Theme.rgba(Theme.highlightDimmerColor, 0.95)
        opacity: 0
        visible: opacity > 0
        Behavior on opacity { FadeAnimation {} }
        Label {
            id: toastLabel
            anchors.centerIn: parent
            width: parent.width - 2 * Theme.paddingLarge
            wrapMode: Text.Wrap
            horizontalAlignment: Text.AlignHCenter
            color: Theme.highlightColor
        }
        Timer { id: toastTimer; interval: 3000; onTriggered: toastBox.opacity = 0 }
    }

    // ---------------- sending: one button press, the service works in the background ----------------
    property bool sending: false
    property string sendProgress: ""
    property int seqBefore: -1
    function press(id, name, index) {   // index >= 0: only that code of a universal button
        if (sending) return
        call("State", [], function(r) {
            seqBefore = JSON.parse(r).seq
            var m = index >= 0 ? "PressIndex" : "Press"
            var a = index >= 0 ? [id, name, index] : [id, name]
            call(m, a, function() { sending = true; sendProgress = ""; sendPoll.start() })
        })
    }
    Timer {
        id: sendPoll
        interval: 150; repeat: true
        onTriggered: app.call("State", [], function(r) {
            var s = JSON.parse(r)
            app.sendProgress = s.progress
            if (s.busy === "" && s.seq !== app.seqBefore) {
                stop(); app.sending = false
                if (s.error) app.toast(s.error)
            }
        })
    }

    // ---------------- layout: where each known button goes ----------------
    // cell: [key, label, icon]; null = empty space
    readonly property var sectionTop: [
        [["power", "Power", ""], ["mute", "", "image://theme/icon-m-speaker-mute"], ["input", "Input", ""]],
        [["on", "On", ""], null, ["off", "Off", ""]]
    ]
    readonly property var sectionNav: [
        [["back", "", "image://theme/icon-m-back"], ["up", "", "image://theme/icon-m-up"], ["home", "", "image://theme/icon-m-home"]],
        [["left", "", "image://theme/icon-m-left"], ["ok", "OK", ""], ["right", "", "image://theme/icon-m-right"]],
        [["menu", "", "image://theme/icon-m-menu"], ["down", "", "image://theme/icon-m-down"], ["info", "Info", ""]]
    ]
    readonly property var sectionRockers: [
        [["vol_up", "Vol +", ""], ["guide", "Guide", ""], ["ch_up", "Ch +", ""]],
        [["vol_dn", "Vol −", ""], null, ["ch_dn", "Ch −", ""]]
    ]
    readonly property var sectionColours: [
        [["red", "", ""], ["green", "", ""], ["yellow", "", ""], ["blue", "", ""]]
    ]
    readonly property var sectionMedia: [
        [["rew", "", "image://theme/icon-m-media-rewind"], ["play", "", "image://theme/icon-m-play"],
         ["pause", "", "image://theme/icon-m-pause"], ["ffwd", "", "image://theme/icon-m-media-forward"]],
        [["prev", "", "image://theme/icon-m-previous"], ["playpause", "Play/Pause", ""],
         ["stop", "", "image://theme/icon-m-stop"], ["next", "", "image://theme/icon-m-next"]],
        [["rec", "Rec", ""]]
    ]
    readonly property var sectionNumbers: [
        [["n1", "1", ""], ["n2", "2", ""], ["n3", "3", ""]],
        [["n4", "4", ""], ["n5", "5", ""], ["n6", "6", ""]],
        [["n7", "7", ""], ["n8", "8", ""], ["n9", "9", ""]],
        [null, ["n0", "0", ""], null]
    ]
    readonly property var sectionClimate: [
        [["temp_up", "Temp +", ""], ["mode", "Mode", ""], ["fan", "Fan", ""]],
        [["temp_dn", "Temp −", ""], ["swing", "Swing", ""], ["timer", "Timer", ""]],
        [["cool_hi", "Cool hi", ""], ["heat_hi", "Heat hi", ""], ["dry", "Dry", ""]],
        [["cool_lo", "Cool lo", ""], ["heat_lo", "Heat lo", ""], null]
    ]
    function sectionsFor(kind) {
        if (kind === "climate") return [sectionTop, sectionClimate, sectionNav, sectionNumbers]
        if (kind === "audio") return [sectionTop, sectionRockers, sectionMedia, sectionNav, sectionNumbers, sectionColours]
        if (kind === "player") return [sectionTop, sectionNav, sectionMedia, sectionRockers, sectionColours, sectionNumbers]
        if (kind === "light") return [sectionTop, sectionNav, sectionRockers, sectionNumbers]
        return [sectionTop, sectionNav, sectionRockers, sectionColours, sectionMedia, sectionNumbers, sectionClimate]
    }
    readonly property var keyColours: ({ red: "#d03030", green: "#30a040", yellow: "#d8b020", blue: "#3060d0" })

    function kindLabel(k) {
        return ({ tv: "TV", projector: "Projector", audio: "Audio", player: "Player", climate: "Climate",
                  light: "Light", other: "Other" })[k] || "Remote"
    }

    // ---------------- one remote button ----------------
    Component {
        id: remoteButton
        BackgroundItem {
            id: btn
            property var remote          // RemotePage
            property string key: ""
            property string name: ""     // button name in the file; "" = no signal yet
            property string text: ""
            property string icon: ""
            readonly property bool empty: name === ""
            readonly property string learnName: empty ? defaultName(key, text) : name
            height: Theme.itemSizeMedium
            highlighted: down || (app.sending && !!remote && remote.lastPressed === learnName)
            opacity: empty ? 0.45 : 1

            function defaultName(k, t) {
                var m = { vol_up: "Vol_up", vol_dn: "Vol_dn", ch_up: "Ch_next", ch_dn: "Ch_prev", n0: "0", n1: "1",
                          n2: "2", n3: "3", n4: "4", n5: "5", n6: "6", n7: "7", n8: "8", n9: "9",
                          temp_up: "Temp_up", temp_dn: "Temp_dn", cool_hi: "Cool_hi", cool_lo: "Cool_lo",
                          heat_hi: "Heat_hi", heat_lo: "Heat_lo", playpause: "Play_pause", ffwd: "Fast_fwd" }
                if (m[k]) return m[k]
                return k.charAt(0).toUpperCase() + k.slice(1)
            }

            Rectangle {
                anchors.fill: parent
                radius: Theme.paddingMedium
                color: "transparent"
                border.width: 2
                border.color: btn.empty ? Theme.rgba(Theme.secondaryColor, 0.4) : Theme.rgba(Theme.highlightColor, 0.5)
            }
            Rectangle {   // colour keys
                visible: !!app.keyColours[btn.key]
                anchors.centerIn: parent
                width: parent.width * 0.5; height: Theme.paddingMedium * 1.5; radius: height / 2
                color: app.keyColours[btn.key] || "transparent"
            }
            Image {
                anchors.centerIn: parent
                visible: btn.icon !== ""
                source: btn.icon === "" ? "" : btn.icon + "?" + (btn.highlighted ? Theme.highlightColor : Theme.primaryColor)
            }
            Label {
                anchors.centerIn: parent
                width: parent.width - Theme.paddingSmall * 2
                horizontalAlignment: Text.AlignHCenter
                visible: btn.icon === "" && !app.keyColours[btn.key]
                text: btn.text
                truncationMode: TruncationMode.Fade
                font.pixelSize: btn.text.length > 7 ? Theme.fontSizeExtraSmall : Theme.fontSizeSmall
                color: btn.key === "power" ? "#ff5050" : (btn.highlighted ? Theme.highlightColor : Theme.primaryColor)
                font.bold: btn.key === "power"
            }
            onClicked: {
                if (empty) { app.toast("No signal yet: long press to record it"); return }
                remote.lastPressed = name
                if (remote.countFor(name) > 1) remote.stepPress(name)
                else app.press(remote.remoteId, name, -1)
            }
            onPressAndHold: {
                if (empty) pageStack.push(learnPage, { remoteId: remote.remoteId, buttonName: learnName })
                else pageStack.push(buttonPage, { remoteId: remote.remoteId, buttonName: name, info: remote.infoFor(name) })
            }
        }
    }

    // ---------------- pages ----------------
    initialPage: Page {
        id: remotesPage
        Component.onCompleted: app.remotesPage = remotesPage
        property var remotes: []
        function reload() { app.json("Remotes", [], function(r) { remotes = r }) }
        onStatusChanged: if (status === PageStatus.Activating) { reload(); app.probe() }

        SilicaListView {
            id: list
            anchors.fill: parent
            model: remotesPage.remotes
            header: Column {
                width: list.width
                PageHeader { title: "IR Remote" }
                Label {
                    visible: app.coverError !== ""
                    x: Theme.horizontalPageMargin; width: parent.width - 2 * x
                    wrapMode: Text.Wrap
                    color: Theme.secondaryHighlightColor
                    font.pixelSize: Theme.fontSizeSmall
                    text: app.coverError + "\nYou can still add and set up remotes."
                }
                Item { width: 1; height: Theme.paddingLarge; visible: app.coverError !== "" }
            }
            PullDownMenu {
                MenuItem { text: "Check cover"; onClicked: { app.probe(); app.toast("Checking…") } }
                MenuItem { text: "Add remote"; onClicked: pageStack.push(addPage) }
            }
            ViewPlaceholder {
                enabled: list.count === 0
                text: "No remotes yet"
                hintText: "Pull down to add one"
            }
            delegate: ListItem {
                id: item
                contentHeight: Theme.itemSizeMedium
                Column {
                    x: Theme.horizontalPageMargin
                    width: parent.width - 2 * x
                    anchors.verticalCenter: parent.verticalCenter
                    Label { width: parent.width; text: modelData.title; truncationMode: TruncationMode.Fade
                            color: item.highlighted ? Theme.highlightColor : Theme.primaryColor }
                    Label { text: app.kindLabel(modelData.kind) + " · " + modelData.count + " buttons"
                            font.pixelSize: Theme.fontSizeExtraSmall; color: Theme.secondaryColor }
                }
                onClicked: pageStack.push(remotePage, { remoteId: modelData.id })
                menu: ContextMenu {
                    MenuItem { text: "Rename"; onClicked: pageStack.push(renameDialog, { remoteId: modelData.id, title: modelData.title }) }
                    MenuItem { text: "Delete"; onClicked: item.remorseDelete(function() {
                        app.call("Delete", [modelData.id], function() { remotesPage.reload() }) }) }
                }
            }
            VerticalScrollDecorator {}
        }
    }

    function openNewRemote(id) {
        pageStack.pop(app.remotesPage, PageStackAction.Immediate)
        pageStack.push(remotePage, { remoteId: id })
    }

    Component {
        id: addPage
        Page {
            id: page
            property var results: []
            property var categories: []
            property bool adding: false
            Component.onCompleted: app.json("DbCategories", [], function(r) { categories = r })
            Timer { id: searchDelay; interval: 300; onTriggered: app.json("DbSearch", [page.searchField.text], function(r) { page.results = r }) }
            function addPath(p) {
                adding = true
                app.call("AddFromDb", [p], function(id) { adding = false; app.openNewRemote(id) },
                         function(e) { adding = false; app.toast(e) })
            }
            BusyIndicator { anchors.centerIn: parent; size: BusyIndicatorSize.Large; running: page.adding; z: 10 }

            SilicaListView {
                id: lv
                anchors.fill: parent
                enabled: !page.adding
                opacity: page.adding ? 0.3 : 1
                header: Column {
                    width: lv.width
                    PageHeader { title: "Add remote" }
                    SearchField {
                        id: search
                        width: parent.width
                        placeholderText: "Search brand or model"
                        onTextChanged: { if (text.length > 1) searchDelay.restart(); else page.results = [] }
                        Component.onCompleted: page.searchField = search
                    }
                    Column {
                        width: parent.width
                        visible: search.text.length < 2
                        BackgroundItem {
                            width: parent.width
                            Label { x: Theme.horizontalPageMargin; anchors.verticalCenter: parent.verticalCenter
                                    text: "Universal remotes"; color: parent.highlighted ? Theme.highlightColor : Theme.primaryColor }
                            onClicked: pageStack.push(universalPage)
                        }
                        BackgroundItem {
                            width: parent.width
                            Label { x: Theme.horizontalPageMargin; anchors.verticalCenter: parent.verticalCenter
                                    text: "Blank remote (record every button)"; color: parent.highlighted ? Theme.highlightColor : Theme.primaryColor }
                            onClicked: pageStack.push(blankDialog)
                        }
                        SectionHeader { text: "Database" }
                    }
                }
                model: page.searchField && page.searchField.text.length >= 2 ? page.results : page.categories
                delegate: BackgroundItem {
                    width: lv.width
                    Label {
                        x: Theme.horizontalPageMargin; width: parent.width - 2 * x
                        anchors.verticalCenter: parent.verticalCenter
                        text: modelData.label + (modelData.count ? "  (" + modelData.count + ")" : "")
                        truncationMode: TruncationMode.Fade
                        color: parent.highlighted ? Theme.highlightColor : Theme.primaryColor
                    }
                    onClicked: {
                        if (modelData.path) page.addPath(modelData.path)
                        else pageStack.push(brandsPage, { category: modelData.name, title: modelData.label })
                    }
                }
                VerticalScrollDecorator {}
            }
            property var searchField
        }
    }

    Component {
        id: brandsPage
        Page {
            id: page
            property string category
            property string title
            property var brands: []
            Component.onCompleted: app.json("DbBrands", [category], function(r) { brands = r })
            SilicaListView {
                anchors.fill: parent
                header: PageHeader { title: page.title }
                model: page.brands
                delegate: BackgroundItem {
                    Label {
                        x: Theme.horizontalPageMargin; width: parent.width - 2 * x
                        anchors.verticalCenter: parent.verticalCenter
                        text: modelData.label + "  (" + modelData.count + ")"
                        color: parent.highlighted ? Theme.highlightColor : Theme.primaryColor
                    }
                    onClicked: pageStack.push(modelsPage, { category: page.category, brand: modelData.name, title: modelData.label })
                }
                VerticalScrollDecorator {}
            }
        }
    }

    Component {
        id: modelsPage
        Page {
            id: page
            property string category
            property string brand
            property string title
            property var models: []
            property bool adding: false
            Component.onCompleted: app.json("DbModels", [category, brand], function(r) { models = r })
            BusyIndicator { anchors.centerIn: parent; size: BusyIndicatorSize.Large; running: page.adding; z: 10 }
            SilicaListView {
                anchors.fill: parent
                enabled: !page.adding
                opacity: page.adding ? 0.3 : 1
                header: PageHeader { title: page.title; description: "Tap a model to add it" }
                model: page.models
                delegate: BackgroundItem {
                    Label {
                        x: Theme.horizontalPageMargin; width: parent.width - 2 * x
                        anchors.verticalCenter: parent.verticalCenter
                        text: modelData.label
                        truncationMode: TruncationMode.Fade
                        color: parent.highlighted ? Theme.highlightColor : Theme.primaryColor
                    }
                    onClicked: {
                        page.adding = true
                        app.call("AddFromDb", [modelData.path], function(id) { page.adding = false; app.openNewRemote(id) },
                                 function(e) { page.adding = false; app.toast(e) })
                    }
                }
                VerticalScrollDecorator {}
            }
        }
    }

    Component {
        id: universalPage
        Page {
            id: page
            property bool adding: false
            BusyIndicator { anchors.centerIn: parent; size: BusyIndicatorSize.Large; running: page.adding; z: 10 }
            SilicaListView {
                anchors.fill: parent
                enabled: !page.adding
                header: Column {
                    width: parent.width
                    PageHeader { title: "Universal remotes" }
                    Label {
                        x: Theme.horizontalPageMargin; width: parent.width - 2 * x
                        wrapMode: Text.Wrap; font.pixelSize: Theme.fontSizeSmall; color: Theme.secondaryHighlightColor
                        text: "Each button sends the codes of many brands, one after another, so a press takes a while. " +
                              "From the Flipper Zero project, downloaded when you add it."
                    }
                    Item { width: 1; height: Theme.paddingLarge }
                }
                model: [["tv", "TV"], ["projector", "Projector"], ["audio", "Audio / soundbar"], ["ac", "Air conditioner"]]
                delegate: BackgroundItem {
                    Label { x: Theme.horizontalPageMargin; anchors.verticalCenter: parent.verticalCenter; text: modelData[1]
                            color: parent.highlighted ? Theme.highlightColor : Theme.primaryColor }
                    onClicked: {
                        page.adding = true
                        app.call("AddUniversal", [modelData[0]], function(id) { page.adding = false; app.openNewRemote(id) },
                                 function(e) { page.adding = false; app.toast(e) })
                    }
                }
            }
        }
    }

    Component {
        id: blankDialog
        Dialog {
            id: dlg
            property string kind: "tv"
            canAccept: nameField.text.trim() !== ""
            Column {
                width: parent.width
                DialogHeader { title: "Blank remote" }
                TextField { id: nameField; width: parent.width; label: "Name"; placeholderText: "Name"; focus: true }
                ComboBox {
                    label: "Layout"
                    menu: ContextMenu {
                        MenuItem { text: "TV / projector"; onClicked: dlg.kind = "tv" }
                        MenuItem { text: "Audio"; onClicked: dlg.kind = "audio" }
                        MenuItem { text: "Player"; onClicked: dlg.kind = "player" }
                        MenuItem { text: "Air conditioner / fan"; onClicked: dlg.kind = "climate" }
                    }
                }
            }
            onAccepted: app.call("AddBlank", [nameField.text.trim(), kind], function(id) { app.openNewRemote(id) })
        }
    }

    Component {
        id: renameDialog
        Dialog {
            property string remoteId
            property string title
            canAccept: f.text.trim() !== ""
            Column {
                width: parent.width
                DialogHeader { title: "Rename" }
                TextField { id: f; width: parent.width; label: "Name"; text: title; focus: true }
            }
            onAccepted: app.call("Rename", [remoteId, f.text.trim()], function() { if (app.remotesPage) app.remotesPage.reload() })
        }
    }

    Component {
        id: addButtonDialog
        Dialog {
            property string remoteId
            canAccept: f.text.trim() !== ""
            Column {
                width: parent.width
                DialogHeader { title: "Add button"; acceptText: "Record" }
                TextField { id: f; width: parent.width; label: "Button name"; placeholderText: "Button name"; focus: true }
            }
            acceptDestination: learnPage
            acceptDestinationAction: PageStackAction.Replace
            acceptDestinationProperties: ({ remoteId: remoteId, buttonName: f.text.trim() })
        }
    }

    Component {
        id: remotePage
        Page {
            id: page
            property string remoteId
            property var remote: ({ title: "", kind: "", buttons: [] })
            property var byKey: ({})
            property var extras: []
            property bool showEmpty: false
            property string lastPressed: ""
            // universal buttons: each tap sends the next code, "Worked" keeps it
            property string stepName: ""
            property int stepIndex: -1
            property int stepCount: 0
            function countFor(name) {
                for (var i = 0; i < remote.buttons.length; i++)
                    if (remote.buttons[i].name === name) return remote.buttons[i].count
                return 0
            }
            function stepPress(name) {
                if (app.sending) return
                if (stepName !== name) { stepName = name; stepIndex = -1; stepCount = countFor(name) }
                stepIndex = (stepIndex + 1) % stepCount
                app.press(remoteId, name, stepIndex)
            }
            function stepSend(i) {
                if (app.sending) return
                stepIndex = (i + stepCount) % stepCount
                app.press(remoteId, stepName, stepIndex)
            }
            function infoFor(name) {
                for (var i = 0; i < remote.buttons.length; i++)
                    if (remote.buttons[i].name === name) {
                        var b = remote.buttons[i]
                        return b.info + (b.count > 1 ? " · " + b.count + " codes" : "")
                    }
                return ""
            }
            function reload() {
                app.json("Remote", [remoteId], function(r) {
                    var m = {}, ex = []
                    for (var i = 0; i < r.buttons.length; i++) {
                        var b = r.buttons[i]
                        if (b.key) m[b.key] = b.name
                        else ex.push(b.name)
                    }
                    byKey = m; extras = ex; remote = r
                    if (r.buttons.length === 0) showEmpty = true
                })
            }
            onStatusChanged: if (status === PageStatus.Activating) reload()

            SilicaFlickable {
                anchors.fill: parent
                contentHeight: col.height + Theme.paddingLarge * 2

                PullDownMenu {
                    MenuItem { text: "Rename"; onClicked: pageStack.push(renameDialog, { remoteId: page.remoteId, title: page.remote.title }) }
                    MenuItem { text: "Add button"; onClicked: pageStack.push(addButtonDialog, { remoteId: page.remoteId }) }
                    MenuItem { text: page.showEmpty ? "Hide empty buttons" : "Show empty buttons"; onClicked: page.showEmpty = !page.showEmpty }
                }

                Column {
                    id: col
                    width: parent.width
                    spacing: Theme.paddingLarge

                    PageHeader {
                        title: page.remote.title
                        description: page.remote.source ? page.remote.source : app.kindLabel(page.remote.kind)
                    }
                    Item {   // sending state (universal remotes take long)
                        width: parent.width
                        height: app.sending && app.sendProgress !== "" ? stopRow.height : 0
                        visible: height > 0
                        Row {
                            id: stopRow
                            x: Theme.horizontalPageMargin
                            spacing: Theme.paddingLarge
                            Label { anchors.verticalCenter: parent.verticalCenter; text: "Sending " + app.sendProgress
                                    color: Theme.highlightColor }
                            Button { text: "Stop"; onClicked: app.call("Cancel", []) }
                        }
                    }
                    Label {
                        visible: app.coverError !== ""
                        x: Theme.horizontalPageMargin; width: parent.width - 2 * x
                        wrapMode: Text.Wrap; font.pixelSize: Theme.fontSizeExtraSmall; color: Theme.secondaryHighlightColor
                        text: app.coverError
                    }
                    Column {   // finding the right code of a universal button
                        visible: page.stepName !== ""
                        x: Theme.horizontalPageMargin; width: parent.width - 2 * x
                        spacing: Theme.paddingSmall
                        Label {
                            width: parent.width; wrapMode: Text.Wrap; color: Theme.highlightColor
                            text: page.stepName.replace(/_/g, " ") + ": code " + (page.stepIndex + 1) + " of " + page.stepCount + " sent"
                        }
                        Label {
                            width: parent.width; wrapMode: Text.Wrap
                            font.pixelSize: Theme.fontSizeExtraSmall; color: Theme.secondaryHighlightColor
                            text: "Tap Next until the device reacts, then Worked to keep only that code."
                        }
                        Row {
                            spacing: Theme.paddingMedium
                            Button { text: "Back"; preferredWidth: Theme.buttonWidthExtraSmall; onClicked: page.stepSend(page.stepIndex - 1) }
                            Button { text: "Next"; preferredWidth: Theme.buttonWidthExtraSmall; onClicked: page.stepSend(page.stepIndex + 1) }
                            Button {
                                text: "Worked"; preferredWidth: Theme.buttonWidthExtraSmall
                                onClicked: app.call("Keep", [page.remoteId, page.stepName, page.stepIndex], function() {
                                    app.toast("Kept code " + (page.stepIndex + 1) + " for " + page.stepName.replace(/_/g, " "))
                                    page.stepName = ""; page.reload()
                                })
                            }
                            Button { text: "×"; preferredWidth: Theme.buttonWidthExtraSmall / 2; onClicked: page.stepName = "" }
                        }
                    }
                    Label {
                        visible: page.showEmpty
                        x: Theme.horizontalPageMargin; width: parent.width - 2 * x
                        wrapMode: Text.Wrap; font.pixelSize: Theme.fontSizeExtraSmall; color: Theme.secondaryColor
                        text: "Long press a button to record its signal from the original remote."
                    }

                    Repeater {   // sections
                        model: app.sectionsFor(page.remote.kind)
                        delegate: Column {
                            id: section
                            property var rows: modelData
                            width: col.width
                            spacing: Theme.paddingMedium
                            Repeater {
                                model: section.rows
                                delegate: Item {
                                    id: rowItem
                                    property var cells: modelData
                                    property bool any: {
                                        for (var i = 0; i < cells.length; i++)
                                            if (cells[i] && (page.showEmpty || page.byKey[cells[i][0]])) return true
                                        return false
                                    }
                                    width: col.width
                                    height: any ? Theme.itemSizeMedium : 0
                                    visible: any
                                    Row {
                                        x: Theme.horizontalPageMargin
                                        spacing: Theme.paddingMedium
                                        Repeater {
                                            model: rowItem.cells
                                            delegate: Item {
                                                property var c: modelData
                                                width: (col.width - 2 * Theme.horizontalPageMargin - (rowItem.cells.length - 1) * Theme.paddingMedium) / rowItem.cells.length
                                                height: Theme.itemSizeMedium
                                                Loader {
                                                    anchors.fill: parent
                                                    active: !!c && (page.showEmpty || !!page.byKey[c[0]])
                                                    sourceComponent: remoteButton
                                                    onLoaded: {
                                                        item.remote = page
                                                        item.key = c[0]
                                                        item.text = c[1]
                                                        item.icon = c[2]
                                                        item.name = Qt.binding(function() { return page.byKey[c[0]] || "" })
                                                    }
                                                }
                                            }
                                        }
                                    }
                                }
                            }
                        }
                    }

                    SectionHeader { text: "More buttons"; visible: page.extras.length > 0 }
                    Flow {
                        x: Theme.horizontalPageMargin
                        width: parent.width - 2 * x
                        spacing: Theme.paddingMedium
                        Repeater {
                            model: page.extras
                            delegate: Loader {
                                width: (col.width - 2 * Theme.horizontalPageMargin - 2 * Theme.paddingMedium) / 3
                                height: Theme.itemSizeMedium
                                sourceComponent: remoteButton
                                onLoaded: {
                                    item.remote = page
                                    item.name = modelData
                                    item.text = modelData.replace(/_/g, " ")
                                }
                            }
                        }
                    }
                }
                VerticalScrollDecorator {}
            }
        }
    }

    Component {
        id: buttonPage
        Page {
            id: page
            property string remoteId
            property string buttonName
            property string info
            Column {
                width: parent.width
                spacing: Theme.paddingLarge
                PageHeader { title: page.buttonName; description: page.info }
                Button {
                    anchors.horizontalCenter: parent.horizontalCenter
                    text: "Record new signal"
                    onClicked: pageStack.replace(learnPage, { remoteId: page.remoteId, buttonName: page.buttonName })
                }
                Button {
                    anchors.horizontalCenter: parent.horizontalCenter
                    text: "Delete button"
                    onClicked: app.call("DeleteButton", [page.remoteId, page.buttonName], function() { pageStack.pop() })
                }
            }
        }
    }

    Component {
        id: learnPage
        Page {
            id: page
            property string remoteId
            property string buttonName
            property string phase: "wait"   // wait, done, error
            property string message: ""
            property int seqBefore: -1
            backNavigation: true
            function begin() {
                phase = "wait"; message = ""
                app.call("State", [], function(r) {
                    seqBefore = JSON.parse(r).seq
                    app.call("Learn", [remoteId, buttonName], function() { poll.start() },
                             function(e) { phase = "error"; message = e })
                })
            }
            onStatusChanged: {
                if (status === PageStatus.Active && seqBefore === -1 && buttonName !== "") begin()
                if (status === PageStatus.Deactivating && phase === "wait") { poll.stop(); app.call("Cancel", []) }
            }
            onButtonNameChanged: if (status === PageStatus.Active && seqBefore === -1) begin()
            Timer {
                id: poll
                interval: 200; repeat: true
                onTriggered: app.call("State", [], function(r) {
                    var s = JSON.parse(r)
                    if (s.busy === "" && s.seq !== page.seqBefore) {
                        poll.stop()
                        if (s.error) { page.phase = "error"; page.message = s.error }
                        else if (s.result === "cancelled") { }
                        else { page.phase = "done"; page.message = s.result; closeTimer.start() }
                    }
                })
            }
            Timer { id: closeTimer; interval: 1200; onTriggered: pageStack.pop() }

            Column {
                width: parent.width
                spacing: Theme.paddingLarge * 2
                PageHeader { title: "Record " + page.buttonName }
                Label {
                    x: Theme.horizontalPageMargin; width: parent.width - 2 * x
                    wrapMode: Text.Wrap
                    horizontalAlignment: Text.AlignHCenter
                    font.pixelSize: Theme.fontSizeLarge
                    color: Theme.highlightColor
                    text: page.phase === "wait" ? "Point the original remote at the back of the phone and press " + page.buttonName + " once"
                        : page.phase === "done" ? "Saved" : "Didn't work"
                }
                BusyIndicator { anchors.horizontalCenter: parent.horizontalCenter; size: BusyIndicatorSize.Large; running: page.phase === "wait" }
                Label {
                    x: Theme.horizontalPageMargin; width: parent.width - 2 * x
                    wrapMode: Text.Wrap; horizontalAlignment: Text.AlignHCenter
                    color: Theme.secondaryHighlightColor
                    text: page.message
                }
                Button {
                    visible: page.phase === "error"
                    anchors.horizontalCenter: parent.horizontalCenter
                    text: "Try again"
                    onClicked: page.begin()
                }
            }
        }
    }

    cover: CoverBackground {
        Label {
            anchors.centerIn: parent
            width: parent.width - 2 * Theme.paddingLarge
            horizontalAlignment: Text.AlignHCenter
            wrapMode: Text.Wrap
            text: app.sending ? "Sending…" : "IR Remote"
        }
    }
}
