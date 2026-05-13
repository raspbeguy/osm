package version

const (
	Product = "osmctl"
	Version = "0.1.0"
)

// CreatedBy returns the value used for the OSM `created_by` changeset tag,
// tagged with the front-end (e.g., "cli", "tui") so server-side stats can
// tell invocation modes apart.
func CreatedBy(frontend string) string {
	return Product + "/" + Version + " (" + frontend + ")"
}

// UserAgent is the HTTP User-Agent sent on every API request.
const UserAgent = Product + "/" + Version + " (+https://github.com/raspbeguy/osm)"
