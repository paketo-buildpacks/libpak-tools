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
}
