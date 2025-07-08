/*
 * Copyright 2018-2020 the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package carton

import (
	"bytes"
	"fmt"
	"os"
	"regexp"

	"github.com/BurntSushi/toml"

	"github.com/paketo-buildpacks/libpak/v2/log"
	"github.com/paketo-buildpacks/libpak/v2/utils"

	"github.com/paketo-buildpacks/libpak-tools/internal"
)

const (
	BuildModuleDependencyPattern      = `(?m)([\s]*.*id[\s]+=[\s]+"%s"\n.*\n[\s]*version[\s]+=[\s]+")%s("\n[\s]*uri[\s]+=[\s]+").*("\n[\s]*sha256[\s]+=[\s]+").*(".*)`
	BuildModuleDependencySubstitution = "${1}%s${2}%s${3}%s${4}"
	defaultArch                       = "amd64"
)

var (
	purlArchExp = regexp.MustCompile(`arch=(.*)`)
)

type BuildModuleDependency struct {
	BuildModulePath string
	ID              string
	Arch            string
	SHA256          string
	URI             string
	Version         string
	VersionPattern  string
	CPE             string
	CPEPattern      string
	PURL            string
	PURLPattern     string
	Source          string
	SourceSHA256    string
	EolID           string
}

func (b BuildModuleDependency) Update(options ...Option) {
	config := Config{
		exitHandler: utils.NewExitHandler(),
	}

	for _, option := range options {
		config = option(config)
	}

	logger := log.NewPaketoLogger(os.Stdout)
	_, _ = fmt.Fprintf(logger.TitleWriter(), "\n%s\n", log.FormatIdentity(b.ID, b.VersionPattern))
	logger.Headerf("Arch:         %s", b.Arch)
	logger.Headerf("Version:      %s", b.Version)
	logger.Headerf("PURL:         %s", b.PURL)
	logger.Headerf("CPEs:         %s", b.CPE)
	logger.Headerf("URI:          %s", b.URI)
	logger.Headerf("SHA256:       %s", b.SHA256)
	logger.Headerf("Source:       %s", b.Source)
	logger.Headerf("SourceSHA256: %s", b.SourceSHA256)
	logger.Headerf("EOL ID:       %s", b.EolID)

	versionExp, err := regexp.Compile(b.VersionPattern)
	if err != nil {
		config.exitHandler.Error(fmt.Errorf("unable to compile version regex %s\n%w", b.VersionPattern, err))
		return
	}

	cpeExp, err := regexp.Compile(b.CPEPattern)
	if err != nil {
		config.exitHandler.Error(fmt.Errorf("unable to compile cpe regex %s\n%w", b.CPEPattern, err))
		return
	}

	purlExp, err := regexp.Compile(b.PURLPattern)
	if err != nil {
		config.exitHandler.Error(fmt.Errorf("unable to compile cpe regex %s\n%w", b.PURLPattern, err))
		return
	}

	c, err := os.ReadFile(b.BuildModulePath)
	if err != nil {
		config.exitHandler.Error(fmt.Errorf("unable to read %s\n%w", b.BuildModulePath, err))
		return
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
		config.exitHandler.Error(fmt.Errorf("unable to decode md%s\n%w", b.BuildModulePath, err))
		return
	}

	metadataUnwrapped, found := md["metadata"]
	if !found {
		config.exitHandler.Error(fmt.Errorf("unable to find metadata block"))
		return
	}

	metadata, ok := metadataUnwrapped.(map[string]interface{})
	if !ok {
		config.exitHandler.Error(fmt.Errorf("unable to cast metadata"))
		return
	}

	dependenciesUnwrapped, found := metadata["dependencies"]
	if !found {
		config.exitHandler.Error(fmt.Errorf("unable to find dependencies block"))
		return
	}

	dependencies, ok := dependenciesUnwrapped.([]map[string]interface{})
	if !ok {
		config.exitHandler.Error(fmt.Errorf("unable to cast dependencies"))
		return
	}

	for _, dep := range dependencies {
		depIDUnwrapped, found := dep["id"]
		if !found {
			continue
		}
		depID, ok := depIDUnwrapped.(string)
		if !ok {
			continue
		}

		// extract the arch from the PURL, it's the only place it lives consistently at the moment
		depArch := dependencyArch(dep)
		if depID == b.ID && depArch == b.Arch {
			depVersionUnwrapped, found := dep["version"]
			if !found {
				continue
			}

			depVersion, ok := depVersionUnwrapped.(string)
			if !ok {
				continue
			}

			if versionExp.MatchString(depVersion) {
				dep["version"] = b.Version
				dep["uri"] = b.URI
				newFormat := updateChecksum(dep, b.SHA256)
				updateSourceChecksum(dep, b.SourceSHA256, newFormat)

				if b.Source != "" {
					dep["source"] = b.Source
				}

				updatePURL(dep, purlExp, b.PURL)
				cpesUnwrapped, found := dep["cpes"]
				if found {
					cpes, ok := cpesUnwrapped.([]interface{})
					if ok {
						for i := 0; i < len(cpes); i++ {
							cpe, ok := cpes[i].(string)
							if !ok {
								continue
							}

							cpes[i] = cpeExp.ReplaceAllString(cpe, b.CPE)
						}
					}
				}

				if b.EolID != "" {
					eolDate, err := internal.GetEolDate(b.EolID, b.Version)
					if err != nil {
						config.exitHandler.Error(fmt.Errorf("unable to fetch deprecation_date"))
						return
					}

					if eolDate != "" {
						eolKey := "deprecation_date"
						if newFormat {
							eolKey = "eol-date"
						}

						dep[eolKey] = eolDate
					}
				}
			}
		}
	}

	c, err = utils.Marshal(md)
	if err != nil {
		config.exitHandler.Error(fmt.Errorf("unable to encode md %s\n%w", b.BuildModulePath, err))
		return
	}

	c = append(comments, c...)

	// #nosec G306 - permissions need to be 644 on the build module
	if err := os.WriteFile(b.BuildModulePath, c, 0644); err != nil {
		config.exitHandler.Error(fmt.Errorf("unable to write %s\n%w", b.BuildModulePath, err))
		return
	}
}

func dependencyArch(dep map[string]interface{}) string {
	if arch, found := dep["arch"]; found {
		if archStr, ok := arch.(string); ok {
			return archStr
		}
	}

	switch p := dep["purl"].(type) {
	case string:
		return archFromPURL(p)
	default:
		switch p := dep["purls"].(type) {
		case []interface{}:
			for _, u := range p {
				if uStr, ok := u.(string); ok {
					if arch := archFromPURL(uStr); arch != "" {
						return arch
					}
				}
			}
		}

		return defaultArch
	}

}

func archFromPURL(purl string) string {
	purlArchMatches := purlArchExp.FindStringSubmatch(purl)
	if len(purlArchMatches) == 2 {
		return purlArchMatches[1]
	}

	return defaultArch
}

func updateChecksum(dep map[string]interface{}, sha256 string) bool {
	if _, ok := dep["sha256"]; ok {
		dep["sha256"] = sha256
		return false
	}

	if _, ok := dep["checksum"]; ok {
		dep["checksum"] = fmt.Sprintf("sha256:%s", sha256)
		return true
	}

	return false
}

func updateSourceChecksum(dep map[string]interface{}, sourceSHA256 string, newFormat bool) {
	if sourceSHA256 == "" {
		return
	}

	checksumKey := "source-sha256"
	if newFormat {
		checksumKey = "source-checksum"
		sourceSHA256 = fmt.Sprintf("sha256:%s", sourceSHA256)
	}

	dep[checksumKey] = sourceSHA256
}

func updatePURL(dep map[string]interface{}, purlExp *regexp.Regexp, newPURL string) {
	switch purl := dep["purl"].(type) {
	case string:
		dep["purl"] = purlExp.ReplaceAllString(purl, newPURL)
	default:
		switch purls := dep["purls"].(type) {
		case []interface{}:
			for i, p := range purls {
				if pStr, ok := p.(string); ok {
					purls[i] = purlExp.ReplaceAllString(pStr, newPURL)
				}
			}
		}
	}
}
