package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"tinygo.org/x/bluetooth"
)

const (
	bledom_ServiceUUID        = "0000fff0-0000-1000-8000-00805f9b34fb"
	bledom_CharacteristicUUID = "0000fff3-0000-1000-8000-00805f9b34fb"
	// Implementation Note: The Android App actually treats two different
	// prefixes slightly differently - "ELK_" and "ELK-". Not immediately
	// clear why, and this will require some more investigation.
	bledom_DeviceNamePrefix = "ELK"
)

var (
	adapter = bluetooth.DefaultAdapter
)

func handleTermination(cancel context.CancelFunc) {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		fmt.Println("recieved SIGTERM, requesting application termination")
		cancel()
	}()
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	handleTermination(cancel)

	fmt.Println("enabling adapter")
	if err := adapter.Enable(); err != nil {
		panic(err)
	}

	deviceCh := make(chan bluetooth.ScanResult, 1)

	serviceUUID, err := bluetooth.ParseUUID(bledom_ServiceUUID)
	if err != nil {
		fmt.Println("invalid service UUID specified", err)
		panic(err)
	}

	// Start scanning.
	go func() {
		fmt.Println("scanning...")
		if err := adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
			fmt.Println("found device:", result.Address.String(), result.RSSI, result.LocalName())

			// Implementation Note: The device does not broadcast the correct Service
			// UUID as part of the advertisement; it broadcasts an (unimplemented?) HID
			// UUID - 00001812-0000-1000-8000-00805F9B34FB. Therefore we cannot filter
			// on this, despite that being nicer.
			//
			// Especially frustrating as there's no guarantee that a valid name will be
			// broadcast usually. This does explain some of the odd pairing logic in the
			// Android application.
			//
			// if result.HasServiceUUID(serviceUUID)
			if strings.Index(result.LocalName(), bledom_DeviceNamePrefix) == 0 {
				adapter.StopScan()
				deviceCh <- result
			}
		}); err != nil {
			fmt.Println("unable to scan via adapter", err)
		}
	}()

	fmt.Println("waiting for scan results")
	var device *bluetooth.Device
	select {
	case result := <-deviceCh:
		device, err = adapter.Connect(result.Address, bluetooth.ConnectionParams{})
		if err != nil {
			panic(err)
		}

		fmt.Println("connected to ", result.Address.String())
	case <-time.After(time.Duration(60) * time.Second):
		if device == nil {
			fmt.Println("device scan has timed out, stopping scan.")
			adapter.StopScan()
			cancel()
			return
		}
	}

	// get services
	fmt.Println("discovering services/characteristics")
	srvcs, err := device.DiscoverServices([]bluetooth.UUID{serviceUUID})
	if (err != nil) || len(srvcs) == 0 {
		fmt.Println("unable to find service on device", err)
		panic(err)
	}

	srvc := srvcs[0]
	fmt.Println("found service", srvc.UUID().String())

	characteristicUUID, err := bluetooth.ParseUUID(bledom_CharacteristicUUID)
	if err != nil {
		fmt.Println("invalid characteristic ID provided")
		panic(err)
	}

	chars, err := srvc.DiscoverCharacteristics([]bluetooth.UUID{characteristicUUID})
	if (err != nil) || len(chars) == 0 {
		fmt.Println("device doesn't implement desired characteristic!")
		panic(err)
	}

	char := chars[0]
	fmt.Println("found characteristic", char.UUID().String())

	demo := newDemo(ctx, cancel, device, char)
	demo.run()

	fmt.Println("waiting for device connection to close")
	<-demo.done

	fmt.Println("connection closed. terminating...")
}
