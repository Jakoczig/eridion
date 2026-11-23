package providers

import (
	"encoding/json"
	"strconv"
)

// eventOutcome is the durable outcome of one submitted event.
type eventOutcome struct {
	Status string
	Reason string
}

// edsmEventOutcomes maps an EDSM api-journal-v1 reply to one outcome per
// submitted event (Task 7 / F08).
//
// Contract evidence: EDSM's journal API documentation as cited by the EDMC
// EDSM plugin (plugins/edsm.py): top-level msgnum 1xx = OK, 2xx = fatal error,
// 5xx = error but events saved for later processing (3xx/4xx not generated at
// top level); per-event msgnum 1xx = OK, 2xx = fatal error, 3xx = error,
// 4xx = ignorable errors; per-event replies pair positionally with the
// submitted events. Only a per-event 1xx is confirmation. A reply whose
// per-event list does not have exactly one entry per submitted event cannot
// be correlated and leaves every event unconfirmed (unknown_delivery).
func edsmEventOutcomes(raw []byte, count int) []eventOutcome {
	all := func(status, reason string) []eventOutcome {
		out := make([]eventOutcome, count)
		for i := range out {
			out[i] = eventOutcome{status, reason}
		}
		return out
	}
	v := object(raw)
	top, ok := msgnum(v)
	if v == nil || !ok {
		return all("unknown_delivery", "EDSM reply unreadable; delivery unconfirmed, not retried")
	}
	switch top / 100 {
	case 2:
		return all("rejected", "EDSM refused the batch (code "+strconv.FormatInt(top, 10)+")")
	case 5:
		return all("accepted_for_later", "EDSM saved the events for later processing (code "+strconv.FormatInt(top, 10)+"); not yet confirmed")
	case 1:
	default:
		return all("unknown_delivery", "Unexpected EDSM batch code "+strconv.FormatInt(top, 10)+"; delivery unconfirmed")
	}
	events, ok := v["events"].([]any)
	if !ok || len(events) != count {
		return all("unknown_delivery", "EDSM per-event receipts missing or not matching the submitted events; delivery unconfirmed")
	}
	out := make([]eventOutcome, count)
	for i, e := range events {
		o, _ := e.(map[string]any)
		code, ok := msgnum(o)
		label := strconv.FormatInt(code, 10)
		switch {
		case !ok:
			out[i] = eventOutcome{"unknown_delivery", "EDSM per-event receipt unreadable"}
		case code/100 == 1:
			out[i] = eventOutcome{"confirmed", "EDSM accepted the event (code " + label + ")"}
		case code/100 == 4:
			out[i] = eventOutcome{"rejected", "EDSM discarded the event as an ignorable error (code " + label + ")"}
		case code/100 == 2 || code/100 == 3:
			out[i] = eventOutcome{"rejected", "EDSM rejected the event (code " + label + ")"}
		default:
			out[i] = eventOutcome{"unknown_delivery", "Unexpected EDSM event code " + label}
		}
	}
	return out
}

func msgnum(v map[string]any) (int64, bool) {
	if v == nil {
		return 0, false
	}
	n, ok := v["msgnum"].(json.Number)
	if !ok {
		return 0, false
	}
	code, err := n.Int64()
	return code, err == nil
}
