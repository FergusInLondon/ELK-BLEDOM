package device

import (
	"context"
	"errors"
	"strings"
	"time"

	"tinygo.org/x/bluetooth"
)

const (
	// Service UUID - Sourced from DuoCol Android APK
	BLEDOM_ServiceUUID = "0000fff0-0000-1000-8000-00805f9b34fb"
	// Characteristic UUID - Sourced from DuoCol Android APK
	BLEDOM_CharacteristicUUID = "0000fff3-0000-1000-8000-00805f9b34fb"
	// Expected BLE Name Prefix - the device doesn't advertise it's true
	// services, so we identify the beginning of their advertised *name*
	BLEDOM_DeviceNamePrefix = "ELK"
)

var (
	// ErrScanTimeout occurs when the user provided timeout duration elapses
	// before the adapter has correctly identified a device from scanning.
	ErrScanTimeout = errors.New("timed out attempting to discover device")
	// ErrServiceNotAvailable occurs when the connected device - i.e. the one
	// identified via scanning that matches the expected device name - does not
	// implement the expected service UUID.
	ErrServiceNotAvailable = errors.New("device does not implement required service")
	// ErrCharacteristicNotAvailable occurs when the connected device *does*
	// implement the expected the service UUID, but the expected characteristic
	// UUID is not available.
	ErrCharacteristicNotAvailable = errors.New("unable to access required characteristic on device")
)

// Options contains general - and *optional* - configuration settings for the
// `BleDom` device. This includes things such as the Bluetooth adapter to use,
// or the parameters for the underlying Bluetooth connection attempt.
type Options struct {
	Adapter    *bluetooth.Adapter
	ConnParams bluetooth.ConnectionParams
}

// BleDom represents a ELK-BLEDOM device.
type BleDom struct {
	opts           *Options
	ctx            context.Context
	cancel         context.CancelFunc
	device         *bluetooth.Device
	characteristic bluetooth.DeviceCharacteristic
	pollCh         chan Poller
	commandCh      chan []byte
}

// NewBleDomRGBController creates a new `BleDom`, and - due to internal state -
// is the only way to actually create one.
func NewBleDomRGBController(parentCtx context.Context, opts Options) *BleDom {
	ctx, cancel := context.WithCancel(parentCtx)
	return &BleDom{
		opts: &opts, ctx: ctx, cancel: cancel,
		pollCh: make(chan Poller), commandCh: make(chan []byte),
	}
}

// Connect handles the initial connection to the BLE device, it may return an
// error for any one of it's multiple failure points. After Connect has been
// called (and ran successfully without error) the device will be ready for
// sending commands to.
func (b *BleDom) Connect(timeout time.Duration) error {
	if b.opts.Adapter == nil {
		b.opts.Adapter = bluetooth.DefaultAdapter
	}

	// Enable adapter
	if err := b.opts.Adapter.Enable(); err != nil {
		return err
	}

	devCh := make(chan bluetooth.ScanResult, 1)
	if err := b.opts.Adapter.Scan(
		func(adapter *bluetooth.Adapter, res bluetooth.ScanResult) {
			if strings.Index(res.LocalName(), BLEDOM_DeviceNamePrefix) == 0 {
				adapter.StopScan()
				devCh <- res
			}
		}); err != nil {
		return err
	}

	select {
	case res := <-devCh:
		device, err := b.opts.Adapter.Connect(res.Address, b.opts.ConnParams)
		if err != nil {
			return err
		}

		b.device = device
	case <-time.After(timeout):
		if b.device == nil {
			b.opts.Adapter.StopScan()
			return ErrScanTimeout
		}
	}

	serviceUUID, _ := bluetooth.ParseUUID(BLEDOM_ServiceUUID)
	services, err := b.device.DiscoverServices([]bluetooth.UUID{serviceUUID})
	if err != nil {
		return err
	} else if len(services) == 0 {
		return ErrServiceNotAvailable
	}

	characteristicUUID, _ := bluetooth.ParseUUID(BLEDOM_CharacteristicUUID)
	characteristics, err := services[0].DiscoverCharacteristics([]bluetooth.UUID{
		characteristicUUID,
	})
	if err != nil {
		return err
	} else if len(characteristics) == 0 {
		return ErrCharacteristicNotAvailable
	}

	b.characteristic = characteristics[0]
	go b.interact()
	return nil
}

// Poller is a consumer-provided callback function used for handling state
// monitoring via `PollState`.
type Poller func([]byte)

// PollState will call `cb` at an interval of `d`, providing access to the
// latest state of the connected device.
func (b *BleDom) PollState(d time.Duration, cb Poller) {
	go func() {
		t := time.NewTimer(d)

		for {
			select {
			case <-b.ctx.Done():
				t.Stop()
				return
			case <-t.C:
				b.pollCh <- cb
				t.Reset(d)
			}
		}
	}()
}

// Command is an interface used internally by the package, allowing different
// Command types (i.e. BrightnessCommand or ColourCommand) to generate their
// own byte payloads.
type Command interface {
	raw() []byte
}

// WriteCommand accepts a Command object and dispatches it to the device.
func (b *BleDom) WriteCommand(cmd Command) {
	b.commandCh <- cmd.raw()
}

// Stop terminates the device connection and prevents any further operations.
// Note that any pending operations may still execute.
func (b *BleDom) Stop() {
	b.cancel()
}

// Done blocks until the device connection has been gracefully terminated.
func (b *BleDom) Done() {
	<-b.ctx.Done()
}

func (b *BleDom) interact() {
	for {
		select {
		case <-b.ctx.Done():
			b.device.Disconnect()
			return
		case pollReq := <-b.pollCh:
			data := make([]byte, 16)
			b.characteristic.Read(data)
			pollReq(data)
		case cmd := <-b.commandCh:
			b.characteristic.WriteWithoutResponse(cmd)
		}
	}
}
