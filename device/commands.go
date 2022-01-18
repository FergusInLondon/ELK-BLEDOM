package device

import "encoding/hex"

type ColourCommand struct {
	Red   uint8
	Green uint8
	Blue  uint8
}

func (c *ColourCommand) raw() []byte {
	return []byte{0x7E, 0x00, 0x05, 0x03, c.Red, c.Green, c.Blue, 0x00, 0xEF}
}

func ColourCommandFromHex(h string) *ColourCommand {
	if len(h) == 6 {
		b, _ := hex.DecodeString(h)
		return &ColourCommand{b[0], b[1], b[2]}
	}

	return &ColourCommand{}
}

func ColourCommandFromValues(red, green, blue uint8) *ColourCommand {
	return &ColourCommand{red, green, blue}
}

type BrightnessCommand struct {
	Brightness uint8
}

func (c *BrightnessCommand) raw() []byte {
	return []byte{0x7E, 0x00, 0x01, c.Brightness, 0x00, 0x00, 0x00, 0x00, 0xEF}
}

func BrightnessCommandFromValue(brightness uint8) *BrightnessCommand {
	return &BrightnessCommand{brightness}
}

func BrightnessCommandFromPercentage(percent uint8) *BrightnessCommand {
	if percent > 100 {
		return &BrightnessCommand{0x80}
	}

	return &BrightnessCommand{uint8(percent * (0x80 / 100))}
}
