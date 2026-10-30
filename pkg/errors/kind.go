package errors

// Kind classifies an error for the API boundary: which HTTP status it should
// leave as, decided where the error is CREATED — the only place that knows.
//
// Before this, handlers classified by matching phrases in the error text
// (`strings.Contains(err.Error(), "already exists")` and friends). That works
// until someone rewords a message, and then a 409 silently becomes a 500 with
// no test failing — the message is prose, and prose is not a contract.
type Kind int

const (
	// KindUnknown is the zero value: an unclassified error. The boundary treats
	// it as a server fault (500, sanitized, logged).
	KindUnknown Kind = iota

	// KindInvalid: the request is well-formed but wrong — a value out of range,
	// a state that forbids the operation. 400.
	KindInvalid

	// KindNotFound: the thing does not exist (or must appear not to). 404.
	KindNotFound

	// KindConflict: the thing already exists, or a uniqueness rule bit. 409.
	KindConflict

	// KindForbidden: the caller is known and refused. 403.
	KindForbidden

	// KindUnauthorized: the caller is not authenticated. 401.
	KindUnauthorized

	// KindPlanRequired: the org's plan does not allow this; upgrading would. 402.
	KindPlanRequired
)

// planLimitError is a pointer so errors.Is compares identity, never struct contents.
type planLimitError struct{ msg string }

func (p *planLimitError) Error() string { return p.msg }

// PlanLimit declares a plan-cap sentinel. Its message reaches the user as-is,
// whatever the error is wrapped in.
func PlanLimit(msg string) error {
	return &planLimitError{msg: msg}
}

// PlanLimitMessage returns the user-facing message of a plan-cap error in err's chain.
func PlanLimitMessage(err error) (string, bool) {
	var p *planLimitError
	if !As(err, &p) {
		return "", false
	}

	return p.msg, true
}

// kinded wraps an error with its boundary classification. Separate from
// structured so kinding composes with wrapping in either order.
type kinded struct {
	err  error
	kind Kind
}

func (k kinded) Error() string { return k.err.Error() }
func (k kinded) Unwrap() error { return k.err }

// WithKind classifies an existing error.
func WithKind(err error, kind Kind) error {
	if err == nil {
		return nil
	}

	return kinded{err: err, kind: kind}
}

// Invalid creates a KindInvalid error with structured attrs.
func Invalid(msg string, attrs ...any) error {
	return WithKind(New(msg, attrs...), KindInvalid)
}

// NotFound creates a KindNotFound error with structured attrs.
func NotFound(msg string, attrs ...any) error {
	return WithKind(New(msg, attrs...), KindNotFound)
}

// Conflict creates a KindConflict error with structured attrs.
func Conflict(msg string, attrs ...any) error {
	return WithKind(New(msg, attrs...), KindConflict)
}

// Forbidden creates a KindForbidden error with structured attrs.
func Forbidden(msg string, attrs ...any) error {
	return WithKind(New(msg, attrs...), KindForbidden)
}

// Unauthorized creates a KindUnauthorized error with structured attrs.
func Unauthorized(msg string, attrs ...any) error {
	return WithKind(New(msg, attrs...), KindUnauthorized)
}

// KindOf reports an error's classification, walking the wrap chain so a kinded
// error keeps its kind through every errors.Wrap applied above it.
func KindOf(err error) Kind {
	var k kinded
	if As(err, &k) {
		return k.kind
	}

	_, isPlanLimit := PlanLimitMessage(err)
	if isPlanLimit {
		return KindPlanRequired
	}

	return KindUnknown
}

// AttrsOf returns the merged structured attrs carried by an error's chain, or
// nil when it carries none.
//
// This exists because Error() deliberately returns only the message — attrs
// were previously visible ONLY via the constructor's log line, which is gone.
// The boundary that logs a fault calls this so the one surviving log line
// carries everything the error knows.
func AttrsOf(err error) []any {
	var s structured
	if !As(err, &s) {
		return nil
	}

	// Wrap merges inner attrs outward, so the outermost structured error
	// already carries the whole chain's attrs.
	return s.Attrs()
}
