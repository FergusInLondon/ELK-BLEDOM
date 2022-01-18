# Proof of Concept

This is a simple demonstration of (a) connecting to the ELK BLE RGB LED Controller, and (b) sending commands for changing colour and brightness.

*It's rough but functional.*

## Usage

```
pi@pibox:~/bledom $ ./poc
enabling adapter
waiting for scan results
scanning...
found device: BE:58:80:01:11:EC -47 ELK-BLEDOM   
connected to  BE:58:80:01:11:EC
discovering services/characteristics
found service 0000fff0-0000-1000-8000-00805f9b34fb
found characteristic 0000fff3-0000-1000-8000-00805f9b34fb
changing brightness, new value 255
recieved payload to write [126 0 1 255 0 0 0 0 239]
writing colour payload [128 0 0]
recieved payload to write [126 0 5 3 128 0 0 0 239]
requesting device status
writing colour payload [128 128 0]
read characteristic:  16 [32 83 72 89 45 86 57 46 49 46 48 82 53 55 54 57]
recieved payload to write [126 0 5 3 128 128 0 0 239]
writing colour payload [0 128 128]
^Crecieved SIGTERM, requesting application termination
context cancelled, no longer writing characteristics context canceled
context cancellation, stopping brightness loop context canceled
waiting for device connection to close
context cancelled, stopping device communication context canceled
connection closed. terminating...
```
