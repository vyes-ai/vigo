// router.go
// Copyright (C) 2024 veypi <i@veypi.com>
// 2024-08-07 13:45
// Distributed under terms of the MIT license.
package vigo

import (
	"fmt"
	"net/http"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/vyes/vigo/logv"
)

type FuncX2None = func(*X)
type FuncX2Any = func(*X) any
type FuncX2Err = func(*X) error
type FuncX2AnyErr = func(*X) (any, error)
type FuncAny2None = func(*X, any)
type FuncAny2Any = func(*X, any) any
type FuncAny2Err = func(*X, any) error
type FuncAny2AnyErr = func(*X, any) (any, error)
type FuncDescription = string

type FuncHttp2None = func(http.ResponseWriter, *http.Request)
type FuncHttp2Any = func(http.ResponseWriter, *http.Request) any
type FuncHttp2Err = func(http.ResponseWriter, *http.Request) error
type FuncHttp2AnyErr = func(http.ResponseWriter, *http.Request) (any, error)
type FuncErr = func(*X, error) error
type FuncSkipBefore func()

var SkipBefore FuncSkipBefore

var allowedMethods = []string{
	http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut,
	http.MethodPatch, http.MethodDelete, http.MethodConnect,
	http.MethodOptions, http.MethodTrace, "PROPFIND", "ANY"}

func NewRouter() Router {
	r := &route{
		funcBefore: make([]any, 0, 10),
		funcAfter:  make([]any, 0, 10),
	}
	return r
}

type Router interface {
	String() string
	Print()
	GetParamsList() []string
	ServeHTTP(http.ResponseWriter, *http.Request)
	SubRouter(prefix string) Router

	Clear(url string, method string)
	Set(url string, method string, handlers ...any) Router
	Get(url string, handlers ...any) Router
	Any(url string, handlers ...any) Router
	Post(url string, handlers ...any) Router
	Head(url string, handlers ...any) Router
	Put(url string, handlers ...any) Router
	Patch(url string, handlers ...any) Router
	Delete(url string, handlers ...any) Router

	UseBefore(middleware ...any) Router
	UseAfter(middleware ...any) Router
	Replace(Router) Router
	Extend(string, Router) Router
}

type route struct {
	// just blank for root router
	fragment       string
	funcBefore     []any
	funcAfter      []any
	handlers       map[string][]any
	handlersCache  map[string][]any
	handlersCaller map[string][3]string
	handlersDesc   map[string][2]string

	parent *route

	subRouters map[string]*route
	colon      *route
	wildcard   *route
}

func (r *route) Print() {
	fmt.Printf("Router Table\n%s\n", strings.Join(r.tree(""), "\n"))
}

func (r *route) tree(root string) []string {
	if root == "/" {
		root = ""
	}
	root = root + "/" + r.fragment
	fc := func(res []string, subt *route) []string {
		if subt != nil {
			for _, s := range subt.tree(root) {
				res = append(res, s)
			}
		}
		return res
	}
	res := make([]string, 0, 10)
	if len(r.handlers) > 0 {
		item := root
		if item == "" {
			item = "/"
		}
		item = "\033[32m" + item + "\033[0m"
		for m := range r.handlers {
			item += "\n    " + m
			for _, h := range r.handlersCache[m] {
				if des, ok := h.(string); ok {
					item += fmt.Sprintf(" |des: %s|", des)
					continue
				}
				op := reflect.ValueOf(h).Pointer()
				fnName := strings.Split(runtime.FuncForPC(op).Name(), "/")
				item += fmt.Sprintf(" %s", fnName[len(fnName)-1])
			}
		}
		res = append(res, item)
	}
	for _, subT := range r.subRouters {
		res = fc(res, subT)
	}
	res = fc(res, r.colon)
	res = fc(res, r.wildcard)
	return res
}

func (r *route) GetParamsList() []string {
	var res []string
	tr := r
	for tr != nil {
		if strings.HasPrefix(tr.fragment, ":") || strings.HasPrefix(tr.fragment, "*") {
			res = append(res, tr.fragment)
		}
		tr = tr.parent
	}
	return res
}

func (r *route) String() string {
	if r.parent != nil {
		return r.parent.String() + "/" + r.fragment
	}
	return r.fragment
}

func (r *route) match(u string, m string, x *X) (*route, []any) {
	if u == "/" || u == "" {
		if len(r.handlers[m]) > 0 {
			return r, r.handlersCache[m]
		} else if len(r.handlers["ANY"]) > 0 {
			return r, r.handlersCache["ANY"]
		}
		if r.wildcard != nil {
			if len(r.wildcard.handlers[m]) > 0 {
				x.setParam(r.wildcard.fragment[1:], "")
				return r.wildcard, r.wildcard.handlersCache[m]
			} else if len(r.wildcard.handlers["ANY"]) > 0 {
				x.setParam(r.wildcard.fragment[1:], "")
				return r.wildcard, r.wildcard.handlersCache["ANY"]
			}
		}
		return nil, nil
	}
	idx := 0
	for i, v := range u {
		if v == '/' {
			break
		} else {
			idx = i + 1
		}
	}
	nexts := u[idx:]
	if len(nexts) > 0 && nexts[0] == '/' {
		nexts = nexts[1:]
	}
	if subr := r.subRouters[u[:idx]]; subr != nil {
		temp, fcs := subr.match(nexts, m, x)
		if temp != nil {
			return temp, fcs
		}
	}
	if r.colon != nil {
		temp, fcs := r.colon.match(nexts, m, x)
		if temp != nil {
			x.setParam(r.colon.fragment[1:], u[:idx])
			return temp, fcs
		}
	}
	if r.wildcard != nil {
		if len(r.wildcard.handlers[m]) > 0 {
			x.setParam(r.wildcard.fragment[1:], u)
			return r.wildcard, r.wildcard.handlersCache[m]
		} else if len(r.wildcard.handlers["ANY"]) > 0 {
			x.setParam(r.wildcard.fragment[1:], u)
			return r.wildcard, r.wildcard.handlersCache["ANY"]
		}
	}
	return nil, nil
}

func (r *route) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	x := acquire()
	defer release(x)
	x.Request = req
	x.writer = w
	start := time.Now()
	_ = start

	if subR, fcs := r.match(req.URL.Path[1:], req.Method, x); subR != nil && len(fcs) > 0 {

		x.fcs = fcs
		x.Next()
		logv.WithNoCaller.Debug().Int("ms", int(time.Since(start).Milliseconds())).Str("method", req.Method).Msg(req.RequestURI)
	} else {
		logv.WithNoCaller.Warn().Str("method", req.Method).Str("path", req.URL.Path).Msg("Not Handled")
	}
}

func (r *route) get_subrouter(url string) *route {
	if url == "" || url == "/" {
		return r
	}
	startIdx := 0
	if url[0] == '/' {
		url = url[1:]
	}
	if url[len(url)-1] != '/' {
		url += "/"
	}
	var next *route
	last := r
	for i, c := range url {
		if c == '/' {
			next = &route{
				fragment: url[startIdx:i],
				parent:   last,
			}
			startIdx = i + 1
			if next.fragment == "" {
				logv.Assert(false, "url path can not has //")
			} else if next.fragment[0] == '*' {
				if last.wildcard != nil {
					if last.wildcard.fragment != next.fragment {
						logv.Warn().Msgf("variable path conflict: %s %s", last.colon.String(), next.String())
					}
					return last.wildcard
				}
				last.wildcard = next
				return next
			} else if next.fragment[0] == ':' {
				if last.colon != nil {
					if last.colon.fragment != next.fragment {
						logv.Warn().Msgf("variable path conflict: %s %s", last.colon.String(), next.String())
					}
					last = last.colon
				} else {
					last.colon = next
					last = next
				}
				continue
			}

			if last.subRouters == nil {
				last.subRouters = make(map[string]*route)
			}
			if tmp := last.subRouters[next.fragment]; tmp != nil {
				last = tmp
			} else {
				last.subRouters[next.fragment] = next
				last = next
			}
		}
	}
	return last
}

func (r *route) Clear(prefix string, method string) {
	var tmp *route
	if len(r.fragment) > 0 && r.fragment[0] == '*' {
		tmp = r
	} else {
		tmp = r.get_subrouter(prefix)
	}
	if method == "*" {
		tmp.handlers = nil
		tmp.subRouters = nil
		tmp.funcAfter = nil
		tmp.funcBefore = nil
	} else {
		delete(tmp.handlers, method)
	}
	tmp.syncCache()
}

func (r *route) Set(prefix string, method string, handlers ...any) Router {
	method = strings.ToUpper(method)

	logv.Assert(slices.Contains(allowedMethods, method), fmt.Sprintf("not support HTTP method: %v", method))
	logv.Assert(len(handlers) > 0, "there must be at least one handler")

	var tmp *route
	if len(r.fragment) > 0 && r.fragment[0] == '*' {
		tmp = r
	} else {
		tmp = r.get_subrouter(prefix)
	}
	if tmp.handlers == nil {
		tmp.handlers = make(map[string][]any)
	}
	if tmp.handlersCaller == nil {
		tmp.handlersCaller = make(map[string][3]string)
	}
	if tmp.handlersDesc == nil {
		tmp.handlersDesc = make(map[string][2]string)
	}
	var desc = ""
	var desarg = ""
	filterHandlers := make([]any, 0, len(handlers))
	for _, fc := range handlers {
		switch fc := fc.(type) {
		case FuncX2None, FuncX2Any, FuncX2Err, FuncX2AnyErr,
			FuncAny2None, FuncAny2Any, FuncAny2Err, FuncAny2AnyErr,
			FuncHttp2None, FuncHttp2Any, FuncHttp2Err, FuncHttp2AnyErr,
			FuncErr, FuncSkipBefore:
			filterHandlers = append(filterHandlers, fc)
		case FuncDescription:
			desc = fc
		default:
			fct := reflect.TypeOf(fc)
			if fct.Kind() == reflect.Ptr {
				fct = fct.Elem()
			}
			if fct.Kind() == reflect.Struct {
				for i := 0; i < fct.NumField(); i++ {
					field := fct.Field(i)
					desarg += fmt.Sprintf("%s    %v    '%v'\n", field.Name, field.Type, field.Tag)
				}
			} else {
				logv.WithNoCaller.Fatal().Caller(2).Msgf("handler type not support: %T", fc)
			}
		}
	}
	tmp.handlersDesc[method] = [2]string{desc, desarg}
	if tmp.handlers[method] != nil {
		logv.Warn().Msgf("handler %s %s already exists", tmp.String(), method)
		tmp.handlers[method] = filterHandlers
	} else {
		tmp.handlers[method] = filterHandlers
	}
	depth := 1
	for {
		pc, file, line, ok := runtime.Caller(depth)
		depth++
		if !ok {
			break
		}
		funcName := runtime.FuncForPC(pc).Name()
		if !strings.HasPrefix(funcName, "github.com/vyes/vigo") {
			tmp.handlersCaller[method] = [3]string{file, fmt.Sprintf("%d", line), funcName}
			break
		}
	}
	tmp.syncCache()
	return tmp
}
func (r *route) Any(url string, handlers ...any) Router {
	return r.Set(url, "ANY", handlers...)
}
func (r *route) Get(url string, handlers ...any) Router {
	return r.Set(url, http.MethodGet, handlers...)
}
func (r *route) Post(url string, handlers ...any) Router {
	return r.Set(url, http.MethodPost, handlers...)
}
func (r *route) Head(url string, handlers ...any) Router {
	return r.Set(url, http.MethodHead, handlers...)
}
func (r *route) Put(url string, handlers ...any) Router {
	return r.Set(url, http.MethodPut, handlers...)
}
func (r *route) Patch(url string, handlers ...any) Router {
	return r.Set(url, http.MethodPatch, handlers...)
}
func (r *route) Delete(url string, handlers ...any) Router {
	return r.Set(url, http.MethodDelete, handlers...)
}

func (r *route) UseAfter(middleware ...any) Router {
	for _, m := range middleware {
		switch m := m.(type) {
		case FuncX2None, FuncX2Any, FuncX2Err, FuncX2AnyErr,
			FuncAny2None, FuncAny2Any, FuncAny2Err, FuncAny2AnyErr,
			FuncHttp2None, FuncHttp2Any, FuncHttp2Err, FuncHttp2AnyErr,
			FuncErr, FuncSkipBefore:
			r.use(m, false)
		default:
			panic(fmt.Sprintf("not support middleware %T", m))
		}
	}
	return r
}

func (r *route) UseBefore(middleware ...any) Router {
	for _, m := range middleware {
		switch m := m.(type) {
		case FuncX2None, FuncX2Any, FuncX2Err, FuncX2AnyErr,
			FuncAny2None, FuncAny2Any, FuncAny2Err, FuncAny2AnyErr,
			FuncHttp2None, FuncHttp2Any, FuncHttp2Err, FuncHttp2AnyErr,
			FuncErr, FuncSkipBefore:
			r.use(m, true)
		default:
			panic(fmt.Sprintf("not support middleware %T", m))
		}
	}
	return r
}

func (r *route) use(m any, before bool) {
	if before {
		r.funcBefore = append(r.funcBefore, m)
	} else {
		r.funcAfter = append(r.funcAfter, m)
	}
	r.syncCache()
}

func (r *route) syncCache() {
	r.handlersCache = make(map[string][]any)
	before := make([]any, 0, 10)
	after := make([]any, 0, 10)
	tmpr := r
	for tmpr != nil {
		// ! slice 陷阱
		// before = append(tmpr.funcBefore[:], before...)
		before = append(before[:0], append(tmpr.funcBefore, before...)...)
		after = append(after, tmpr.funcAfter...)
		tmpr = tmpr.parent
	}
	for k := range r.handlers {
		r.handlersCache[k] = append(append([]any{}, before...), r.handlers[k]...)
		r.handlersCache[k] = append(r.handlersCache[k], after...)
		skipIdx := -1
		for i := range r.handlersCache[k] {
			if _, ok := r.handlersCache[k][i].(FuncSkipBefore); ok {
				skipIdx = i
			}
		}
		if skipIdx >= 0 {
			r.handlersCache[k] = append([]any{}, r.handlersCache[k][skipIdx+1:]...)
		}
	}

	for _, sub := range r.subRouters {
		sub.syncCache()
	}
	if r.colon != nil {
		r.colon.syncCache()
	}
	if r.wildcard != nil {
		r.wildcard.syncCache()
	}
}

func (r *route) Extend(prefix string, subr Router) Router {
	return r.get_subrouter(prefix).Replace(subr)
}
func (r *route) Replace(subr Router) Router {
	// r.parent = parent.(*route)
	logv.Assert(r.parent != nil, "root router can not replace")
	name := r.fragment
	sub := subr.(*route)
	sub.fragment = name
	sub.parent = r.parent
	if name[0] == '*' {
		r.parent.wildcard = sub
	} else if name[0] == ':' {
		r.parent.colon = sub
	} else {
		r.parent.subRouters[name] = sub
	}
	sub.syncCache()
	return sub
}

func (r *route) SubRouter(prefix string) Router {
	logv.Assert(prefix != "" && prefix != "/", "subrouter path can not be '' or '/'")
	return r.get_subrouter(prefix)
}

type rschema struct {
	Tag      string              `json:"tag"`
	Handlers []map[string]string `json:"handlers"`
	Sub      []*rschema          `json:"sub"`
	Full     string              `json:"full"`
}

func (r *route) getSchema() *rschema {
	resp := &rschema{
		Tag:  r.fragment,
		Full: r.String(),
	}
	resp.Handlers = make([]map[string]string, 0, len(r.handlersCache))
	for m, fcs := range r.handlersCache {
		fc := make(map[string]string)
		fc["desc"] = r.handlersDesc[m][0]
		fc["args"] = r.handlersDesc[m][1]
		funcs := make([]string, 0, len(fcs))
		for _, h := range fcs {
			op := reflect.ValueOf(h).Pointer()
			fullName := runtime.FuncForPC(op).Name()
			funcs = append(funcs, fullName)
		}
		fc["funcs"] = strings.Join(funcs, ",")
		fc["method"] = m
		if info, ok := r.handlersCaller[m]; ok {
			fc["file"] = info[0]
			fc["line"] = info[1]
			fc["caller"] = info[2]
		}
		resp.Handlers = append(resp.Handlers, fc)
	}
	resp.Sub = make([]*rschema, 0, 10)
	for _, sub := range r.subRouters {
		resp.Sub = append(resp.Sub, sub.getSchema())
	}
	if r.colon != nil {
		resp.Sub = append(resp.Sub, r.colon.getSchema())
	}
	if r.wildcard != nil {
		resp.Sub = append(resp.Sub, r.wildcard.getSchema())
	}
	return resp
}
