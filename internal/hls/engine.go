package hls

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const liveWindowSize = 3

type ChannelInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
}

type ChannelStream struct {
	Info         ChannelInfo
	allSegments  []string
	windowSize   int
	window       []string
	mu           sync.RWMutex
	stopChan     chan struct{}
	mediaSeq     uint64
	currentIndex int
}

func NewChannelStream(info ChannelInfo, segments []string) *ChannelStream {
	if len(segments) == 0 {
		segments = generarSegmentosSinteticos(64)
	}

	// Each source segment lasts 10 seconds; keep exactly 3 for a 30-second live window.
	windowSize := 3
	if len(segments) < windowSize {
		windowSize = len(segments)
	}

	initialWindow := make([]string, windowSize)
	copy(initialWindow, segments[:windowSize])

	return &ChannelStream{
		Info:         info,
		allSegments:  segments,
		windowSize:   windowSize,
		window:       initialWindow,
		stopChan:     make(chan struct{}),
		mediaSeq:     1,
		currentIndex: windowSize - 1,
	}
}

func (cs *ChannelStream) StartSlidingWindow(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				cs.rotateWindow()
			case <-cs.stopChan:
				ticker.Stop()
				return
			}
		}
	}()
}

func (cs *ChannelStream) Stop() {
	close(cs.stopChan)
}

func (cs *ChannelStream) rotateWindow() {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.currentIndex = (cs.currentIndex + 1) % len(cs.allSegments)
	nextSegment := cs.allSegments[cs.currentIndex]

	cs.window = append(cs.window[1:], nextSegment)
	cs.mediaSeq++
}

func (cs *ChannelStream) GenerateM3U8() string {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	var sb strings.Builder
	sb.WriteString("#EXTM3U\n")
	sb.WriteString("#EXT-X-VERSION:3\n")
	sb.WriteString("#EXT-X-TARGETDURATION:10\n")
	sb.WriteString(fmt.Sprintf("#EXT-X-MEDIA-SEQUENCE:%d\n", cs.mediaSeq))

	for _, seg := range cs.window {
		sb.WriteString("#EXTINF:10.0,\n")
		sb.WriteString(fmt.Sprintf("/api/stream/segments/%s\n", seg))
	}

	return sb.String()
}

type StreamManager struct {
	segmentsDir  string
	channels     map[string]*ChannelStream
	segmentCache sync.Map
	bufPool      sync.Pool
	mu           sync.RWMutex
}

func NewStreamManager(segmentsDir string) (*StreamManager, error) {
	sm := &StreamManager{
		segmentsDir: segmentsDir,
		channels:    make(map[string]*ChannelStream),
		bufPool: sync.Pool{
			New: func() interface{} {
				return bytes.NewBuffer(make([]byte, 0, 4096))
			},
		},
	}

	segmentos, err := cargarSegmentosDirectorio(segmentsDir)
	if err != nil || len(segmentos) == 0 {
		segmentos = generarSegmentosSinteticos(64)
	}

	canalesIniciales := []ChannelInfo{
		{ID: "kuspid-sports", Name: "Kuspid Sports HD", Category: "Deportes", Description: "Transmisión en directo 24/7 de eventos deportivos de alta intensidad."},
		{ID: "kuspid-cinema", Name: "Kuspid Cinema 4K", Category: "Cine", Description: "Películas taquilleras y producciones internacionales en formato HLS."},
		{ID: "kuspid-tech", Name: "Kuspid Tech & Gaming", Category: "Tecnología", Description: "Torneos eSports, análisis de software y emisiones en vivo de tecnología."},
	}

	for _, info := range canalesIniciales {
		chStream := NewChannelStream(info, segmentos)
		chStream.StartSlidingWindow(10 * time.Second)
		sm.channels[info.ID] = chStream
	}

	return sm, nil
}

func (sm *StreamManager) GetChannel(id string) (*ChannelStream, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	ch, ok := sm.channels[id]
	if !ok {
		ch, ok = sm.channels["kuspid-sports"]
	}
	return ch, ok
}

func (sm *StreamManager) ListChannels() []ChannelInfo {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	list := make([]ChannelInfo, 0, len(sm.channels))
	for _, ch := range sm.channels {
		list = append(list, ch.Info)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].ID < list[j].ID
	})

	return list
}

func (sm *StreamManager) GetSegmentContent(filename string) ([]byte, error) {
	cleanName := filepath.Base(filename)

	if cachedData, ok := sm.segmentCache.Load(cleanName); ok {
		return cachedData.([]byte), nil
	}

	fullPath := filepath.Join(sm.segmentsDir, cleanName)
	data, err := os.ReadFile(fullPath)
	if err == nil {
		sm.segmentCache.Store(cleanName, data)
		return data, nil
	}

	syntheticData := generarBufferSegmentoSintetico(cleanName)
	sm.segmentCache.Store(cleanName, syntheticData)
	return syntheticData, nil
}

func cargarSegmentosDirectorio(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var segs []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".ts") {
			segs = append(segs, entry.Name())
		}
	}

	sort.Slice(segs, func(i, j int) bool {
		leftIndex, leftIsNumbered := segmentIndex(segs[i])
		rightIndex, rightIsNumbered := segmentIndex(segs[j])

		switch {
		case leftIsNumbered && rightIsNumbered && leftIndex != rightIndex:
			return leftIndex < rightIndex
		case leftIsNumbered != rightIsNumbered:
			return leftIsNumbered
		default:
			return segs[i] < segs[j]
		}
	})
	return segs, nil
}

// segmentIndex obtains the trailing number in names such as segment2.ts or segment_2.ts.
// Files without a trailing number fall back to a stable lexical ordering.
func segmentIndex(filename string) (int, bool) {
	baseName := strings.TrimSuffix(filename, filepath.Ext(filename))
	end := len(baseName)
	start := end

	for start > 0 && baseName[start-1] >= '0' && baseName[start-1] <= '9' {
		start--
	}

	if start == end {
		return 0, false
	}

	index, err := strconv.Atoi(baseName[start:])
	if err != nil {
		return 0, false
	}

	return index, true
}

func generarSegmentosSinteticos(count int) []string {
	segs := make([]string, count)
	for i := 0; i < count; i++ {
		segs[i] = fmt.Sprintf("segment%d.ts", i)
	}
	return segs
}

func generarBufferSegmentoSintetico(filename string) []byte {
	buf := make([]byte, 188*10)
	for i := 0; i < len(buf); i += 188 {
		buf[i] = 0x47
		buf[i+1] = 0x1F
		buf[i+2] = 0xFF
		buf[i+3] = 0x10
	}
	return buf
}
