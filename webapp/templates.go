package webapp

import (
	"html/template"
	"io"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	initTemplates sync.Once
	rootTemplate  *template.Template
)

func addTemplate(parent *template.Template, name, code string) {
	_, err := parent.New(name).Parse(code)
	if err != nil {
		panic(err)
	}
}

func initRootTemplate() {
	t := template.New("root")
	addTemplate(t, "landingPage", landingPageTemplate)
	rootTemplate = t
}

func getTemplates() *template.Template {
	if rootTemplate == nil {
		initTemplates.Do(initRootTemplate)
	}
	return rootTemplate
}

func Render(out io.Writer, name string, data interface{}) error {
	err := getTemplates().ExecuteTemplate(out, name, data)
	if err != nil {
		renderLog.Error().Err(err).Str("page", name).Send()
		renderErrors.With(prometheus.Labels{"page": name}).Add(1)
	}
	return err
}
