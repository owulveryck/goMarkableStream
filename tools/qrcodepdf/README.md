# reMarkable IP Streamer QR Code Generator

This tool is designed to fetch the IP addresses of your reMarkable device and generate a QR code containing these IP addresses in a PDF. The generated PDF can be displayed on the reMarkable and scanned with a mobile device to reveal the IP addresses. Please note that this feature is experimental.

## Installation

To set up the tool, follow these steps:

1. **Upload the PDF File**: First, you need to upload the file `goMarkableStreamQRCode.pdf` to your reMarkable device. You can do this using the official reMarkable tool.

2. **Compile the Tool**: 
    - On your computer, open a terminal and navigate to the directory containing the source code.
    - Run the following command to compile the tool:
    ```bash
    GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 go build -o genIP.arm
    ```

3. **Copy the Binary to reMarkable**: 
    - Use `scp` or any other SSH tool to copy the `genIP.arm` binary to your reMarkable device. 
    - Example:
    ```bash
    scp genIP.arm root@<remarkable_ip>:
    ```

4. **Launch the Tool**: 
    - SSH into your reMarkable device:
    ```bash
    ssh root@<remarkable_ip>
    ```
    - Run the tool:
    ```bash
    ./genIP.arm &
    ```
    Adding `&` at the end of the command will make it run in the background, allowing it to continue running even after you disconnect.

## How it Works

The utility runs in an endless loop, checking every minute to see if the network has changed. If a change in the network is detected, it updates the content of the `goMarkableStreamQRCode.pdf` file with the new IP addresses.

## Notes

- This tool is experimental; use it at your own risk and feel free to contribute or report issues.

