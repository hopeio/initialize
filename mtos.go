package initialize

import (
	"github.com/go-viper/mapstructure/v2"
)

var decodeHook = mapstructure.ComposeDecodeHookFunc(
	mapstructure.StringToTimeDurationHookFunc(),
	mapstructure.TextUnmarshallerHookFunc(),
	mapstructure.StringToSliceHookFunc(","),
)

type DecoderConfigOption func(*mapstructure.DecoderConfig)

// defaultDecoderConfig returns a mapstructure.DecoderConfig with weak typing, squash, and standard decode hooks.
func defaultDecoderConfig(output any, opts ...DecoderConfigOption) *mapstructure.DecoderConfig {
	c := &mapstructure.DecoderConfig{
		Metadata:         nil,
		Result:           output,
		WeaklyTypedInput: true,
		Squash:           true,
		DecodeHook:       decodeHook,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Decode decodes a map[string]any into dst using mapstructure with the default decoder options.
func Decode(dst any, mapData map[string]any, opts ...DecoderConfigOption) error {
	config := defaultDecoderConfig(dst, opts...)
	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return err
	}
	return decoder.Decode(mapData)
}
