package tron

import (
	"testing"
)

func TestBase58ToHexRoundtrip(t *testing.T) {
	// tron:nile USDT
	base58 := "TXYZopYRdj2D9XRtbG411XZZ3kM5VkAeBf"
	hex, err := Base58ToHex(base58)
	if err != nil {
		t.Fatal(err)
	}
	if len(hex) != 42 || hex[:2] != "0x" {
		t.Errorf("expected 0x+40 hex, got %q", hex)
	}
	back, err := HexToBase58(hex)
	if err != nil {
		t.Fatal(err)
	}
	if back != base58 {
		t.Errorf("roundtrip: got %q want %q", back, base58)
	}
}
