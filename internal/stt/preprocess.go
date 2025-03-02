package stt

import (
	"bytes"
	"fmt"
	"os/exec"
	"path"
	"strings"
)

func whisperPreprocess(fname string) (string, error) {
	resfile := strings.TrimSuffix(fname, path.Ext(fname)) + ".wav"
	cmd := exec.Command("ffmpeg", "-i", fname, "-ac", "1", "-ar", "16000", "-c:a", "pcm_s16le", resfile)
	stderr := attachStderr(cmd)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ffmpeg error: %s, %w", stderr.String(), err)
	}

	return resfile, nil
}

func attachStderr(cmd *exec.Cmd) *bytes.Buffer {
	buf := bytes.NewBuffer(nil)
	cmd.Stderr = buf

	return buf
}
