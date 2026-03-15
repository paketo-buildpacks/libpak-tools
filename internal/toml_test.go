package internal_test

import (
	"fmt"
	"os"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/libpak-tools/internal"
)

func testTOML(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect
	)

	context("empty toml file", func() {
		it.Before(func() {
			Expect(os.WriteFile("test.toml", []byte(""), 0600)).To(Succeed())
		})

		it.After(func() {
			Expect(os.RemoveAll("test.toml")).To(Succeed())
		})

		it("updates the file", func() {
			Expect(internal.UpdateTOMLFile("test.toml", func(md map[string]interface{}) {
				md["key"] = "value"
			})).To(Succeed())

			b, err := os.ReadFile("test.toml")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(b)).To(Equal("key = \"value\"\n"))
		})
	})

	context("toml file with comments", func() {
		it.Before(func() {
			Expect(os.WriteFile("test.toml", []byte(`# leading comment

key = "value"
`), 0600)).To(Succeed())
		})

		it.After(func() {
			Expect(os.RemoveAll("test.toml")).To(Succeed())
		})

		it("updates and preserves comments", func() {
			Expect(internal.UpdateTOMLFile("test.toml", func(md map[string]interface{}) {
				md["key"] = "new-value"
				md["foo"] = "bar"
			})).To(Succeed())

			b, err := os.ReadFile("test.toml")
			Expect(err).NotTo(HaveOccurred())
			fmt.Println(string(b))
			Expect(string(b)).To(Equal(`# leading comment

foo = "bar"
key = "new-value"
`))
		})

		it("deletes a key", func() {
			Expect(internal.UpdateTOMLFile("test.toml", func(md map[string]interface{}) {
				delete(md, "key")
			})).To(Succeed())

			b, err := os.ReadFile("test.toml")
			Expect(err).NotTo(HaveOccurred())
			Expect(string(b)).To(Equal(`# leading comment

`))
		})
	})

	context("multiple toml updates", func() {
		var cfgPath string

		it.Before(func() {
			cfgPath = "multi-update.toml"
			Expect(os.WriteFile(cfgPath, []byte("key = \"value\"\n"), 0600)).To(Succeed())
		})

		it.After(func() {
			Expect(os.RemoveAll(cfgPath)).To(Succeed())
		})

		it("applies updates in order", func() {
			Expect(internal.MultiUpdateTOMLFILE(cfgPath,
				func(md map[string]interface{}) {
					md["key"] = "new-value"
				},
				func(md map[string]interface{}) {
					md["added"] = "value"
				},
			)).To(Succeed())

			b, err := os.ReadFile(cfgPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(b)).To(ContainSubstring("key = \"new-value\""))
			Expect(string(b)).To(ContainSubstring("added = \"value\""))
		})

		it("stops applying updates after a failure", func() {
			secondUpdateCalled := false

			err := internal.MultiUpdateTOMLFILE(cfgPath,
				func(md map[string]interface{}) {
					md["invalid"] = make(chan int)
				},
				func(md map[string]interface{}) {
					secondUpdateCalled = true
					md["never-written"] = "true"
				},
			)

			Expect(err).To(HaveOccurred())
			Expect(secondUpdateCalled).To(BeFalse())
		})
	})
}
