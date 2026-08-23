import QtQuick 2.15
import QtQuick.Controls 2.15
import net.asivery.AppLoad 1.0

// GMS Console — control the goMarkableStream systemd service.
//
// The backend runs systemctl / journalctl and forwards their output here.
// Message protocol (must match backend/main.go):
//   backend -> frontend:  1 = append text, 2 = full log buffer, 3 = status word
//   frontend -> backend:  100 = request buffer,
//                         101 = start, 102 = stop, 103 = restart,
//                         104 = status + logs, 105 = clear
Rectangle {
    id: root
    anchors.fill: parent
    color: "white"

    property string logText: ""
    property string svcState: "unknown"

    signal close
    function unloading() { }

    // Bundled JetBrains Mono (packed into resources.rcc). Falls back to a
    // generic monospace family if the font fails to load for any reason.
    FontLoader { id: mono; source: "qrc:/fonts/JetBrainsMono-Regular.ttf" }
    property string monoFamily: mono.status === FontLoader.Ready ? mono.name : "monospace"

    function summaryText(s) {
        switch (s) {
        case "active":       return "\u25CF  Service running";
        case "activating":   return "\u25CF  Service starting\u2026";
        case "reloading":    return "\u25CF  Service reloading\u2026";
        case "deactivating": return "\u25CF  Service stopping\u2026";
        case "inactive":     return "\u25CF  Service not running";
        case "failed":       return "\u25CF  Service errored out";
        case "unknown":      return "\u25CF  Service state unknown";
        default:             return "\u25CF  " + s;
        }
    }
    function summaryColor(s) {
        switch (s) {
        case "active":                     return "#1a7f1a";
        case "failed":                     return "#b00020";
        case "activating":
        case "reloading":
        case "deactivating":               return "#a15c00";
        default:                           return "#444444";
        }
    }

    AppLoad {
        id: endpoint
        applicationID: "gms-console"
        onMessageReceived: (type, contents) => {
            if (type === 2) {
                root.logText = contents;
            } else if (type === 1) {
                root.logText += contents;
                if (root.logText.length > 200000)
                    root.logText = root.logText.slice(root.logText.length - 150000);
            } else if (type === 3) {
                root.svcState = contents;
            }
            // auto-scroll to the bottom
            flick.contentY = Math.max(0, logView.height - flick.height);
        }
    }

    // On (re)attach, ask the backend to replay whatever it has captured so far.
    Component.onCompleted: endpoint.sendMessage(100, "")

    // A reusable flat button.
    component ActionButton: Rectangle {
        property alias text: label.text
        signal clicked
        width: 190
        height: 76
        color: "white"
        border.width: 2
        border.color: "black"
        radius: 8
        Text {
            id: label
            anchors.centerIn: parent
            font.pixelSize: 30
        }
        MouseArea {
            anchors.fill: parent
            onPressed: parent.color = "#dddddd"
            onReleased: parent.color = "white"
            onCanceled: parent.color = "white"
            onClicked: parent.clicked()
        }
    }

    Column {
        id: top
        anchors.top: parent.top
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.margins: 12
        spacing: 12

        Flow {
            width: parent.width
            spacing: 12
            ActionButton { text: "Start";         onClicked: endpoint.sendMessage(101, "") }
            ActionButton { text: "Stop";          onClicked: endpoint.sendMessage(102, "") }
            ActionButton { text: "Restart";       onClicked: endpoint.sendMessage(103, "") }
            ActionButton { text: "Status & Logs"; width: 250; onClicked: endpoint.sendMessage(104, "") }
            ActionButton { text: "Clear";         onClicked: { root.logText = ""; endpoint.sendMessage(105, ""); } }
        }

        // Status summary badge.
        Text {
            id: badge
            text: root.summaryText(root.svcState)
            color: root.summaryColor(root.svcState)
            font.pixelSize: 34
            font.bold: true
        }
    }

    Rectangle {
        anchors.top: top.bottom
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.bottom: parent.bottom
        anchors.margins: 12
        anchors.topMargin: 6
        border.width: 2
        border.color: "black"

        Flickable {
            id: flick
            anchors.fill: parent
            anchors.margins: 8
            contentWidth: width
            contentHeight: logView.height
            clip: true
            boundsBehavior: Flickable.StopAtBounds

            Text {
                id: logView
                width: flick.width
                wrapMode: Text.WrapAnywhere
                font.family: root.monoFamily
                font.pixelSize: 24
                textFormat: Text.PlainText
                text: root.logText.length ? root.logText
                        : "Ready. Tap a button to control the goMarkableStream service."
            }
        }
    }
}
