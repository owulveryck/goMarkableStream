import QtQuick 2.5
import QtQuick.Controls 2.5
import net.asivery.AppLoad 1.0

// Console window for goMarkableStream.
// The backend spawns the real binary and forwards its stdout+stderr here.
// Message protocol (must match backend/main.go):
//   backend -> frontend:  type 1 = append one line,  type 2 = full log buffer (on attach)
//   frontend -> backend:  type 100 = request buffer, type 101 = stop stream + kill backend
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
                // keep the on-device buffer from growing without bound
                if (root.logText.length > 200000)
                    root.logText = root.logText.slice(root.logText.length - 150000);
            }
            // auto-scroll to the bottom
            flick.contentY = Math.max(0, logView.height - flick.height);
        }
    }

    // On (re)attach, ask the backend to replay whatever it has captured so far.
    Component.onCompleted: endpoint.sendMessage(100, "")

    Row {
        id: controls
        height: 70
        spacing: 12
        anchors.top: parent.top
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.margins: 10

        Rectangle {
            width: 220; height: 60; border.width: 2; border.color: "black"
            Text { anchors.centerIn: parent; text: "Stop stream"; font.pointSize: 20 }
            MouseArea { anchors.fill: parent; onClicked: endpoint.sendMessage(101, "") }
        }
        Rectangle {
            width: 220; height: 60; border.width: 2; border.color: "black"
            Text { anchors.centerIn: parent; text: "Clear log"; font.pointSize: 20 }
            MouseArea { anchors.fill: parent; onClicked: root.logText = "" }
        }
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
                font.pointSize: 14
                textFormat: Text.PlainText
                text: root.logText.length ? root.logText : "Waiting for goMarkableStream output..."
            }
        }
    }
}
