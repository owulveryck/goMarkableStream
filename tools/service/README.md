This service allows auto-start and restart of the goMarkableStreaming service

## Installation

```shell
cat > /etc/systemd/system/goMarkableStream.service << EOF
[Unit]
Description=goMarkableStream Service
After=xochitl.service

[Service]
ExecStart=/home/root/goMarkableStream
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable goMarkableStream.Service
systemctl start goMarkableStream.service
```

The system will start automatically

_note_ any updates on the reMarkable removes this service
