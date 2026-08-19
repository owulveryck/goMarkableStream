import QtQuick 2.5
import QtQuick.Controls 2.5
import net.asivery.AppLoad 1.0

// GMS Console — control the goMarkableStream systemd service.
//
// The backend runs systemctl / journalctl and forwards their output here.
// Message protocol (must match backend/main.go):
//   backend -> frontend:  type 1 = append text, type 2 = full log buffer (on attach)
//   frontend -> backend:  100 = request buffer,
//                         101 = start, 102 = stop, 103 = restart,
//                         104 = status + logs, 105 = clear
Rectangle {
    id: root
    anchors.fill: parent
    color: "white"

    property string logText: ""

    signal close
    function unloading() { }

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
        width: 150
        height: 64
        color: "white"
        border.width: 2
        border.color: "black"
        radius: 6
        Text {
            id: label
            anchors.centerIn: parent
            font.pointSize: 18
        }
        MouseArea {
            anchors.fill: parent
            onPressed: parent.color = "#dddddd"
            onReleased: parent.color = "white"
            onCanceled: parent.color = "white"
            onClicked: parent.clicked()
        }
    }

    Flow {
        id: controls
        spacing: 10
        anchors.top: parent.top
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.margins: 10

        ActionButton { text: "Start";         onClicked: endpoint.sendMessage(101, "") }
        ActionButton { text: "Stop";          onClicked: endpoint.sendMessage(102, "") }
        ActionButton { text: "Restart";       onClicked: endpoint.sendMessage(103, "") }
        ActionButton { text: "Status & Logs"; width: 200; onClicked: endpoint.sendMessage(104, "") }
        ActionButton { text: "Clear";         onClicked: { root.logText = ""; endpoint.sendMessage(105, ""); } }
    }

    Rectangle {
        anchors.top: controls.bottom
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.bottom: parent.bottom
        anchors.margins: 10
        border.width: 2
        border.color: "black"

        Flickable {
            id: flick
            anchors.fill: parent
            anchors.margins: 6
            contentWidth: width
            contentHeight: logView.height
            clip: true
            boundsBehavior: Flickable.StopAtBounds

            Text {
                id: logView
                width: flick.width
                wrapMode: Text.WrapAnywhere
                font.family: "monospace"
                font.pointSize: 13
                textFormat: Text.PlainText
                text: root.logText.length ? root.logText
                        : "Ready. Tap a button to control the goMarkableStream service."
            }
        }
    }
}
