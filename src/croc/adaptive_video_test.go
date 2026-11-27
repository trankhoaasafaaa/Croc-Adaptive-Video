package croc

import (
	"bytes"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/schollz/croc/v11/src/comm"
	"github.com/schollz/croc/v11/src/message"
)

func TestVideoProbeAcknowledgement(t *testing.T) {
	id := bytes.Repeat([]byte{1}, 16)
	key := make([]byte, 32)
	for _, tc := range []struct {
		name    string
		replyID []byte
		minimum int
		errText string
	}{
		{"valid", id, 360, ""},
		{"stale ID", make([]byte, 16), 360, "invalid probe acknowledgement"},
		{"invalid height", id, -1, "invalid probe acknowledgement"},
		{"timeout", nil, 0, "probe timed out"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			left, right := net.Pipe()
			defer left.Close()
			defer right.Close()
			client, peer := comm.New(left), comm.New(right)
			until := time.Now().Add(time.Second)
			if tc.replyID != nil {
				go func() {
					_ = peer.Send([]byte{1})
					_ = sendVideoControl(peer, key, message.Message{Type: typeVideoAck, Bytes2: tc.replyID, Num: tc.minimum}, until)
				}()
			} else {
				until = time.Now().Add(20 * time.Millisecond)
			}
			got, err := readVideoAck(client, key, id, until)
			if tc.errText == "" {
				if err != nil || got != tc.minimum {
					t.Fatalf("readVideoAck = %d, %v", got, err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.errText) {
				t.Fatalf("readVideoAck = %d, %v; want %q", got, err, tc.errText)
			}
		})
	}
}

func TestVideoProbeRejectsDuplicates(t *testing.T) {
	c := Client{video: videoState{probeReceived: true}}
	if err := c.receiveVideoProbe(message.Message{}); err == nil {
		t.Fatal("accepted a duplicate probe")
	}
	key := make([]byte, 32)
	c.Key = key
	ack, err := message.Encode(key, message.Message{Type: typeVideoAck, Bytes2: make([]byte, 16)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.processMessage(ack, nil); err == nil || !strings.Contains(err.Error(), "duplicate probe acknowledgement") {
		t.Fatalf("unexpected duplicate ACK result: %v", err)
	}
}
