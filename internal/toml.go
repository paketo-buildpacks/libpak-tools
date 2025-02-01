package internal

import (
	"bytes"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/paketo-buildpacks/libpak/v2/utils"
)

// MultiUpdateTOMLFILE will apply multiple updaters to a TOML file
func MultiUpdateTOMLFILE(cfgPath string, fns ...func(md map[string]interface{})) error {
	for _, f := range fns {
		if err := UpdateTOMLFile(cfgPath, f); err != nil {
			return err
		}
	}
	return nil
}

// UpdateTOMLFile will update a TOML file by calling the provided function with the decoded map
//
// It will preserve leading comments, but not inline comments. It will not preserve the order or formatting of the file.
func UpdateTOMLFile(cfgPath string, f func(md map[string]interface{})) error {
	// preserve the file permissions
	fstat, err := os.Stat(cfgPath)
	if err != nil {
		return fmt.Errorf("unable to stat %s\n%w", cfgPath, err)
	}

	c, err := os.ReadFile(cfgPath)
	if err != nil {
		return fmt.Errorf("unable to read %s\n%w", cfgPath, err)
	}

	// save any leading comments, this is to preserve license headers
	// inline comments will be lost
	comments := []byte{}
	for i, line := range bytes.SplitAfter(c, []byte("\n")) {
		if bytes.HasPrefix(line, []byte("#")) || (i > 0 && len(bytes.TrimSpace(line)) == 0) {
			comments = append(comments, line...)
		} else {
			break // stop on first comment
		}
	}

	md := make(map[string]interface{})
	if err := toml.Unmarshal(c, &md); err != nil {
		return fmt.Errorf("unable to decode md %s\n%w", cfgPath, err)
	}

	f(md)

	b, err := utils.Marshal(md)
	if err != nil {
		return fmt.Errorf("unable to encode md %s\n%w", cfgPath, err)
	}

	b = append(comments, b...)

	if err := os.WriteFile(cfgPath, b, fstat.Mode()); err != nil {
		return fmt.Errorf("unable to write %s\n%w", cfgPath, err)
	}

	return nil
}
