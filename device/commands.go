package device

import "encoding/hex"

// ColourCommand provides updates to the LED controller with new RGB values.
// These values are contained as a uint8/byte types.
type ColourCommand struct {
	Red   uint8
	Green uint8
	Blue  uint8
}

func (c *ColourCommand) raw() []byte {
	return []byte{0x7E, 0x00, 0x05, 0x03, c.Red, c.Green, c.Blue, 0x00, 0xEF}
}

// ColourCommandFromHex accepts a hex string - i.e. "FF00EE" - and generates
// a valid `ColourCommand`.
func ColourCommandFromHex(h string) *ColourCommand {
	if len(h) == 6 {
		b, _ := hex.DecodeString(h)
		return &ColourCommand{b[0], b[1], b[2]}
	}

	return &ColourCommand{}
}

// ColourCommandFromValues accepts raw RGB byte values, and generates a valid
// `ColourCommand`.
func ColourCommandFromValues(red, green, blue uint8) *ColourCommand {
	return &ColourCommand{red, green, blue}
}

// BrightnessCommand provides control of the output brightness for the managed
// LEDs.
type BrightnessCommand struct {
	Brightness uint8
}

func (c *BrightnessCommand) raw() []byte {
	return []byte{0x7E, 0x00, 0x01, c.Brightness, 0x00, 0x00, 0x00, 0x00, 0xEF}
}

// BrightnessCommandFromValue accepts a raw uint8/byte representing the desired
// brightness. Range of values is 0-100.
func BrightnessCommandFromValue(brightness uint8) *BrightnessCommand {
	if brightness > 100 {
		brightness = 100
	}

	return &BrightnessCommand{brightness}
}
