package stores

import "strings"

// ResolveUploadPublicURL returns a URL clients can use to fetch an object stored under key,
// typically served by the app at /uploads/<key>. Resolution order matches PublicObjectURL (absolute
// URLs from the store or STORAGE_PUBLIC_BASE_URL), then optional explicit publicBase (e.g. config),
// then request proto+host, then a relative path.
func ResolveUploadPublicURL(s Store, bucketName, key, publicBase, forwardedProto, host string) string {
	key = strings.TrimPrefix(strings.TrimSpace(key), "/")
	u := strings.TrimSpace(PublicObjectURL(s, bucketName, key))
	lower := strings.ToLower(u)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return u
	}
	base := strings.TrimSuffix(strings.TrimSpace(publicBase), "/")
	if base != "" {
		return base + "/" + key
	}
	proto := strings.TrimSpace(forwardedProto)
	if proto == "" {
		proto = "http"
	}
	h := strings.TrimSpace(host)
	if h != "" {
		return proto + "://" + h + "/uploads/" + key
	}
	return "/uploads/" + key
}
