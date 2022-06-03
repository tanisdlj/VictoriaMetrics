// Code generated by qtc from "labels_response.qtpl". DO NOT EDIT.
// See https://github.com/valyala/quicktemplate for details.

//line app/vmselect/prometheus/labels_response.qtpl:3
package prometheus

//line app/vmselect/prometheus/labels_response.qtpl:3
import (
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/querytracer"
)

// LabelsResponse generates response for /api/v1/labels .See https://prometheus.io/docs/prometheus/latest/querying/api/#getting-label-names

//line app/vmselect/prometheus/labels_response.qtpl:9
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line app/vmselect/prometheus/labels_response.qtpl:9
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line app/vmselect/prometheus/labels_response.qtpl:9
func StreamLabelsResponse(qw422016 *qt422016.Writer, labels []string, qt *querytracer.Tracer, qtDone func()) {
//line app/vmselect/prometheus/labels_response.qtpl:9
	qw422016.N().S(`{"status":"success","data":[`)
//line app/vmselect/prometheus/labels_response.qtpl:13
	for i, label := range labels {
//line app/vmselect/prometheus/labels_response.qtpl:14
		qw422016.N().Q(label)
//line app/vmselect/prometheus/labels_response.qtpl:15
		if i+1 < len(labels) {
//line app/vmselect/prometheus/labels_response.qtpl:15
			qw422016.N().S(`,`)
//line app/vmselect/prometheus/labels_response.qtpl:15
		}
//line app/vmselect/prometheus/labels_response.qtpl:16
	}
//line app/vmselect/prometheus/labels_response.qtpl:16
	qw422016.N().S(`]`)
//line app/vmselect/prometheus/labels_response.qtpl:19
	qt.Printf("generate response for %d labels", len(labels))
	qtDone()

//line app/vmselect/prometheus/labels_response.qtpl:22
	streamdumpQueryTrace(qw422016, qt)
//line app/vmselect/prometheus/labels_response.qtpl:22
	qw422016.N().S(`}`)
//line app/vmselect/prometheus/labels_response.qtpl:24
}

//line app/vmselect/prometheus/labels_response.qtpl:24
func WriteLabelsResponse(qq422016 qtio422016.Writer, labels []string, qt *querytracer.Tracer, qtDone func()) {
//line app/vmselect/prometheus/labels_response.qtpl:24
	qw422016 := qt422016.AcquireWriter(qq422016)
//line app/vmselect/prometheus/labels_response.qtpl:24
	StreamLabelsResponse(qw422016, labels, qt, qtDone)
//line app/vmselect/prometheus/labels_response.qtpl:24
	qt422016.ReleaseWriter(qw422016)
//line app/vmselect/prometheus/labels_response.qtpl:24
}

//line app/vmselect/prometheus/labels_response.qtpl:24
func LabelsResponse(labels []string, qt *querytracer.Tracer, qtDone func()) string {
//line app/vmselect/prometheus/labels_response.qtpl:24
	qb422016 := qt422016.AcquireByteBuffer()
//line app/vmselect/prometheus/labels_response.qtpl:24
	WriteLabelsResponse(qb422016, labels, qt, qtDone)
//line app/vmselect/prometheus/labels_response.qtpl:24
	qs422016 := string(qb422016.B)
//line app/vmselect/prometheus/labels_response.qtpl:24
	qt422016.ReleaseByteBuffer(qb422016)
//line app/vmselect/prometheus/labels_response.qtpl:24
	return qs422016
//line app/vmselect/prometheus/labels_response.qtpl:24
}
