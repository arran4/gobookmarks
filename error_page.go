package gobookmarks

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/sessions"
)

const (
	sessionReturnURLKey    = "return:url"
	sessionReturnMethodKey = "return:method"
	sessionReturnBodyKey   = "return:body"
)

// RequestReplay captures enough information about a request so it can be
// repeated after the user signs in again.
type RequestReplay struct {
	Method      string
	URL         string
	Form        url.Values
	EncodedForm string
}

// HasForm reports whether the replay contains form data.
func (r *RequestReplay) HasForm() bool {
	return r != nil && len(r.Form) > 0
}

// CaptureRequestReplay builds a RequestReplay for the supplied request. Only
// form data encoded as application/x-www-form-urlencoded is captured.
func CaptureRequestReplay(req *http.Request) *RequestReplay {
	if req == nil {
		return nil
	}

	rr := &RequestReplay{
		Method: strings.ToUpper(req.Method),
	}
	if req.URL != nil {
		rr.URL = req.URL.RequestURI()
	}
	if rr.URL == "" {
		rr.URL = "/"
	}

	switch req.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		if err := req.ParseForm(); err == nil && len(req.PostForm) > 0 {
			form := url.Values{}
			for k, vals := range req.PostForm {
				copyVals := append([]string(nil), vals...)
				form[k] = copyVals
			}
			rr.Form = form
			rr.EncodedForm = form.Encode()
		}
	}

	rr.URL = sanitizeReturnURL(rr.URL)
	return rr
}

// ErrorPageData is passed to the shared error template. It implements the
// error interface so template helpers can inspect the underlying cause.
type ErrorPageData struct {
	*CoreData
	error
	Message       string
	RequestReplay *RequestReplay
}

// Error implements the error interface.
func (d ErrorPageData) Error() string {
	if d.error != nil {
		return d.error.Error()
	}
	return d.Message
}

// Unwrap exposes the underlying error for errors.Is/As.
func (d ErrorPageData) Unwrap() error { return d.error }

// NewErrorPageData constructs ErrorPageData for the current request.
func NewErrorPageData(r *http.Request, err error, display string) ErrorPageData {
	core, _ := r.Context().Value(ContextValues("coreData")).(*CoreData)
	data := ErrorPageData{
		CoreData:      core,
		error:         err,
		Message:       display,
		RequestReplay: CaptureRequestReplay(r),
	}
	if data.error == nil && display != "" {
		data.error = errors.New(display)
	}
	return data
}

// StoreRequestReplay persists the replay information in the session so it can
// be replayed after the user logs in again.
func StoreRequestReplay(session *sessions.Session, replay *RequestReplay) {
	if session == nil {
		return
	}
	if replay == nil {
		delete(session.Values, sessionReturnURLKey)
		delete(session.Values, sessionReturnMethodKey)
		delete(session.Values, sessionReturnBodyKey)
		return
	}

	dest := sanitizeReturnURL(replay.URL)
	if dest == "" {
		delete(session.Values, sessionReturnURLKey)
		delete(session.Values, sessionReturnMethodKey)
		delete(session.Values, sessionReturnBodyKey)
		return
	}

	method := strings.ToUpper(replay.Method)
	if method == "" {
		method = http.MethodGet
	}

	session.Values[sessionReturnURLKey] = dest
	session.Values[sessionReturnMethodKey] = method
	if replay.EncodedForm != "" {
		session.Values[sessionReturnBodyKey] = replay.EncodedForm
	} else {
		delete(session.Values, sessionReturnBodyKey)
	}
}

// ConsumeRequestReplay retrieves and clears the replay information from the
// session. It returns nil when no replay data is stored.
func ConsumeRequestReplay(session *sessions.Session) *RequestReplay {
	if session == nil {
		return nil
	}
	rawURL, _ := session.Values[sessionReturnURLKey].(string)
	if rawURL == "" {
		return nil
	}

	method, _ := session.Values[sessionReturnMethodKey].(string)
	if method == "" {
		method = http.MethodGet
	}
	body, _ := session.Values[sessionReturnBodyKey].(string)

	delete(session.Values, sessionReturnURLKey)
	delete(session.Values, sessionReturnMethodKey)
	delete(session.Values, sessionReturnBodyKey)

	replay := &RequestReplay{
		Method:      strings.ToUpper(method),
		URL:         rawURL,
		EncodedForm: body,
	}
	if body != "" {
		if form, err := url.ParseQuery(body); err == nil {
			replay.Form = form
		}
	}
	return replay
}

func sanitizeReturnURL(raw string) string {
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	if u.Scheme != "" || u.Host != "" {
		return ""
	}
	if u.Path == "" {
		u.Path = "/"
	}
	u.Fragment = ""
	return u.String()
}
