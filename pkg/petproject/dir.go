package petproject

import (
	"path"
	"strings"
)

const MarketRootPrefix = "pet-market-listings"

func MarketListingPrefix(listingID string) string {
	return path.Join(MarketRootPrefix, listingID) + "/"
}

func ManifestJSONFromFiles(files map[string]string) string {
	raw, ok := files[ManifestFile]
	if !ok {
		return "{}"
	}
	return strings.TrimSpace(raw)
}

func EntryScriptFromFiles(files map[string]string, entry string) string {
	entry = strings.TrimSpace(entry)
	if entry == "" {
		entry = DefaultEntry
	}
	if script, ok := files[entry]; ok {
		return script
	}
	return ""
}
