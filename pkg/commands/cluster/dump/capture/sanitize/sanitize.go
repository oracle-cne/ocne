// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package sanitize

import (
	"bufio"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"k8s.io/apimachinery/pkg/util/uuid"
)

const (
	// File containing a map from redacted values to their original values

	RedactionPrefix = "REDACTED-"
	RedactionMap    = "sensitive-do-not-share-redaction-map.csv"
)

type regexPlan struct {
	preprocess  func(string) string
	regex       string
	postprocess func(string) string
}

// Synchronizing access is required due to multiple go-routines simultaneously accessing
var regexToReplacementListMutex sync.RWMutex
var regexToReplacementList = []regexPlan{}

var KnownHostNames = make(map[string]bool)
var KnownHostNamesMutex = &sync.Mutex{}
var KnownNodeNames = make(map[string]string)
var KnownNodeNamesMutex = &sync.Mutex{}

// A map to keep track of all the strings that have been redacted.
var redactedValues = make(map[string]string)
var redactedValuesMutex = &sync.Mutex{}

var ipv4Regex = regexPlan{regex: "[[:digit:]]{1,3}\\.[[:digit:]]{1,3}\\.[[:digit:]]{1,3}\\.[[:digit:]]{1,3}"}
var userData = regexPlan{regex: "\"user_data\":\\s+\"[A-Za-z0-9=+]+\""}
var sshAuthKeys = regexPlan{regex: "ssh-rsa\\s+[A-Za-z0-9=+ \\-\\/@]+"}
var ocid = regexPlan{regex: "ocid1\\.[[:lower:]]+\\.[[:alnum:]]+\\.[[:alnum:]]*\\.[[:alnum:]]+"}
var opcid = regexPlan{
	preprocess: func(s string) string {
		return strings.Trim(strings.TrimPrefix(s, "Opc request id:"), " ")
	},
	regex: "(?:Opc request id:) *[A-Z,a-z,/,0-9]+",
	postprocess: func(s string) string {
		return "Opc request id: " + s
	},
}

// InitRegexToReplacementMap Initialize the regex string to replacement string map
// Append to this map for any future additions
func InitRegexToReplacementMap() {
	regexToReplacementListMutex.Lock()
	if len(regexToReplacementList) == 0 {
		regexToReplacementList = append(regexToReplacementList, ipv4Regex)
		regexToReplacementList = append(regexToReplacementList, userData)
		regexToReplacementList = append(regexToReplacementList, sshAuthKeys)
		regexToReplacementList = append(regexToReplacementList, ocid)
		regexToReplacementList = append(regexToReplacementList, opcid)
	}
	regexToReplacementListMutex.Unlock()
}

// SanitizeString sanitizes each line in a given file,
// Sanitizes based on the regex map initialized above, which is currently filtering for IPv4 addresses and hostnames
//
// The redactedValuesOverride parameter can be used to override the default redactedValues map for keeping track of
// redacted strings.
func SanitizeString(l string, redactedValuesOverride map[string]string) string {
	InitRegexToReplacementMap()

	KnownHostNamesMutex.Lock()
	for knownHost := range KnownHostNames {
		wholeOccurrenceHostPattern := "\"" + knownHost + "\""
		l = regexp.MustCompile(wholeOccurrenceHostPattern).ReplaceAllString(l, "\""+RedactionPrefix+GetShortSha256Hash(knownHost)+"\"")
	}
	KnownHostNamesMutex.Unlock()

	KnownNodeNamesMutex.Lock()
	for knownNode, hash := range KnownNodeNames {
		l = regexp.MustCompile(knownNode).ReplaceAllString(l, hash)
	}
	KnownNodeNamesMutex.Unlock()

	regexToReplacementListMutex.Lock()
	for _, eachRegex := range regexToReplacementList {
		redactedValuesMutex.Lock()
		l = regexp.MustCompile(eachRegex.regex).ReplaceAllStringFunc(l, eachRegex.compilePlan(redactedValuesOverride))
		redactedValuesMutex.Unlock()
	}
	regexToReplacementListMutex.Unlock()

	return l
}

// WriteRedactionMapFile creates a CSV file to document all the values this tool has
// redacted so far, stored in the redactedValues (or redactedValuesOverride) map.
func WriteRedactionMapFile(captureDir string, redactedValuesOverride map[string]string) error {
	fileName := filepath.Join(captureDir, RedactionMap)
	f, err := os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("Error creating file %s: %v", fileName, err.Error())
	}
	defer f.Close()

	redactedValuesMutex.Lock()
	redactedValues := determineRedactedValuesMap(redactedValuesOverride)
	csvWriter := csv.NewWriter(f)
	for s, r := range redactedValues {
		if err = csvWriter.Write([]string{r, s}); err != nil {
			return fmt.Errorf("An error occurred while writing the file %s: %s\n", fileName, err.Error())
			return err
		}
	}
	redactedValuesMutex.Unlock()
	csvWriter.Flush()
	return nil
}

// compilePlan returns a function which processes strings according the regexPlan rp.
func (rp regexPlan) compilePlan(redactedValuesOverride map[string]string) func(string) string {
	return func(s string) string {
		if rp.preprocess != nil {
			s = rp.preprocess(s)
		}
		s = redact(s, redactedValuesOverride)
		if rp.postprocess != nil {
			return rp.postprocess(s)
		}
		return s
	}
}

// redact outputs a string, representing a piece of redacted text.
// If a new string is encountered, keep track of it.
func redact(s string, redactedValuesOverride map[string]string) string {
	redactedValues := determineRedactedValuesMap(redactedValuesOverride)
	if r, ok := redactedValues[s]; ok {
		return r
	}
	r := GetShortSha256Hash(s)
	redactedValues[s] = RedactionPrefix + r
	return r
}

// GetShortSha256Hash generates the one way hash for the input string and then returns "REDACTED-"and the first 7 characters of that hash
func GetShortSha256Hash(line string) string {
	data := []byte(line)
	hashedVal := sha256.Sum256(data)
	hexString := hex.EncodeToString(hashedVal[:])
	returnString := hexString[0:8]
	return returnString
}

// determineRedactedValuesMap returns the map of redacted values to use, according to the override provided
func determineRedactedValuesMap(redactedValuesOverride map[string]string) map[string]string {
	if redactedValuesOverride != nil {
		return redactedValuesOverride
	}
	return redactedValues
}

// SanitizeFilesInDirTree all files in a directory tree, including files in all nested subdirectories
func SanitizeFilesInDirTree(rootDir string) error {
	err := filepath.WalkDir(rootDir,
		// Sanitize each file
		func(path string, dirEntry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if dirEntry.IsDir() {
				return nil // walk into this dir
			}
			if err := sanitizeFile(path); err != nil {
				return err
			}
			return nil
		})
	return err
}

// sanitizeFile sanitizes the file by redacting sensitive information.
func sanitizeFile(srcPath string) error {
	fSrc, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer fSrc.Close()

	destPath := srcPath + "." + string(uuid.NewUUID())
	fDest, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer fDest.Close()

	// read each line, sanitize it and write to the destination file
	if err := SanitizeLines(fSrc, fDest); err != nil {
		return err
	}

	// Move the tmp file to the old file
	fSrc.Close()
	if err := os.Remove(srcPath); err != nil {
		return err
	}
	if err := os.Rename(destPath, srcPath); err != nil {
		return err
	}
	return nil
}

// read each line, sanitize it and write to writer
func SanitizeLines(reader io.Reader, writer io.Writer) error {
	scanner := bufio.NewScanner(reader)
	bufWriter := bufio.NewWriter(writer)
	for scanner.Scan() {
		_, err := bufWriter.WriteString(SanitizeString(scanner.Text()+"\n", nil))
		if err != nil {
			return err
		}
	}
	bufWriter.Flush()
	return nil
}

// PutIntoNodeNamesIfNotPresent populates the node map with a given node name
func PutIntoNodeNamesIfNotPresent(inputKey string) {
	KnownNodeNamesMutex.Lock()
	defer KnownNodeNamesMutex.Unlock()
	_, ok := KnownNodeNames[inputKey]
	if !ok {
		KnownNodeNames[inputKey] = RedactionPrefix + GetShortSha256Hash(inputKey)
	}
}
