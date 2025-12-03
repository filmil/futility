# serial_upload

`serial_upload` is a Go program designed to upload text content from a specified file over a serial connection. It takes various command-line flags to configure the serial port parameters, the file to upload, and a prompt string to wait for before initiating the upload.

### Functionality

Upon execution, the program opens the configured serial port and sets its parameters (baud rate, start/stop bits, parity). It then listens for an incoming string on the serial port that exactly matches the provided prompt line. Once the prompt is received, the program transmits the entire content of the specified file through the serial connection.

For detailed requirements and development tasks, please refer to the [specification document](spec.md).
