package croc

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"os"
	"path"
	"strings"
	"time"

	"github.com/schollz/croc/v11/src/adaptivevideo"
	"github.com/schollz/croc/v11/src/comm"
	"github.com/schollz/croc/v11/src/message"
)

const (
	videoProbeFrameLimit              = 512 << 10
	videoProbeWait                    = 10 * time.Second
	videoFeature                      = "adaptive-video-v1"
	videoProbeBytes                   = 256 << 10
	videoProbeTimeout                 = 8 * time.Second
	videoAckLimit                     = 4096
	videoHashAlgorithm                = "highway"
	typeVideoProbe       message.Type = "adaptive-video-probe"
	typeVideoAck         message.Type = "adaptive-video-ack"
)

type videoOffer struct {
	Height int
	Bytes  int64
	ETA    time.Duration
	Late   bool
}

type videoState struct {
	probeDeadline time.Time
	peerCapable   bool
	probeReceived bool
	offer         *videoOffer
}

// videoFailure's text can be disclosed to the peer because callers supply
// fixed protocol diagnostics. Local I/O details stay on the local endpoint.
type videoFailure string

func (e videoFailure) Error() string { return "adaptive video: " + string(e) }

// ErrVideoCanceled lets the CLI report canceled adaptive transfers as failures.
var ErrVideoCanceled error = videoFailure("transfer canceled")

func (c *Client) videoActive() bool {
	return c.Options.VideoSend != nil || c.video.probeReceived || c.video.peerCapable && !c.Options.IsSender
}

func (c *Client) videoExit(err error) error {
	if (c.videoActive() || c.Options.VideoMinimum != 0) && errors.Is(err, context.Canceled) {
		return ErrVideoCanceled
	}
	return c.redactError(err)
}

func (c *Client) awaitingVideoProbe() bool {
	return !c.Options.IsSender && c.video.peerCapable && !c.video.probeReceived && c.lifecycleSnapshot().ChannelSecured
}

func (c *Client) sendVideoFailure(control *comm.Comm, err error) {
	var failure videoFailure
	if !errors.As(err, &failure) {
		failure = "session interrupted before completion"
	}
	_ = sendVideoControl(control, c.Key, message.Message{Type: message.TypeError, Message: failure.Error()}, time.Now().Add(time.Second))
}

func sendVideoControl(control *comm.Comm, key []byte, m message.Message, until time.Time) error {
	frame, err := message.Encode(key, m)
	if err != nil {
		return err
	}
	return writeVideoFrame(control, frame, until)
}

func writeVideoFrame(control *comm.Comm, frame []byte, until time.Time) error {
	conn := control.Connection()
	if err := conn.SetWriteDeadline(until); err != nil {
		return err
	}
	defer conn.SetWriteDeadline(time.Time{})
	return control.Send(frame)
}

func (c *Client) receiveVideoProbe(m message.Message) error {
	if !c.awaitingVideoProbe() ||
		c.lifecycleSnapshot().FileInfoTransferred || len(m.Bytes) != videoProbeBytes || len(m.Bytes2) != 16 {
		return videoFailure("unexpected or invalid probe")
	}
	c.video.probeReceived = true
	// Disable automatic reconnect across attempts for this protocol version.
	c.reconnectVersion = 0
	return sendVideoControl(c.connection(0), c.Key, message.Message{
		Type: typeVideoAck, Bytes2: m.Bytes2, Num: c.Options.VideoMinimum,
	}, time.Now().Add(videoProbeTimeout))
}

func readVideoAck(control *comm.Comm, key, id []byte, until time.Time) (int, error) {
	for {
		frame, err := control.ReceiveWithDeadlineLimit(until, videoAckLimit)
		if err != nil {
			var networkErr net.Error
			if errors.As(err, &networkErr) && networkErr.Timeout() {
				return 0, videoFailure("probe timed out")
			}
			if errors.Is(err, comm.ErrMessageTooLarge) {
				return 0, videoFailure("probe acknowledgement exceeded its frame limit")
			}
			return 0, fmt.Errorf("adaptive video: connection lost during probe: %w", err)
		}
		if bytes.Equal(frame, []byte{1}) {
			continue
		}
		m, err := message.DecodeWithLimit(key, frame, videoAckLimit)
		if err != nil || m.Type != typeVideoAck || !bytes.Equal(m.Bytes2, id) ||
			len(m.Bytes) != 0 || m.Num < 0 || m.Num > adaptivevideo.MaxHeight {
			return 0, videoFailure("invalid probe acknowledgement")
		}
		return m.Num, nil
	}
}

func (c *Client) prepareAdaptiveVideo() error {
	req := c.Options.VideoSend
	if !c.video.peerCapable {
		return videoFailure("incompatible peer; use a Croc Adaptive Video receiver")
	}
	sample := make([]byte, videoProbeBytes+16)
	if _, err := rand.Read(sample); err != nil {
		return fmt.Errorf("adaptive video: generate probe: %w", err)
	}
	id, payload := sample[:16], sample[16:]
	frame, err := message.Encode(c.Key, message.Message{Type: typeVideoProbe, Bytes: payload, Bytes2: id})
	if err != nil {
		return err
	}
	control := c.connection(0)
	start := time.Now()
	until := start.Add(min(videoProbeTimeout, req.Deadline))
	if err := writeVideoFrame(control, frame, until); err != nil {
		var networkErr net.Error
		if errors.As(err, &networkErr) && networkErr.Timeout() {
			return videoFailure("probe timed out")
		}
		return fmt.Errorf("adaptive video: send probe: %w", err)
	}
	// updateState is invoked by the control owner after its outstanding read
	// has completed, so only this bounded read can consume the probe ACK.
	minimum, err := readVideoAck(control, c.Key, id, until)
	if err != nil {
		return err
	}
	elapsed := time.Since(start)
	rawRate := videoProbeBytes / elapsed.Seconds()
	safeRate := adaptivevideo.SafetyFactor * rawRate
	fmt.Fprintf(os.Stderr, "Adaptive video probe bytes=%d elapsed=%s raw_Bps=%.3f safe_Bps=%.3f minimum=%d\n",
		videoProbeBytes, elapsed, rawRate, safeRate, minimum)
	variants := make([]adaptivevideo.Variant, 0, len(req.Sources))
	for _, source := range req.Sources {
		variants = append(variants, source.Variant)
	}
	selection, err := adaptivevideo.Select(variants, minimum, safeRate, max(req.Deadline-elapsed, 0))
	switch {
	case errors.Is(err, adaptivevideo.ErrNoFit):
		return videoFailure("no variant fits the video deadline")
	case errors.Is(err, adaptivevideo.ErrNoQuality):
		return videoFailure("no variant meets the receiver's minimum height")
	case err != nil:
		return err
	}
	for _, source := range req.Sources {
		if source.Height != selection.Height {
			continue
		}
		if err := source.CheckUnchanged(); err != nil {
			return err
		}
		files, folders, count, err := GetFilesInfoContext(c.clientContext(), []string{source.Path}, false, false, nil)
		if err != nil {
			return err
		}
		if len(files) != 1 || len(folders) != 0 || count != 0 || files[0].Size != selection.Bytes || !files[0].Mode.IsRegular() {
			return videoFailure("selected source changed before preparation")
		}
		if err := c.sendCollectFiles(files); err != nil {
			return err
		}
		if err := source.CheckUnchanged(); err != nil {
			return err
		}
		c.TotalNumberOfContents = 1
		c.video.offer = &videoOffer{Height: selection.Height, Bytes: selection.Bytes, ETA: selection.ETA, Late: selection.Late}
		fmt.Fprintf(os.Stderr, "Adaptive video selected height=%d bytes=%d eta=%s late=%t\n",
			selection.Height, selection.Bytes, selection.ETA, selection.Late)
		return nil
	}
	return errors.New("adaptive video: selected source unavailable")
}

func (c *Client) validateVideoOffer(info SenderInfo) error {
	offer := info.Video
	if offer == nil {
		if c.Options.VideoMinimum > 0 || c.video.probeReceived || c.video.peerCapable {
			return videoFailure("expected an adaptive video offer")
		}
		return nil
	}
	if !c.video.probeReceived || offer.Height <= 0 || offer.Height > adaptivevideo.MaxHeight ||
		offer.Height < c.Options.VideoMinimum || offer.Bytes <= 0 || offer.ETA < 0 ||
		(offer.Late && c.Options.VideoMinimum == 0) ||
		len(c.FilesToTransfer) != 1 || len(c.EmptyFoldersToTransfer) != 0 || info.TotalNumberFolders != 0 ||
		info.SendingText || info.ReconnectVersion != 0 || info.NextReconnectRoom != "" ||
		info.HashAlgorithm != videoHashAlgorithm || c.Options.Stdout {
		return videoFailure("invalid adaptive video offer")
	}
	file := c.FilesToTransfer[0]
	if file.Size != offer.Bytes || !file.Mode.IsRegular() || file.Symlink != "" || file.TempFile ||
		!strings.EqualFold(path.Ext(file.Name), ".mp4") || len(file.Hash) != 32 {
		return videoFailure("file manifest does not match the adaptive video offer")
	}
	c.video.offer = offer
	return nil
}
