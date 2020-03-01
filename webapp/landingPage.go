package webapp

import (
	"net/http"
	"os"
	"strings"
)

const (
	landingPageTemplate = `
<!doctype html>
<html>
	<head>
		<title>Covid Watch - Landing page</title>
	</head>
	<body>
		<p>Currently watching out for the following terms:</p>
		<ul>
		{{ range .SearchTerms }}
			<li>{{ . }}</li>
		{{ end }}
		</ul>
	</body>
</html>
`
)

func landingPage(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
	Render(w, "landingPage", struct{ SearchTerms []string }{SearchTerms: strings.Split(os.Getenv("SEARCH_TERMS"), ";")})
}
