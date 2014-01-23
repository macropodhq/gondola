package app

import (
	"bytes"
	"encoding/json"
	"gnd.la/app/debug"
	"html/template"
	"strconv"
	"time"
)

const (
	debugCacheKey = "cache"
	debugOrmKey   = "orm"
)

type debugInfo struct {
	Elapsed   time.Duration
	Timings   []*debug.Timing
	Remaining time.Duration
}

// DebugComment returns an HTML comment with some debug information,
// including the time when the template was rendered, the time it
// took to serve the request and the number of queries to the cache
// and the ORM. It is intended to be used in the templates like e.g.
//
//    </html>
//    {{ $Ctx.DebugComment }}
//
// Keep in mind that the comment will only include the number of ORM and
// cache queries when debug is enabled in the App, because otherwise the
// cache and ORM are shared among all contexts and counting the queries
// per request is not possible.
func (c *Context) DebugComment() template.HTML {
	var buf bytes.Buffer
	buf.WriteString("<!-- generated on ")
	buf.WriteString(c.started.String())
	buf.WriteString(" - took ")
	buf.WriteString(c.Elapsed().String())
	if c.debugStorage != nil {
		o, _ := c.debugStorage[debugOrmKey].(*Orm)
		if o != nil {
			buf.WriteString(" - ")
			buf.WriteString(strconv.Itoa(o.NumQueries()))
			buf.WriteString(" ORM queries")
		}
		ca, _ := c.debugStorage[debugCacheKey].(*Cache)
		if ca != nil {
			buf.WriteString(" - ")
			buf.WriteString(strconv.Itoa(ca.NumQueries()))
			buf.WriteString(" cache queries")
		}
	}
	buf.WriteString(" -->")
	return template.HTML(buf.String())
}

func (c *Context) storeDebug(name string, val interface{}) {
	if c.debugStorage == nil {
		c.debugStorage = make(map[string]interface{})
	}
	c.debugStorage[name] = val
}

func (c *Context) getDebug(name string) interface{} {
	return c.debugStorage[name]
}

func debugInfoHandler(ctx *Context) {
	var info *debugInfo
	data := ctx.RequireFormValue("data")
	if err := json.Unmarshal([]byte(data), &info); err != nil {
		panic(err)
	}
	info.Remaining = info.Elapsed
	for _, v := range info.Timings {
		info.Remaining -= v.Total()
	}
	t := newInternalTemplate(ctx.app)
	if err := t.Parse("debug_info.html"); err != nil {
		panic(err)
	}
	if err := t.tmpl.Compile(); err != nil {
		panic(err)
	}
	t.tmpl.MustExecute(ctx, info)
}
