import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import org.kde.plasma.plasmoid
import org.kde.kirigami as Kirigami

PlasmoidItem {
    id: root

    switchWidth: Kirigami.Units.gridUnit * 18
    switchHeight: Kirigami.Units.gridUnit * 22
    toolTipSubText: errorMsg ? "Error: " + errorMsg : (lastRefresh > new Date(0) ? "Refreshed " + agoText(lastRefresh) : "Loading...")

    // ── Data ─────────────────────────────────────────────────
    property var rawProviders: []
    property var groupedProviders: []
    property string errorMsg: ""
    property bool loading: false
    property date lastRefresh: new Date(0)
    property double worstPct: -1
    property var worstUnit: null
    property string compactText: "--"
    property string compactColor: "#7f8c8d"
    property var widgetConfig: ({})
    property string pinnedProvider: ""
    property var hiddenProviders: ([])
    property bool configLoading: false
    property bool hiddenProvidersCollapsed: false


    // ── Timer ────────────────────────────────────────────────
    Timer {
        id: pollTimer
        interval: 30000
        running: false
        repeat: true
        onTriggered: fetchStatus()
    }

    Timer {
        id: refreshTimer
        interval: 2000
        running: false
        repeat: false
        onTriggered: fetchStatus()
    }

    Component.onCompleted: {
        fetchConfig()
        fetchStatus()
        pollTimer.running = true
    }


    // ── Network ──────────────────────────────────────────────
    function fetchStatus() {
        loading = true
        var xhr = new XMLHttpRequest()
        xhr.open("GET", "http://127.0.0.1:7731/status")
        xhr.timeout = 5000
        xhr.onreadystatechange = function () {
            if (xhr.readyState === XMLHttpRequest.DONE) {
                loading = false
                if (xhr.status === 200) {
                    try {
                        root.rawProviders = JSON.parse(xhr.responseText)
                        root.errorMsg = ""
                        root.lastRefresh = new Date()
                    } catch (e) {
                        root.errorMsg = "Parse error"
                        root.rawProviders = []
                        root.lastRefresh = new Date()
                    }
                } else {
                    root.errorMsg = "Daemon unreachable"
                    root.rawProviders = []
                }
                processData()
            }
        }
        xhr.onerror = function () {
            loading = false
            root.errorMsg = "Cannot connect"
            root.rawProviders = []
            processData()
        }
        xhr.send()
    }

    function fetchConfig() {
        configLoading = true
        var xhr = new XMLHttpRequest()
        xhr.open("GET", "http://127.0.0.1:7731/api/config")
        xhr.timeout = 5000
        xhr.onreadystatechange = function () {
            if (xhr.readyState === XMLHttpRequest.DONE) {
                configLoading = false
                if (xhr.status === 200) {
                    try {
                        var resp = JSON.parse(xhr.responseText)
                        root.widgetConfig = resp.widget || {}
                        root.pinnedProvider = resp.widget ? (resp.widget.pinned || "") : ""
                        root.hiddenProviders = resp.widget ? (resp.widget.hidden || []) : []
                        // Re-process to apply hide filter locally
                        processData()
                    } catch (e) {}
                }
            }
        }
        xhr.onerror = function () { configLoading = false }
        xhr.send()
    }

    function apiConfig(query) {
        var xhr = new XMLHttpRequest()
        xhr.open("GET", "http://127.0.0.1:7731/api/config?" + query)
        xhr.timeout = 5000
        xhr.onreadystatechange = function () {
            if (xhr.readyState === XMLHttpRequest.DONE && xhr.status === 200) {
                setTimeout(function() {
                    fetchConfig()
                    fetchStatus()
                }, 1000)
            }
        }
        xhr.send()
    }

    // ── Data processing ──────────────────────────────────────
    function processData() {
        // Strip any previously injected synthetic "opencode" entries
        rawProviders = rawProviders.filter(function(p) { return p.provider !== "opencode" })
        var standalone = []
        var subs = []

        for (var i = 0; i < rawProviders.length; i++) {
            var p = rawProviders[i]
            normalizeWindows(p)
            if (p.provider.indexOf("opencode/") === 0) {
                // Skip sub-providers that are hidden individually or via group parent
                if (root.hiddenProviders.indexOf(p.provider) >= 0) continue
                if (root.hiddenProviders.indexOf("opencode") >= 0) continue
                subs.push(p)
            } else {
                // Skip standalone providers that are hidden
                if (root.hiddenProviders.indexOf(p.provider) >= 0) continue
                standalone.push(p)
            }
        }

        // Promote single sub to standalone — no group wrapper needed
        if (subs.length === 1) {
            subs[0].label = "OpenCode"
            standalone.push(subs[0])
            subs = []
        }

        var groups = []
        for (var j = 0; j < standalone.length; j++) {
            groups.push({
                isGroup: false,
                provider: standalone[j]
            })
        }
        if (subs.length > 0) {
            groups.push({
                isGroup: true,
                provider: makeGroupSummary(subs),
                subs: subs
            })
            // Add synthetic opencode entry to rawProviders for pinning support
            var syn = makeGroupSummary(subs)
            syn.windows = []
            for (var si = 0; si < subs.length; si++) {
                var subWins = subs[si].windows
                if (subWins) {
                    for (var wi = 0; wi < subWins.length; wi++) {
                        syn.windows.push(subWins[wi])
                    }
                }
            }
            rawProviders.push(syn)
        }

        groupedProviders = groups
        computeWorst()
    }

    function makeGroupSummary(subs) {
        var totalCost = 0, hasErr = false, errMsg = ""
        for (var i = 0; i < subs.length; i++) {
            if (subs[i].status === "error") {
                hasErr = true
                if (errMsg) errMsg += "; "
                errMsg += shortSubLabel(subs[i]) + ": " + subs[i].error
            }
            if (subs[i].unit === "usd") totalCost += subs[i].used
        }
        return {
            provider: "opencode", label: "OpenCode",
            used: totalCost, limit: 0, unit: "usd",
            status: hasErr ? "error" : "ok", error: errMsg,
            refreshed_at: subs.length > 0 ? subs[0].refreshed_at : ""
        }
    }

    function computeWorst() {
        worstPct = -1
        worstUnit = null
        var highest = -1, info = null

        var providersToCheck = rawProviders
        if (root.pinnedProvider) {
            var pinned = []
            for (var pi = 0; pi < rawProviders.length; pi++) {
                if (rawProviders[pi].provider === root.pinnedProvider) {
                    pinned.push(rawProviders[pi])
                    break
                }
            }
            providersToCheck = pinned
        }

        for (var i = 0; i < providersToCheck.length; i++) {
            var u = providersToCheck[i]
            if (u.status !== "ok") continue

            if (root.pinnedProvider) {
                // When pinned: prefer monthly window, fall back to largest time window (monthly > weekly > 5h)
                var targetWin = null
                var ws = u.windows
                if (ws) {
                    // First: look for a monthly window with a valid limit
                    for (var j = 0; j < ws.length; j++) {
                        var key = ws[j].key || ws[j].label
                        if (key === "monthly" && ws[j].limit > 0) {
                            targetWin = ws[j]
                            break
                        }
                    }
                    // No monthly with limit? Fall back to largest window by rank
                    if (!targetWin) {
                        var largestRank = -1
                        for (var j = 0; j < ws.length; j++) {
                            var rank = windowRank(ws[j].key || ws[j].label)
                            if (rank > largestRank && ws[j].limit > 0) {
                                largestRank = rank
                                targetWin = ws[j]
                            }
                        }
                    }
                }
                if (targetWin) {
                    var wp = (targetWin.used / targetWin.limit) * 100
                    if (wp > highest) { highest = wp; info = { pct: wp, unit: targetWin.unit, used: targetWin.used } }
                } else if (u.limit > 0) {
                    var pct = (u.used / u.limit) * 100
                    if (pct > highest) { highest = pct; info = { pct: pct, unit: u.unit, used: u.used } }
                } else if (u.used > 0) {
                    info = { pct: -1, unit: u.unit, used: u.used }
                }
            } else {
                // Not pinned: show highest percentage across all windows
                var pct = u.limit > 0 ? (u.used / u.limit) * 100 : -1
                if (pct > highest) { highest = pct; info = { pct: pct, unit: u.unit, used: u.used } }
                var ws = u.windows
                if (ws) {
                    for (var j = 0; j < ws.length; j++) {
                        var wp = ws[j].limit > 0 ? (ws[j].used / ws[j].limit) * 100 : -1
                        if (wp > highest) { highest = wp; info = { pct: wp, unit: ws[j].unit, used: ws[j].used } }
                    }
                }
            }
        }

        var pinnedError = false
        if (root.pinnedProvider) {
            for (var pe = 0; pe < rawProviders.length; pe++) {
                if (rawProviders[pe].provider === root.pinnedProvider && rawProviders[pe].status === "error") {
                    pinnedError = true
                    break
                }
            }
        }

        if (errorMsg && rawProviders.length === 0) {
            compactText = "ERR"; compactColor = "#7f8c8d"
        } else if (pinnedError) {
            compactText = "ERR"; compactColor = "#e74c3c"
        } else if (providersToCheck.length === 0) {
            compactText = "--"; compactColor = Kirigami.Theme.disabledTextColor
        } else if (highest >= 0) {
            worstPct = highest
            compactText = Math.round(highest) + "%"
            compactColor = leftPctColor(100 - highest)
        } else if (info && info.used > 0) {
            compactText = formatCompact(info.used, info.unit)
            compactColor = Kirigami.Theme.textColor
        } else {
            compactText = "OK"
            compactColor = "#27ae60"
        }
    }

    function windowRank(key) {
        var k = (key || "").toLowerCase()
        if (k === "monthly" || k === "month" || k === "30d") return 3
        if (k === "weekly" || k === "week" || k === "7d") return 2
        if (k === "5h" || k === "five_hour" || k === "session") return 1
        return 0
    }

    // ── Helpers ──────────────────────────────────────────────
    function pctColor(pct) {
        if (pct > 100) return "#e74c3c"
        if (pct >= 80) return "#e67e22"
        if (pct >= 50) return "#f39c12"
        return "#27ae60"
    }

    function pctColorDim(pct) {
        if (pct > 100) return "#33e74c3c"
        if (pct >= 80) return "#33e67e22"
        if (pct >= 50) return "#33f39c12"
        return "#3327ae60"
    }

    function formatCompact(value, unit) {
        if (unit === "usd") return "$" + (value % 1 === 0 ? value.toFixed(0) : value.toFixed(2))
        if (value >= 1000) return (value / 1000).toFixed(1).replace(/\.0$/, "") + "k"
        return value % 1 === 0 ? value.toFixed(0) : value.toFixed(1)
    }

    function formatUsed(value, unit) {
        if (unit === "usd") return "$" + (value % 1 === 0 ? value.toFixed(0) : value.toFixed(2))
        if (unit === "messages") return Math.round(value) + " msg"
        if (unit === "tokens") {
            if (value >= 1000000) return (value / 1000000).toFixed(1) + "M tok"
            if (value >= 1000) return Math.round(value / 1000) + "K tok"
            return Math.round(value) + " tok"
        }
        if (unit === "percent") return Math.round(value) + "%"
        return value.toFixed(2) + " " + unit
    }

    function subLabel(p) {
        var label = p.label
        if (!label || label === p.provider) {
            var parts = p.provider.split("/")
            var result = parts.length > 1 ? parts[parts.length - 1] : p.provider
            return result === "opencode" ? "OpenCode" : result
        }
        var parts = label.split("/")
        var result = parts.length > 1 ? parts[parts.length - 1] : label
        return result === "opencode" ? "OpenCode" : result
    }

    function shortSubLabel(p) {
        return subLabel(p)
    }

    function agoText(d) {
        var secs = Math.floor((new Date() - d) / 1000)
        if (secs < 5) return "just now"
        if (secs < 60) return secs + "s ago"
        if (secs < 3600) return Math.floor(secs / 60) + "m ago"
        return Math.floor(secs / 3600) + "h ago"
    }

    function shortWinLabel(key) {
        if (key === "monthly") return "mo"
        if (key === "weekly") return "wk"
        if (key === "5h") return "5h"
        if (key === "daily") return "dy"
        return key.substring(0, 3)
    }

    function normalizeWindows(p) {
        var existing = {}
        if (p.windows) {
            for (var i = 0; i < p.windows.length; i++) {
                existing[p.windows[i].key] = p.windows[i]
            }
        }
        var winDefs = [
            { key: "5h", label: "5h" },
            { key: "weekly", label: "weekly" },
            { key: "monthly", label: "monthly" }
        ]
        var result = []
        for (var j = 0; j < winDefs.length; j++) {
            var def = winDefs[j]
            if (existing[def.key]) {
                result.push(existing[def.key])
            } else {
                result.push({
                    key: def.key,
                    label: def.label,
                    used: 0,
                    limit: 0,
                    unit: p.unit || ""
                })
            }
        }
        p.windows = result
    }

    // ── New display helpers (CodexBar-style) ─────────────────
    function pctLeft(used, limit) {
        if (limit <= 0) return -1
        return 100 - (used / limit) * 100
    }

    function leftPctColor(left) {
        if (left < 0) return "#7f8c8d"
        if (left >= 80) return "#27ae60"
        if (left >= 50) return "#f39c12"
        return "#e74c3c"
    }

    function leftPctColorDim(left) {
        if (left < 0) return "#337f8c8d"
        if (left >= 80) return "#3327ae60"
        if (left >= 50) return "#33f39c12"
        return "#33e74c3c"
    }

    function reserveStr(w) {
        if (!w.limit || w.limit <= 0 || !w.reset_at) return ""

        var startDate, resetDate = new Date(w.reset_at)
        if (resetDate.getFullYear() <= 1) return ""

        if (w.period_start) {
            startDate = new Date(w.period_start)
            if (startDate.getFullYear() <= 1) startDate = null
        }

        if (!startDate) {
            var winHours = windowDurationHours(w.key || w.label)
            if (winHours <= 0) return ""
            startDate = new Date(resetDate.getTime() - winHours * 3600000)
        }

        var now = new Date()
        var totalDuration = (resetDate - startDate) / 1000
        var elapsed = (now - startDate) / 1000
        // Require at least 1% of window to have elapsed for meaningful burn rate
        if (elapsed <= 0 || elapsed < totalDuration * 0.01) return ""

        var remaining = (resetDate - now) / 1000
        if (remaining <= 0) return ""

        var rate = w.used / elapsed
        var projectedEnd = w.used + rate * remaining
        var reserve = w.limit - projectedEnd
        var reservePct = (reserve / w.limit) * 100

        if (reservePct >= 0) {
            return Math.round(reservePct) + "% in reserve"
        }

        var depletionSecs = ((w.limit - w.used) / rate)
        if (depletionSecs <= 0) return ""
        return "Runs out in " + formatDuration(depletionSecs)
    }
    function windowDurationHours(key) {
        var k = (key || "").toLowerCase()
        if (k === "5h" || k === "five_hour" || k === "session") return 5
        if (k === "weekly" || k === "week" || k === "7d") return 168
        if (k === "monthly" || k === "month" || k === "30d") return 744
        return 0
    }

    // Find the largest time window from a provider's windows, preferring monthly
    function largestWindow(provider) {
        var ws = provider.windows
        if (!ws || ws.length === 0) return null
        // Prefer monthly with a valid limit
        for (var i = 0; i < ws.length; i++) {
            var key = ws[i].key || ws[i].label
            if (key === "monthly" && ws[i].limit > 0) return ws[i]
        }
        // Fall back to largest window by rank
        var best = null, bestRank = -1
        for (var i = 0; i < ws.length; i++) {
            var rank = windowRank(ws[i].key || ws[i].label)
            if (rank > bestRank && ws[i].limit > 0) { bestRank = rank; best = ws[i] }
        }
        return best
    }

    // Reserve string for a provider's largest window
    function providerReserveStr(provider) {
        var w = largestWindow(provider)
        return w ? reserveStr(w) : ""
    }

    // Reset string for a provider's largest window
    function providerFormatReset(provider) {
        var w = largestWindow(provider)
        return w ? formatReset(w) : ""
    }

    function formatDuration(secs) {
        secs = Math.round(secs / 60) * 60 // round to minute
        var d = Math.floor(secs / 86400)
        var h = Math.floor((secs % 86400) / 3600)
        var m = Math.floor((secs % 3600) / 60)
        if (secs <= 0) return ""
        if (d > 0) return d + "d" + (h > 0 ? " " + h + "h" : "")
        if (h > 0) return h + "h " + String(m).padStart(2, "0") + "m"
        return m + "m"
    }

    function isLargestWindow(w, windows) {
        if (!windows || windows.length <= 1) return true
        var myRank = windowRank(w.key || w.label)
        for (var i = 0; i < windows.length; i++) {
            if (windows[i] !== w && windowRank(windows[i].key || windows[i].label) > myRank) return false
        }
        return true
    }

    function formatReset(w) {
        if (!w || !w.reset_at) return ""
        var d = new Date(w.reset_at)
        if (d.getFullYear() <= 1) return ""
        var remaining = (d - new Date()) / 1000
        if (remaining <= 0) return "resets now"
        var days = Math.floor(remaining / 86400)
        var hours = Math.floor((remaining % 86400) / 3600)
        var mins = Math.floor((remaining % 3600) / 60)
        if (days > 0) return "resets in " + days + "d" + (hours > 0 ? " " + hours + "h" : "")
        if (hours > 0) return "resets in " + hours + "h " + String(mins).padStart(2, "0") + "m"
        return "resets in " + mins + "m"
    }

    // ══════════════════════════════════════════════════════════
    //  Compact representation (panel)
    // ══════════════════════════════════════════════════════════
    compactRepresentation: MouseArea {
        id: compactMouse
        implicitWidth: compactRow.implicitWidth + Kirigami.Units.smallSpacing * 2
        implicitHeight: compactRow.implicitHeight + Kirigami.Units.smallSpacing * 2
        hoverEnabled: true
        cursorShape: Qt.PointingHandCursor
        onClicked: root.expanded = !root.expanded

        RowLayout {
            id: compactRow
            anchors.centerIn: parent
            spacing: 2

            // App icon (shown when unpinned)
            Kirigami.Icon {
                source: "office-chart-bar"
                width: 16; height: 16
                visible: !root.pinnedProvider
                Layout.alignment: Qt.AlignVCenter
            }

            // Colored dot (shown when pinned)
            Rectangle {
                width: 8; height: 8; radius: 4
                color: root.compactColor
                visible: !!root.pinnedProvider
                Layout.alignment: Qt.AlignVCenter
            }

            // Percentage / status text (shown when pinned)
            Label {
                text: root.compactText
                color: root.compactColor
                font.pixelSize: 11
                font.weight: root.worstPct >= 50 ? Font.DemiBold : Font.Normal
                visible: !!root.pinnedProvider
                Layout.alignment: Qt.AlignVCenter
            }
        }
    }

    // ══════════════════════════════════════════════════════════
    //  Full representation (popup)
    // ══════════════════════════════════════════════════════════
    fullRepresentation: ColumnLayout {

        spacing: 0

        // ── Header ───────────────────────────────────────────
        RowLayout {
            Layout.fillWidth: true
            Layout.margins: Kirigami.Units.smallSpacing

            Kirigami.Icon {
                source: "office-chart-bar"
                width: Kirigami.Units.iconSizes.small
                height: Kirigami.Units.iconSizes.small
            }
            Label {
                text: "PakAI"
                font.weight: Font.Bold
                font.pixelSize: 13
                Layout.fillWidth: true
            }
            Label {
                text: loading ? "..." : agoText(lastRefresh)
                color: Kirigami.Theme.disabledTextColor
                font.pixelSize: 10
            }
            ToolButton {
                icon.name: "view-refresh"
                onClicked: { fetchStatus(); fetchConfig() }
                display: AbstractButton.IconOnly
            }
        }

        // Separator
        Rectangle {
            Layout.fillWidth: true
            height: 1
            color: Qt.rgba(Kirigami.Theme.textColor.r, Kirigami.Theme.textColor.g, Kirigami.Theme.textColor.b, 0.15)
        }

        // ── Provider list ────────────────────────────────────
        ListView {
            id: listView
            Layout.fillWidth: true
            Layout.fillHeight: true
            Layout.topMargin: Kirigami.Units.smallSpacing
            Layout.bottomMargin: Kirigami.Units.smallSpacing
            model: root.groupedProviders
            clip: true
            boundsBehavior: Flickable.StopAtBounds
            spacing: 0

            delegate: Item {
                width: listView.width - Kirigami.Units.smallSpacing * 2
                x: Kirigami.Units.smallSpacing
                height: contentCol.implicitHeight + Kirigami.Units.smallSpacing * 2 * 2 + Kirigami.Units.smallSpacing * 2.5

                // ── Card background (fills full card area including padding) ──
                Rectangle {
                    anchors.fill: parent
                    anchors.bottomMargin: Kirigami.Units.smallSpacing * 2.5
                    radius: 10
                    z: -1
                    color: {
                        var tint = Qt.rgba(Kirigami.Theme.textColor.r, Kirigami.Theme.textColor.g, Kirigami.Theme.textColor.b, 0.04 + (index % 2) * 0.06)
                        var base = Kirigami.Theme.backgroundColor
                        var a = tint.a
                        return Qt.rgba(base.r * (1 - a) + tint.r * a, base.g * (1 - a) + tint.g * a, base.b * (1 - a) + tint.b * a, base.a)
                    }
                }

                // ── Padded card content (x/y/width creates internal padding) ──
                ColumnLayout {
                    id: contentCol
                    x: Kirigami.Units.smallSpacing * 2
                    y: Kirigami.Units.smallSpacing * 2
                    width: parent.width - Kirigami.Units.smallSpacing * 4
                    spacing: 0


                // ── Provider row ─────────────────────────────
                RowLayout {
                    Layout.fillWidth: true
                    spacing: Kirigami.Units.smallSpacing

                    Rectangle {
                        width: 8; height: 8; radius: 4
                        color: modelData.provider.status === "error" ? "#e74c3c"
                            : (modelData.provider.limit > 0 ? leftPctColor(pctLeft(modelData.provider.used, modelData.provider.limit)) : "#7f8c8d")
                    }
                    Label {
                        text: modelData.provider.label || modelData.provider.provider
                        font.weight: Font.Medium
                        font.pixelSize: 12
                        elide: Text.ElideRight
                        Layout.fillWidth: true
                    }

                    Label {
                        text: {
                            var p = modelData.provider
                            var left = pctLeft(p.used, p.limit)
                            if (left >= 0) return Math.round(left) + "% left"
                            if (p.unit === "usd" && !modelData.isGroup) return "$" + p.used.toFixed(2)
                            return formatUsed(p.used, p.unit)
                        }
                        color: modelData.provider.limit > 0 ? leftPctColor(pctLeft(modelData.provider.used, modelData.provider.limit)) : Kirigami.Theme.disabledTextColor
                        font.pixelSize: 11
                        visible: modelData.provider.status === "ok" && modelData.provider.limit > 0
                    }

                    // Pin button (only for working providers)
                    ToolButton {
                        visible: modelData.provider.status === "ok"
                        icon.name: root.pinnedProvider === modelData.provider.provider ? "pin" : "pin-off"
                        display: AbstractButton.IconOnly
                        opacity: root.pinnedProvider === modelData.provider.provider ? 1.0 : 0.5
                        ToolTip.visible: hovered
                        ToolTip.text: root.pinnedProvider === modelData.provider.provider ? "Unpin from panel" : "Pin to panel"
                        onClicked: {
                            root.pinnedProvider = root.pinnedProvider === modelData.provider.provider ? "" : modelData.provider.provider
                            if (root.pinnedProvider)
                                root.apiConfig("pin=" + root.pinnedProvider)
                            else
                                root.apiConfig("unpin")
                        }
                    }

                    // Eye / hide button
                    ToolButton {
                        icon.name: "visibility-off"
                        display: AbstractButton.IconOnly
                        opacity: 0.6
                        ToolTip.visible: hovered
                        ToolTip.text: "Hide provider"
                        onClicked: {
                            var hideId = "" + modelData.provider.provider
                            root.apiConfig("hide=" + hideId)
                            var h = []
                            for (var ci = 0; ci < root.hiddenProviders.length; ci++)
                                h.push(root.hiddenProviders[ci])
                            h.push(hideId)
                            root.hiddenProviders = h
                            root.processData()
                        }
                    }
                }

                // ── Windows (5h / weekly / monthly) ──────────
                ColumnLayout {
                    visible: modelData.provider.status === "ok"
                    id: provWinColumn
                    Layout.fillWidth: true
                    property var providerWindows: (modelData.provider && modelData.provider.windows) || []
                    spacing: 4
                    Layout.leftMargin: Kirigami.Units.gridUnit

                    Repeater {
                        model: provWinColumn.providerWindows
                        delegate: ColumnLayout {
                            required property var modelData
                            property var win: modelData
                            Layout.fillWidth: true
                            spacing: 1

                            // Bar row: label, bar, percent left
                            RowLayout {
                                Layout.fillWidth: true
                                spacing: Kirigami.Units.smallSpacing

                                Label {
                                    text: shortWinLabel(win ? (win.key || win.label) : "")
                                    font.pixelSize: 10
                                    color: Kirigami.Theme.disabledTextColor
                                    Layout.preferredWidth: Kirigami.Units.gridUnit * 1.5
                                }

                                Rectangle {
                                    Layout.fillWidth: true
                                    Layout.alignment: Qt.AlignVCenter
                                    height: 4; radius: 2
                                    color: Qt.rgba(0.5, 0.5, 0.5, 0.2)

                                    Rectangle {
                                        height: 4; radius: 2
                                        width: win && win.limit > 0 ? Math.min(parent.width, (win.used / win.limit) * parent.width) : 0
                                        color: "#e74c3c"
                                    }
                                }

                                Label {
                                    text: {
                                        if (win && win.limit > 0) return Math.round(pctLeft(win.used, win.limit)) + "% left"
                                        return win ? formatCompact(win.used, win.unit) : ""
                                    }
                                    font.pixelSize: 9
                                    color: win && win.limit > 0 ? leftPctColor(pctLeft(win.used, win.limit)) : Kirigami.Theme.disabledTextColor
                                }
                            }

                            // Reset + reserve row
                            RowLayout {
                                Layout.fillWidth: true
                                spacing: Kirigami.Units.smallSpacing

                                // Reset countdown
                                Label {
                                    text: win ? formatReset(win) : ""
                                    font.pixelSize: 8
                                    color: Kirigami.Theme.disabledTextColor
                                    visible: text !== ""
                                }

                                // Reserve — only on monthly window
                                Label {
                                    text: win && (win.key === "monthly" || win.key === "month" || shortWinLabel(win.key || win.label) === "mo") ? "| " + reserveStr(win) : ""
                                    font.pixelSize: 8
                                    color: Kirigami.Theme.disabledTextColor
                                    visible: text !== ""
                                }
                            }
                        }
                    }
                }
                Label {
                    visible: modelData.provider.status !== "ok" && (modelData.provider.error || modelData.provider.warning)
                    text: modelData.provider.error || modelData.provider.warning || ""
                    color: "#e74c3c"
                    font.pixelSize: 10
                    Layout.fillWidth: true
                    wrapMode: Text.WordWrap
                }

                // ── Sub-providers (OpenCode group) ───────────
                ColumnLayout {
                    visible: modelData.isGroup && modelData.subs && modelData.subs.length > 0
                    spacing: 0
                    Layout.topMargin: 2
                    Layout.leftMargin: Kirigami.Units.gridUnit * 1.5

                    Repeater {
                        model: modelData.subs
                        delegate: ColumnLayout {
                            Layout.fillWidth: true
                            spacing: 0

                            RowLayout {
                                Layout.fillWidth: true
                                spacing: Kirigami.Units.smallSpacing

                                Rectangle {
                                    width: 6; height: 6; radius: 3
                                    color: modelData.status === "error" ? "#e74c3c"
                                        : (modelData.limit > 0 ? leftPctColor(pctLeft(modelData.used, modelData.limit)) : "#7f8c8d")
                                }

                                Label {
                                    text: subLabel(modelData)
                                    font.pixelSize: 11
                                    elide: Text.ElideRight
                                    Layout.fillWidth: true
                                }

                                Label {
                                    text: {
                                        var left = pctLeft(modelData.used, modelData.limit)
                                        if (left >= 0) return Math.round(left) + "% left"
                                        return formatUsed(modelData.used, modelData.unit)
                                    }
                                    color: modelData.limit > 0 ? leftPctColor(pctLeft(modelData.used, modelData.limit)) : Kirigami.Theme.disabledTextColor
                                    font.pixelSize: 10
                                }
                            }

                            Rectangle {
                                visible: modelData.status === "ok"
                                Layout.fillWidth: true
                                Layout.topMargin: 1
                                height: 4; radius: 2
                                color: modelData.limit > 0 ? leftPctColorDim(pctLeft(modelData.used, modelData.limit)) : Qt.rgba(0.5, 0.5, 0.5, 0.15)

                                Rectangle {
                                    height: 4; radius: 2
                                    width: modelData.limit > 0 ? Math.min(parent.width, (modelData.used / modelData.limit) * parent.width) : 0
                                    color: modelData.limit > 0 ? leftPctColor(pctLeft(modelData.used, modelData.limit)) : "transparent"
                                }
                            }
                            // Sub-provider windows
                            ColumnLayout {
                                id: subWinCol
                                Layout.fillWidth: true
                                property var subWindowsList: modelData ? modelData.windows : []
                                visible: modelData.status === "ok" && subWindowsList.length > 0
                                spacing: 0
                                Layout.topMargin: 1

                                Repeater {
                                    model: subWindowsList ? subWindowsList.length : 0
                                    delegate: ColumnLayout {
                                        property var win: subWindowsList ? subWindowsList[index] : null
                                        Layout.fillWidth: true
                                        spacing: 1

                                        // Bar row: label, bar, percent left
                                        RowLayout {
                                            Layout.fillWidth: true
                                            spacing: Kirigami.Units.smallSpacing

                                            Label {
                                                text: shortWinLabel(win ? (win.key || win.label) : "")
                                                font.pixelSize: 9
                                                color: Kirigami.Theme.disabledTextColor
                                                Layout.preferredWidth: Kirigami.Units.gridUnit
                                            }

                                            Rectangle {
                                                Layout.fillWidth: true
                                                Layout.alignment: Qt.AlignVCenter
                                                height: 3; radius: 2
                                                color: Qt.rgba(0.5, 0.5, 0.5, 0.2)

                                                Rectangle {
                                                    height: 3; radius: 2
                                                    width: win && win.limit > 0 ? Math.min(parent.width, (win.used / win.limit) * parent.width) : 0
                                                    color: "#e74c3c"
                                                }
                                            }

                                            Label {
                                                text: {
                                                    if (win && win.limit > 0) return Math.round(pctLeft(win.used, win.limit)) + "% left"
                                                    return win ? formatCompact(win.used, win.unit) : ""
                                                }
                                                font.pixelSize: 8
                                                color: win && win.limit > 0 ? leftPctColor(pctLeft(win.used, win.limit)) : Kirigami.Theme.disabledTextColor
                                            }
                                        }

                                        // Reset + reserve row
                                        RowLayout {
                                            Layout.fillWidth: true
                                            spacing: Kirigami.Units.smallSpacing

                                            Label {
                                                text: win ? formatReset(win) : ""
                                                font.pixelSize: 7
                                                color: Kirigami.Theme.disabledTextColor
                                                visible: text !== ""
                                            }

                                            Label {
                                                text: win && (win.key === "monthly" || win.key === "month" || shortWinLabel(win.key || win.label) === "mo") ? reserveStr(win) : ""
                                                font.pixelSize: 7
                                                color: Kirigami.Theme.disabledTextColor
                                                visible: text !== ""
                                            }
                                        }
                                    }
                                }
                            }
                        }
                    }
                }

                }

            }

            // ── Empty state ──────────────────────────────────
            Label {
                anchors.centerIn: parent
                text: root.errorMsg ? "Daemon unreachable\n\nRun: pakai daemon start"
                    : (root.loading ? "Loading..." : "No providers")
                color: Kirigami.Theme.disabledTextColor
                horizontalAlignment: Text.AlignHCenter
                font.pixelSize: 12
                visible: listView.count === 0
            }
        }

        // ── Hidden providers (shown for unhiding) ────────────
        Item {
            visible: root.hiddenProviders.length > 0
            Layout.fillWidth: true
            Layout.leftMargin: Kirigami.Units.smallSpacing
            Layout.rightMargin: Kirigami.Units.smallSpacing
            implicitHeight: hiddenColumn.implicitHeight + Kirigami.Units.smallSpacing * 4

            // Card background (covers full section)
            Rectangle {
                anchors.fill: parent
                radius: 10
                color: {
                    var tint = Qt.rgba(Kirigami.Theme.textColor.r, Kirigami.Theme.textColor.g, Kirigami.Theme.textColor.b, 0.12)
                    var base = Kirigami.Theme.backgroundColor
                    var a = tint.a
                    return Qt.rgba(base.r * (1 - a) + tint.r * a, base.g * (1 - a) + tint.g * a, base.b * (1 - a) + tint.b * a, base.a)
                }
            }

            ColumnLayout {
                id: hiddenColumn
                x: Kirigami.Units.smallSpacing * 2
                y: Kirigami.Units.smallSpacing * 2
                width: parent.width - Kirigami.Units.smallSpacing * 4
                spacing: 0

                // ── Header with collapse toggle ──
                RowLayout {
                    id: headerRow
                    Layout.fillWidth: true
                    spacing: Kirigami.Units.smallSpacing

                    Label {
                        text: "Hidden providers:"
                        font.pixelSize: 10
                        font.weight: Font.Medium
                        color: Kirigami.Theme.disabledTextColor
                        Layout.fillWidth: true
                    }

                    ToolButton {
                        icon.name: root.hiddenProvidersCollapsed ? "arrow-right" : "arrow-down"
                        display: AbstractButton.IconOnly
                        ToolTip.visible: hovered
                        ToolTip.text: root.hiddenProvidersCollapsed ? "Show hidden providers" : "Hide hidden providers"
                        onClicked: root.hiddenProvidersCollapsed = !root.hiddenProvidersCollapsed
                    }
                }

                // ── Hidden provider rows (visible when expanded) ──
                ColumnLayout {
                    id: rowsColumn
                    Layout.fillWidth: true
                    Layout.preferredHeight: root.hiddenProvidersCollapsed ? 0 : implicitHeight
                    clip: true
                    spacing: 2

                    Repeater {
                        model: root.hiddenProviders
                        delegate: RowLayout {
                            Layout.fillWidth: true
                            spacing: Kirigami.Units.smallSpacing

                            Label {
                                text: modelData
                                font.pixelSize: 10
                                color: Kirigami.Theme.disabledTextColor
                                Layout.fillWidth: true
                            }

                            ToolButton {
                                icon.name: "visibility"
                                display: AbstractButton.IconOnly
                                ToolTip.visible: hovered
                                ToolTip.text: "Unhide provider"
                                onClicked: {
                                    var removeId = "" + modelData
                                    root.apiConfig("unhide=" + removeId)
                                    var h = []
                                    for (var ci = 0; ci < root.hiddenProviders.length; ci++) {
                                        var id = root.hiddenProviders[ci]
                                        if (id !== removeId)
                                            h.push(id)
                                    }
                                    root.hiddenProviders = h
                                    root.processData()
                                }
                            }
                        }
                    }
                }
            }
        }
        // Separator
        Rectangle {
            Layout.fillWidth: true
            height: 1
            color: Qt.rgba(Kirigami.Theme.textColor.r, Kirigami.Theme.textColor.g, Kirigami.Theme.textColor.b, 0.15)
        }

        // ── Footer ───────────────────────────────────────────
        RowLayout {
            Layout.fillWidth: true
            Layout.margins: Kirigami.Units.smallSpacing

            Label {
                text: {
                    var n = root.groupedProviders.length
                    if (n === 0) return "No providers"
                    return n + " provider" + (n !== 1 ? "s" : "")
                }
                font.pixelSize: 10
                color: Kirigami.Theme.disabledTextColor
                Layout.fillWidth: true
            }

            Label {
                text: "port 7731"
                font.pixelSize: 10
                color: Kirigami.Theme.disabledTextColor
            }
        }
    }
}
