package types

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/collections/codec"
)

type vrfParamsValueCodec struct{}

func (vrfParamsValueCodec) Encode(value VrfParams) ([]byte, error) {
	return json.Marshal(value)
}

func (vrfParamsValueCodec) Decode(b []byte) (VrfParams, error) {
	var v VrfParams
	if len(b) == 0 {
		return v, nil
	}
	err := json.Unmarshal(b, &v)
	return v, err
}

func (vrfParamsValueCodec) EncodeJSON(value VrfParams) ([]byte, error) {
	return json.Marshal(value)
}

func (vrfParamsValueCodec) DecodeJSON(b []byte) (VrfParams, error) {
	return vrfParamsValueCodec{}.Decode(b)
}

func (vrfParamsValueCodec) Stringify(value VrfParams) string {
	return fmt.Sprintf("VrfParams{enabled=%v, period=%d}", value.Enabled, value.PeriodSeconds)
}

func (vrfParamsValueCodec) ValueType() string {
	return "vrf/VrfParams"
}

type vrfBeaconValueCodec struct{}

func (vrfBeaconValueCodec) Encode(value VrfBeacon) ([]byte, error) {
	return json.Marshal(value)
}

func (vrfBeaconValueCodec) Decode(b []byte) (VrfBeacon, error) {
	var v VrfBeacon
	if len(b) == 0 {
		return v, nil
	}
	err := json.Unmarshal(b, &v)
	return v, err
}

func (vrfBeaconValueCodec) EncodeJSON(value VrfBeacon) ([]byte, error) {
	return json.Marshal(value)
}

func (vrfBeaconValueCodec) DecodeJSON(b []byte) (VrfBeacon, error) {
	return vrfBeaconValueCodec{}.Decode(b)
}

func (vrfBeaconValueCodec) Stringify(value VrfBeacon) string {
	return fmt.Sprintf("VrfBeacon{round=%d}", value.DrandRound)
}

func (vrfBeaconValueCodec) ValueType() string {
	return "vrf/VrfBeacon"
}

func VrfParamsValueCodec() codec.ValueCodec[VrfParams] {
	return vrfParamsValueCodec{}
}

func VrfBeaconValueCodec() codec.ValueCodec[VrfBeacon] {
	return vrfBeaconValueCodec{}
}

type vrfIdentityValueCodec struct{}

func (vrfIdentityValueCodec) Encode(value VrfIdentity) ([]byte, error) {
	return json.Marshal(value)
}

func (vrfIdentityValueCodec) Decode(b []byte) (VrfIdentity, error) {
	var v VrfIdentity
	if len(b) == 0 {
		return v, nil
	}
	err := json.Unmarshal(b, &v)
	return v, err
}

func (vrfIdentityValueCodec) EncodeJSON(value VrfIdentity) ([]byte, error) {
	return json.Marshal(value)
}

func (vrfIdentityValueCodec) DecodeJSON(b []byte) (VrfIdentity, error) {
	return vrfIdentityValueCodec{}.Decode(b)
}

func (vrfIdentityValueCodec) Stringify(value VrfIdentity) string {
	return fmt.Sprintf("VrfIdentity{validator=%s}", value.ValidatorAddress)
}

func (vrfIdentityValueCodec) ValueType() string {
	return "vrf/VrfIdentity"
}

func VrfIdentityValueCodec() codec.ValueCodec[VrfIdentity] {
	return vrfIdentityValueCodec{}
}
