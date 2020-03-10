package webapp

import (
	"net/http"

	"github.com/andrebq/covid-watch/analysis"
	"github.com/rs/zerolog/log"
)

const (
	showTagsPageTemplate = `
<!doctype html>
<html>
	<head>
		<title>Covid Watch - Show tags</title>
	</head>
	<body>
		<p>How tags are being used?</p>
		<ul>
		{{ range $key, $value := .Tags }}
			<li><strong>{{ $key }}</strong>: {{ $value }}</li>
		{{ end }}
		</ul>
	</body>
</html>
`
)

func showTagsPage(w http.ResponseWriter, req *http.Request) {
	tags, err := analysis.DiscoverHashtags("*.msgpack")
	if err != nil {
		log.Error().Err(err).Msg("Unable to extract list of hashtags")
		http.Error(w, "Unable to get the list of tags", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	Render(w, "showTagsPage", tags)
}
