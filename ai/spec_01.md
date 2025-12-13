# Task: add a "linger" option

## Constraints

Read the file `spec.md` to learn about constraints.

## Subtask 1

* Add a flag `--linger`, which if specified tells the program to continue
  receiving on serial port after having sent the file contents. All received
  data from serial port is to be echoed to standard output until the user
  explicitly interrupts the program.
