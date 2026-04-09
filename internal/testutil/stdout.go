package testutil

import (
	"io"
	"os"
	"testing"
)

func CaptureStdout(t *testing.T, action func()) string {
	t.Helper()

	originalStdout := os.Stdout
	defer func() {
		os.Stdout = originalStdout
	}()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed creating stdout pipe: %s", err)
	}

	os.Stdout = writer

	action()

	if err := writer.Close(); err != nil {
		t.Fatalf("failed closing writer: %s", err)
	}

	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("failed reading output: %s", err)
	}

	if err := reader.Close(); err != nil {
		t.Fatalf("failed closing reader: %s", err)
	}

	return string(output)
}
