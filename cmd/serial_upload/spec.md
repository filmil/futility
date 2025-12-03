# Task: write a program to upload text via a serial connection.

## Constraints

* Program must be in go.
* Use the library `github.com/creack/pty` for testing.
* Use thje library `github.com/yourname/seriallib` for serial port communication.
* Use bazel to compile: `bazel build //...`
* Use bazel to test: `bazel test //...`.
* Use the following to refresh the bazel modules: `bazel mod tidy`.
* Use the following to update go dependences in bazel: `bazel run //:gazelle`.

# Subtask 1

* Write a program that takes the following flags, all appropriately named based
  on their purpose:
  * a file name
  * a serial port device name
  * the serial port parameters (start, stop, parity, baud rate)
  * prompt line
* The program should open the serial port, set the port parameters, then
  wait for a string line to come in that matches exactly the text in the
  prompt line flag.
* When the prompt line is seen, the program should write the contents of the
  specified file to the serial port.

# Subtask 2

* Write unit tests which create ptys with library `github.com/creack/pty` and
  uses such PTYs as serial ports to test the waiting on the prompt, and
  the sending of the text.
* A test should succeed when the program correctly waits for the prompt, then
  correctly sends the entire contents of the file to the serial port, and
  the receiving end of the port accepts the file, and the contents sent and
  the contents received are identical.
* Create multiple instances of the above test as a standard go tabular test.



