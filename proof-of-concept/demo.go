package main

import (
	"context"
	"fmt"
	"time"

	"tinygo.org/x/bluetooth"
)

type bledomDemonstration struct {
	ctx       context.Context
	cancel    context.CancelFunc
	payloadCh chan []byte
	done      chan struct{}
	device    *bluetooth.Device
}

func newDemo(
	ctx context.Context, cancel context.CancelFunc, device *bluetooth.Device, char bluetooth.DeviceCharacteristic,
) *bledomDemonstration {
	d := &bledomDemonstration{
		ctx: ctx, cancel: cancel,
		device:    device,
		payloadCh: make(chan []byte),
		done:      make(chan struct{}),
	}

	go d._writer(char)
	return d
}

func (d *bledomDemonstration) run() {
	brightnessIncrease := false
	for {
		if err := d.brightnessIterator(
			d.ctx, brightnessIncrease, d.payloadCh, d.colourIterator,
		); err != nil {
			break
		}

		brightnessIncrease = !brightnessIncrease
	}
}

func (d *bledomDemonstration) _writer(char bluetooth.DeviceCharacteristic) {
	defer close(d.done)
	errCount := 0

	t := time.NewTimer(time.Duration(2) * time.Second)

	for {
		if errCount >= 5 {
			fmt.Println("closing connection due to error count", errCount)
			d.cancel()
		}

		select {
		case <-d.ctx.Done():
			fmt.Println("context cancelled, stopping device communication", d.ctx.Err())
			d.device.Disconnect()
			t.Stop()
			return
		case payload := <-d.payloadCh:
			fmt.Println("recieved payload to write", payload)
			if _, err := char.WriteWithoutResponse(payload); err != nil {
				fmt.Println("received error when writing to characteristic", err)
				errCount++
			}
		case <-t.C:
			fmt.Println("requesting device status")
			data := make([]byte, 16)
			count, err := char.Read(data)
			if err != nil {
				fmt.Println("unable to read characteristic", err)
				continue
			}

			fmt.Println("read characteristic: ", count, data)
		}
	}
}

func (d *bledomDemonstration) colourIterator() {
	p := func(colour []byte) []byte {
		if len(colour) != 3 {
			fmt.Println("a colour should have 3 bytes! falling back to RED.", colour)
			colour = []byte{0x80, 0x00, 0x00}
		}

		return []byte{0x7e, 0x00, 0x05, 0x03, colour[0], colour[1], colour[2], 0x00, 0xef}
	}

	t := time.NewTimer(time.Second)
	defer t.Stop()

	for _, c := range [][]byte{
		{0x80, 0x00, 0x00},
		{0x80, 0x80, 0x00},
		{0x00, 0x80, 0x80},
		{0x00, 0x80, 0x80},
		{0x00, 0x00, 0x80},
		{0x80, 0x00, 0x80},
		{0x80, 0x80, 0x80},
	} {
		select {
		case <-d.ctx.Done():
			fmt.Println("context cancelled, no longer writing characteristics", d.ctx.Err())
			return
		case <-t.C:
			fmt.Println("writing colour payload", c)
			d.payloadCh <- p(c)
		}

		t.Reset(time.Second)
	}
}

func (d *bledomDemonstration) brightnessIterator(
	ctx context.Context, gettingBrighter bool, payloadCh chan []byte, fn func(),
) error {
	makePayload := func(b uint8) []byte {
		return []byte{0x7e, 0x00, 0x01, b, 0x00, 0x00, 0x00, 0x00, 0xef}
	}

	brightness := byte(0xFF)
	if gettingBrighter {
		brightness = byte(0xFF)
	}

	for {
		fmt.Println("changing brightness, new value", brightness)

		select {
		case <-d.ctx.Done():
			fmt.Println("context cancellation, stopping brightness loop", ctx.Err())
			return d.ctx.Err()
		default:
		}

		d.payloadCh <- makePayload(brightness)

		fn()

		completed := (gettingBrighter && brightness == 0xFA)
		completed = completed || (!gettingBrighter && brightness == 0x0A)

		if completed {
			fmt.Println("finished one brightness iteration")
			return nil
		}

		if gettingBrighter {
			brightness = brightness - 0x0A
		} else {
			brightness = brightness + 0x0A
		}
	}
}
