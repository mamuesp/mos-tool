package common

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"cesanta.com/common/go/ourutil"
	"cesanta.com/mos/version"
	"github.com/cesanta/errors"
)

const (
	manifestFileName = "manifest.json"
)

func getFirmwareURL(appName, platformWithVariation string) string {
	return fmt.Sprintf(
		"https://github.com/mongoose-os-apps/%s/releases/download/%s/%s-%s.zip",
		appName, version.GetMosVersion(), appName, platformWithVariation,
	)
}

func getDemoAppName(platformWithVariation string) string {
	appName := "demo-js"
	if strings.HasPrefix(platformWithVariation, "cc3200") {
		appName = "demo-c"
	}
	return appName
}

func NewZipFirmwareBundle(fname string) (*FirmwareBundle, error) {
	var r *zip.Reader

	// If firmware name is given but does not end up with .zip, this is
	// a shortcut for `mos flash esp32`. Transform that into the canonical URL
	_, err := os.Stat(fname)
	if fname != "" && !strings.HasSuffix(fname, ".zip") && os.IsNotExist(err) && !strings.Contains(fname, "/") {
		platforWithVariation := fname
		appName := getDemoAppName(platforWithVariation)
		fname = getFirmwareURL(appName, platforWithVariation)
	}

	zipData, err := ourutil.ReadOrFetchFile(fname)
	if err != nil {
		return nil, errors.Trace(err)
	}

	r, err = zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, errors.Annotatef(err, "%s: invalid firmware file", fname)
	}

	fwb := &FirmwareBundle{Blobs: make(map[string][]byte)}
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return nil, errors.Annotatef(err, "%s: failed to open", fname)
		}
		data, err := ioutil.ReadAll(rc)
		if err != nil {
			return nil, errors.Annotatef(err, "%s: failed to read", fname)
		}
		rc.Close()
		fwb.Blobs[path.Base(f.Name)] = data
	}
	if fwb.Blobs[manifestFileName] == nil {
		return nil, errors.Errorf("%s: no %s in the archive", fname, manifestFileName)
	}
	err = json.Unmarshal(fwb.Blobs[manifestFileName], &fwb.FirmwareManifest)
	if err != nil {
		return nil, errors.Annotatef(err, "%s: failed to parse manifest", fname)
	}
	for n, p := range fwb.FirmwareManifest.Parts {
		p.Name = n
		// Backward compat
		p.CC32XXFileSignature = p.CC32XXFileSignature + p.CC32XXFileSignatureOld
	}
	return fwb, nil
}
