package main

import (
	"os"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/paketo-buildpacks/libpak-tools/internal/testutil"
)

func TestUnit(t *testing.T) {
	suite := spec.New("libpak-tools/main", spec.Report(report.Terminal{}))
	suite("Entrypoint", testEntrypoint)
	suite.Run(t)
}

func testEntrypoint(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect
	)

	context("main", func() {
		it("dispatches to the command tree", func() {
			originalArgs := os.Args
			defer func() { os.Args = originalArgs }()
			os.Args = []string{"libpak-tools", "version"}

			output := testutil.CaptureStdout(t, func() {
				main()
			})

			Expect(output).To(ContainSubstring("libpak-tools"))
		})
	})
}
