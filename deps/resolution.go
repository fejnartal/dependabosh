package deps

import (
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"github.com/Masterminds/semver/v3"

	"github.com/fejnartal/dependabosh/logs"
)

type Dependabosh struct {
	Updates []Dependency `yaml:"updates"`
}

type Dependency struct {
	NamePattern string `yaml:"name_pattern"`
	Constraints string `yaml:"constraints"`
	CurVersion  string `yaml:"cur_version"`
	VersRegexp  string `yaml:"vers_regexp"`
	VersURL     string `yaml:"vers_url"`
	BlobURL     string `yaml:"blob_url"`
	SrcURL      string `yaml:"src_url"`
}

func findAllVersionsWithCaptureGroups(textContainingVersions string, regex *regexp.Regexp) []string {
	var versionCaptureGroup int

	versionCaptureGroup = 0
	for i, name := range regex.SubexpNames() {
		logs.DebugLogger.Println("Capture Group " + name + " has index " + strconv.Itoa(i))
		if name == "version" {
			versionCaptureGroup = i
		}
	}


	allSubmatches := regex.FindAllStringSubmatch(textContainingVersions, -1)
	allVersions := make([]string, len(allSubmatches))
	for _, submatch := range allSubmatches {
		if submatch[versionCaptureGroup] != "" {
			allVersions = append(allVersions, submatch[versionCaptureGroup])
			logs.DebugLogger.Println("Found version in input data: " + submatch[versionCaptureGroup] + ".")
		}
	}
	return allVersions
}

func CheckLatestVersion(dep Dependency) (string, error) {
	vregexp := regexp.MustCompile(dep.VersRegexp)
	latestVersionsData := fetchLatestVersionsData(dep.VersURL)
	detectedVersions := findAllVersionsWithCaptureGroups(latestVersionsData, vregexp)

	_, err := semver.NewVersion(dep.CurVersion)
	if err != nil {
		logs.ErrorLogger.Println("Vers " + dep.CurVersion + " can't be interpreted as a semver.")
		return "", semver.ErrInvalidSemVer
	}

	candidateVersion := dep.CurVersion
	for _, version := range detectedVersions {
		if version == "" {
			// FIXME
			continue
		}
		logs.DebugLogger.Println("Comparing current candidate " + candidateVersion + " with " + version + ".")
		canonicalCandidate, err := semver.NewVersion(candidateVersion)
		if err != nil {
			logs.ErrorLogger.Println("Vers " + candidateVersion + " can't be interpreted as a semver.")
			continue
		}
		canonicalVersion, err := semver.NewVersion(version)
		if err != nil {
			logs.ErrorLogger.Println("Vers " + version + " can't be interpreted as a semver.")
			return "", semver.ErrInvalidSemVer
		}

		greaterThanCandidateConstraint, _ := semver.NewConstraint("> " + canonicalCandidate.String())
		versionInRangeConstraint, err := semver.NewConstraint(dep.Constraints)
		if err != nil {
			logs.ErrorLogger.Println("Impossible to parse constraints: " + dep.Constraints)
		}

		if !greaterThanCandidateConstraint.Check(canonicalVersion) {
			logs.DebugLogger.Println("Rejecting version " + version + " as it's older than current candidate " + candidateVersion)
		} else {
			if !versionInRangeConstraint.Check(canonicalVersion) {
				logs.DebugLogger.Println("Rejecting version " + version + " despite being newer than current candidate " + candidateVersion + " because it doesn't pass constraints")
			} else {
				logs.DebugLogger.Println("Adopting version " + version + " as it's newer than current candidate " + candidateVersion + " and passes constraints")
				candidateVersion = version
			}
		}
	}
	return candidateVersion, nil
}

func fetchLatestVersionsData(url string) string {
	resp, err := http.Get(url)
	if err != nil {
	   panic(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	return string(body)
}

