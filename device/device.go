package device

import (
	"context"
	"errors"
	"strings"
	"time"

	"tinygo.org/x/bluetooth"
)

const (
	BLEDOM_ServiceUUID        = "0000fff0-0000-1000-8000-00805f9b34fb"
	BLEDOM_CharacteristicUUID = "0000fff3-0000-1000-8000-00805f9b34fb"
	BLEDOM_DeviceNamePrefix   = "ELK"
)

var (
	ErrScanTimeout                = errors.New("timed out attempting to discover device")
	ErrServiceNotAvailable        = errors.New("device does not implement required service")
	ErrCharacteristicNotAvailable = errors.New("unable to access required characteristic on device")
)

type Options struct {
	Adapter    *bluetooth.Adapter
	ConnParams bluetooth.ConnectionParams
}

type BleDom struct {
	opts           *Options
	ctx            context.Context
	cancel         context.CancelFunc
	device         *bluetooth.Device
	characteristic bluetooth.DeviceCharacteristic
	pollCh         chan Poller
	commandCh      chan []byte
}

func NewBleDomRGBController(parentCtx context.Context, opts Options) *BleDom {
	ctx, cancel := context.WithCancel(parentCtx)
	return &BleDom{
		opts: &opts, ctx: ctx, cancel: cancel,
		pollCh: make(chan Poller), commandCh: make(chan []byte),
	}
}

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

type Poller func([]byte)

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

type Command interface {
	raw() []byte
}

func (b *BleDom) WriteCommand(cmd Command) {
	b.commandCh <- cmd.raw()
}

func (b *BleDom) Stop() {
	b.cancel()
}

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
