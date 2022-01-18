# Go interactions for the ELK-BLEDOM RGB LED Controller

This repository contains information on the common (and cheap) ELK-BLEDOM Bluetooth Low Energy RGB LED controller, as well as a Go package for interacting with it.

Bluetooth interactions have been reverse engineered from the Android app. See [`PROTOCOL.md`](PROTCOL.md) for more information on both the hardware and how it can be controlled.

## Example

<p align="center">
	<a href="https://www.youtube.com/watch?v=xiWuZdq0pWM">
		<img src="https://img.youtube.com/vi/xiWuZdq0pWM/0.jpg" />
	</a>
</p>

```go
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	d := device.NewBleDomRGBController(ctx, device.Options{})

	if err := d.Connect(time.Duration(time.Minute)); err != nil {
		panic(err)
	}

	gracefulTermination(cancel, d)

	// Don't terminate until device has been cleanly shutdown
	defer func() {
		fmt.Println("waiting for device disconnection")
		d.Done()
		fmt.Println("device disconnected: terminating app")
	}()

	// Configure state polling; retrieve state every 30 seconds and log
	// it to stdout.
	d.PollState(time.Duration(30)*time.Second, func(state []byte) {
		fmt.Println("recieved latest state from device", state)
	})

	fmt.Println("setting brightness to maximum")
	d.WriteCommand(device.BrightnessCommandFromValue(96)) // 96%

	i := 1
	t := time.NewTimer(time.Second)
	for {
		select {
		case <-ctx.Done():
			fmt.Println("context cancelled: stopping commands")
			t.Stop()
			return
		case <-t.C:
			cmd := commands[i%len(commands)]
			fmt.Println("changing colour: ", cmd.Red, cmd.Green, cmd.Blue)
			d.WriteCommand(cmd)

			i = i + 1
			t.Reset(time.Second)
		}
	}
}
```

See [colours.go](example/colours.go).
