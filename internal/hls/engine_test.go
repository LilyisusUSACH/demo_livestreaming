package hls

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestChannelStream_GenerateM3U8(t *testing.T) {
	info := ChannelInfo{
		ID:       "test-channel",
		Name:     "Test Channel",
		Category: "Test",
	}

	segments := []string{"segment0.ts", "segment1.ts", "segment2.ts", "segment3.ts"}

	ch := NewChannelStream(info, segments)
	m3u8Str := ch.GenerateM3U8()

	if !strings.Contains(m3u8Str, "#EXTM3U") {
		t.Errorf("Expected #EXTM3U header in m3u8 playlist")
	}

	if !strings.Contains(m3u8Str, "#EXT-X-MEDIA-SEQUENCE:1") {
		t.Errorf("Expected initial EXT-X-MEDIA-SEQUENCE:1")
	}
	if count := strings.Count(m3u8Str, "#EXTINF:10.0,"); count != 3 {
		t.Fatalf("Expected exactly 3 segments (30 seconds), got %d", count)
	}
	if strings.Index(m3u8Str, "segment0.ts") > strings.Index(m3u8Str, "segment1.ts") ||
		strings.Index(m3u8Str, "segment1.ts") > strings.Index(m3u8Str, "segment2.ts") {
		t.Errorf("Expected the initial playlist to start in sequence")
	}

	// Test rotation
	ch.rotateWindow()
	m3u8Str = ch.GenerateM3U8()

	if !strings.Contains(m3u8Str, "#EXT-X-MEDIA-SEQUENCE:2") {
		t.Errorf("Expected EXT-X-MEDIA-SEQUENCE:2 after rotation")
	}
	if strings.Index(m3u8Str, "segment1.ts") > strings.Index(m3u8Str, "segment2.ts") ||
		strings.Index(m3u8Str, "segment2.ts") > strings.Index(m3u8Str, "segment3.ts") {
		t.Errorf("Expected rotation to remove the oldest segment and append the next one")
	}
}

func TestCargarSegmentosDirectorio_OrdenaNumericamente(t *testing.T) {
	dir := t.TempDir()
	for _, filename := range []string{"segment10.ts", "segment2.ts", "segment1.ts", "segment0.ts"} {
		if err := os.WriteFile(filepath.Join(dir, filename), []byte("test"), 0o600); err != nil {
			t.Fatalf("Failed to create fixture %s: %v", filename, err)
		}
	}

	segments, err := cargarSegmentosDirectorio(dir)
	if err != nil {
		t.Fatalf("Failed to load segments: %v", err)
	}

	want := []string{"segment0.ts", "segment1.ts", "segment2.ts", "segment10.ts"}
	if strings.Join(segments, ",") != strings.Join(want, ",") {
		t.Errorf("Expected numeric segment order %v, got %v", want, segments)
	}
}

func TestStreamManager_GetChannel(t *testing.T) {
	sm, err := NewStreamManager("media/segments")
	if err != nil {
		t.Fatalf("Failed to initialize StreamManager: %v", err)
	}

	ch, ok := sm.GetChannel("kuspid-sports")
	if !ok || ch == nil {
		t.Errorf("Expected to find default channel kuspid-sports")
	}

	channels := sm.ListChannels()
	if len(channels) == 0 {
		t.Errorf("Expected non-empty list of channels")
	}
}

func TestSlidingWindowTicker(t *testing.T) {
	info := ChannelInfo{ID: "ticker-channel", Name: "Ticker", Category: "Test"}
	segments := []string{"seg0.ts", "seg1.ts", "seg2.ts", "seg3.ts"}

	ch := NewChannelStream(info, segments)
	ch.StartSlidingWindow(50 * time.Millisecond)

	time.Sleep(120 * time.Millisecond)

	ch.mu.RLock()
	seq := ch.mediaSeq
	ch.mu.RUnlock()

	if seq <= 1 {
		t.Errorf("Expected media sequence to increase after ticker runs, got %d", seq)
	}

	ch.Stop()
}
