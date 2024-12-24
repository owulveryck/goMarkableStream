# RAW client

This tool connects to a server over HTTPS, fetches raw RGBA image data, and saves it as a PNG file. It supports basic authentication and allows configuration of server details, image dimensions, and output file.

Run the server on your reMarkable:

```
RK_DEV_MODE=true ./goMarkableStream
```

Run the client:

```
go run tools/client/client.go --ip 10.0.0.1  # Replace with your device IP
```
