// SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

//go:build !js
// +build !js

package webrtc

import (
	"testing"
	"time"

	"github.com/pion/srtp/v3"
	"github.com/stretchr/testify/assert"
)

// A connection with only data channels starts no SRTP sessions and no
// undeclared media processors.
func TestDataChannelOnlyStartsNoMedia(t *testing.T) {
	pcOffer, pcAnswer, err := newPair()
	assert.NoError(t, err)
	defer closePairNow(t, pcOffer, pcAnswer)

	got := make(chan struct{})
	dc, err := pcOffer.CreateDataChannel("data", nil)
	assert.NoError(t, err)
	dc.OnOpen(func() { assert.NoError(t, dc.SendText("hi")) })
	pcAnswer.OnDataChannel(func(d *DataChannel) {
		d.OnMessage(func(DataChannelMessage) { close(got) })
	})

	assert.NoError(t, signalPair(pcOffer, pcAnswer))
	select {
	case <-got:
	case <-time.After(10 * time.Second):
		t.Fatal("no data channel message")
	}

	for _, pc := range []*PeerConnection{pcOffer, pcAnswer} {
		_, srtpStarted := pc.dtlsTransport.srtpSession.Load().(*srtp.SessionSRTP)
		assert.False(t, srtpStarted, "SRTP session started")
		assert.False(t, pc.undeclaredStarted.Load(), "undeclared media processors started")
	}
}

// A connection with a track still starts SRTP on both ends.
func TestMediaStartsSRTP(t *testing.T) {
	pcOffer, pcAnswer, err := newPair()
	assert.NoError(t, err)
	defer closePairNow(t, pcOffer, pcAnswer)

	track, err := NewTrackLocalStaticSample(RTPCodecCapability{MimeType: MimeTypeOpus}, "audio", "pion")
	assert.NoError(t, err)
	_, err = pcOffer.AddTrack(track)
	assert.NoError(t, err)

	assert.NoError(t, signalPair(pcOffer, pcAnswer))
	for _, pc := range []*PeerConnection{pcOffer, pcAnswer} {
		select {
		case <-pc.dtlsTransport.srtpReady:
		case <-time.After(10 * time.Second):
			t.Fatal("SRTP never started")
		}
		assert.True(t, pc.undeclaredStarted.Load())
	}
}
