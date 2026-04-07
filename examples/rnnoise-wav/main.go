// Command rnnoise-wav denoises a 48 kHz mono 16-bit PCM WAV using pkg/rnnoise.
//
// Real denoising requires Xiph librnnoise installed (see pkg/rnnoise/README.md).
// Do not use the Homebrew VST cask named rnnoise for development.
//
//	CGO_ENABLED=1 go run -tags rnnoise ./examples/rnnoise-wav/ -- in.wav out.wav
//
// Stub build (copies input to output, exits 0; useful for CI / no library):
//
//	go run ./examples/rnnoise-wav/ -- in.wav out.wav
package main

import (
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"

	"github.com/LingByte/SoulNexus/pkg/rnnoise"
)

func main() {
	verbose := flag.Bool("v", false, "print PCM difference stats to stderr (use to verify processing)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [-v] [--] in.wav out.wav\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Input must be PCM WAV: 48000 Hz, mono, 16-bit.\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	args := flag.Args()
	if len(args) != 2 {
		flag.Usage()
		os.Exit(2)
	}
	inPath, outPath := args[0], args[1]

	if err := run(inPath, outPath, *verbose); err != nil {
		fmt.Fprintf(os.Stderr, "rnnoise-wav: %v\n", err)
		os.Exit(1)
	}
}

func run(inPath, outPath string, verbose bool) error {
	raw, err := os.ReadFile(inPath)
	if err != nil {
		return err
	}
	pcm, err := pcm16Mono48kFromWAV(raw)
	if err != nil {
		return err
	}

	d, err := rnnoise.New()
	if err != nil {
		if errors.Is(err, rnnoise.ErrUnavailable) {
			fmt.Fprintf(os.Stderr, "note: built without rnnoise tag — copying input to output.\n")
			fmt.Fprintf(os.Stderr, "  For real denoise: CGO_ENABLED=1 go run -tags rnnoise ./examples/rnnoise-wav/ -- %s %s\n", inPath, outPath)
			if verbose {
				printPCMStats("stub (passthrough)", pcm, pcm)
			}
			return writePCM16WAV(outPath, pcm)
		}
		return err
	}
	defer d.Close()

	outPCM := d.Process(pcm)
	if verbose {
		printPCMStats("rnnoise", pcm, outPCM)
	}
	return writePCM16WAV(outPath, outPCM)
}

func printPCMStats(label string, inPCM, outPCM []byte) {
	n := len(inPCM) / 2
	if len(inPCM) != len(outPCM) {
		fmt.Fprintf(os.Stderr, "rnnoise-wav [%s]: length mismatch in=%d out=%d\n", label, len(inPCM), len(outPCM))
		return
	}
	var (
		diffSamples int
		maxAbs      int
		sumSq       float64
	)
	for i := 0; i < n; i++ {
		sIn := int16(binary.LittleEndian.Uint16(inPCM[i*2:]))
		sOut := int16(binary.LittleEndian.Uint16(outPCM[i*2:]))
		d := int(sIn) - int(sOut)
		sumSq += float64(d) * float64(d)
		if d != 0 {
			diffSamples++
			ad := d
			if ad < 0 {
				ad = -ad
			}
			if ad > maxAbs {
				maxAbs = ad
			}
		}
	}
	rms := math.Sqrt(sumSq / float64(max(n, 1)))
	fmt.Fprintf(os.Stderr, "rnnoise-wav [%s]: samples=%d changed=%d (%.2f%%) max_abs_diff=%d rms_diff≈%.1f (int16 units; 0=identical)\n",
		label, n, diffSamples, 100*float64(diffSamples)/float64(n), maxAbs, rms)
}

func writePCM16WAV(path string, pcm []byte) error {
	const (
		sampleRate   = 48000
		numChannels  = 1
		bitsPerSample = 16
	)
	blockAlign := numChannels * bitsPerSample / 8
	byteRate := uint32(sampleRate * blockAlign)
	dataSize := uint32(len(pcm))
	riffChunkSize := 36 + dataSize

	out, err := os.Create(path)
	if err != nil {
		return err
	}

	// RIFF header + fmt + data
	hdr := make([]byte, 44)
	copy(hdr[0:4], "RIFF")
	binary.LittleEndian.PutUint32(hdr[4:8], riffChunkSize)
	copy(hdr[8:12], "WAVE")
	copy(hdr[12:16], "fmt ")
	binary.LittleEndian.PutUint32(hdr[16:20], 16)
	binary.LittleEndian.PutUint16(hdr[20:22], 1)
	binary.LittleEndian.PutUint16(hdr[22:24], numChannels)
	binary.LittleEndian.PutUint32(hdr[24:28], sampleRate)
	binary.LittleEndian.PutUint32(hdr[28:32], byteRate)
	binary.LittleEndian.PutUint16(hdr[32:34], uint16(blockAlign))
	binary.LittleEndian.PutUint16(hdr[34:36], bitsPerSample)
	copy(hdr[36:40], "data")
	binary.LittleEndian.PutUint32(hdr[40:44], dataSize)
	if _, err := out.Write(hdr); err != nil {
		return err
	}
	if _, err := out.Write(pcm); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func pcm16Mono48kFromWAV(b []byte) ([]byte, error) {
	if len(b) < 12 || string(b[0:4]) != "RIFF" || string(b[8:12]) != "WAVE" {
		return nil, errors.New("need RIFF WAVE file")
	}
	var (
		channels   uint16
		rate       uint32
		bits       uint16
		pcm        []byte
		gotFmt     bool
	)
	off := 12
	for off+8 <= len(b) {
		id := string(b[off : off+4])
		sz := int(binary.LittleEndian.Uint32(b[off+4 : off+8]))
		payload := off + 8
		end := payload + sz
		if end > len(b) {
			return nil, errors.New("truncated WAV")
		}
		switch id {
		case "fmt ":
			if sz < 16 {
				return nil, errors.New("bad fmt chunk")
			}
			audio := binary.LittleEndian.Uint16(b[payload : payload+2])
			if audio != 1 {
				return nil, fmt.Errorf("need PCM (format 1), got %d", audio)
			}
			channels = binary.LittleEndian.Uint16(b[payload+2 : payload+4])
			rate = binary.LittleEndian.Uint32(b[payload+4 : payload+8])
			bits = binary.LittleEndian.Uint16(b[payload+14 : payload+16])
			gotFmt = true
		case "data":
			pcm = b[payload:end]
		}
		off = end
		if sz%2 == 1 {
			off++
		}
	}
	if !gotFmt || pcm == nil {
		return nil, errors.New("WAV missing fmt or data chunk")
	}
	if channels != 1 {
		return nil, fmt.Errorf("need mono WAV, got %d channels", channels)
	}
	if rate != 48000 {
		return nil, fmt.Errorf("need 48000 Hz sample rate, got %d", rate)
	}
	if bits != 16 {
		return nil, fmt.Errorf("need 16-bit samples, got %d", bits)
	}
	return pcm, nil
}
