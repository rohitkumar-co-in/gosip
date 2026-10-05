package api

import (
	"net/http"
	"strconv"
)

type messageActorKey struct{}

func (h *BusinessHandler) Activity(w http.ResponseWriter, r *http.Request) {
	offset, err := strconv.Atoi(r.URL.Query().Get("offset"))
	if r.URL.Query().Get("offset") == "" {
		offset = 0
		err = nil
	}
	if err != nil || offset < 0 {
		WriteValidationError(w, "Invalid activity offset", nil)
		return
	}
	kind := r.URL.Query().Get("kind")
	if kind != "" && kind != "call" && kind != "sms" {
		WriteValidationError(w, "Invalid activity kind", nil)
		return
	}
	query := `WITH activity AS (
 SELECT 'call' kind,id,direction,from_number,to_number,COALESCE(disposition,'unknown') status,strftime('%Y-%m-%dT%H:%M:%fZ',started_at) occurred_at,duration,'' body FROM cdrs
 UNION ALL SELECT 'sms',id,direction,from_number,to_number,COALESCE(status,'unknown'),strftime('%Y-%m-%dT%H:%M:%fZ',created_at),0,COALESCE(body,'') FROM messages)
 SELECT x.kind,x.id,x.direction,x.from_number,x.to_number,x.status,x.occurred_at,x.duration,x.body,COALESCE(a.actor,'Actor not captured')
 FROM activity x LEFT JOIN activity_actors a ON a.kind=x.kind AND a.record_id=x.id
 WHERE (?='' OR x.kind=?) ORDER BY x.occurred_at DESC,x.id DESC LIMIT 50 OFFSET ?`
	rows, err := h.deps.DB.Conn().QueryContext(r.Context(), query, kind, kind, offset)
	if err != nil {
		WriteInternalError(w)
		return
	}
	defer rows.Close()
	data := []map[string]interface{}{}
	for rows.Next() {
		var k, d, f, t, s, at, b, actor string
		var id int64
		var duration int
		if err = rows.Scan(&k, &id, &d, &f, &t, &s, &at, &duration, &b, &actor); err != nil {
			WriteInternalError(w)
			return
		}
		data = append(data, map[string]interface{}{"kind": k, "id": id, "direction": d, "from_number": f, "to_number": t, "status": s, "occurred_at": at, "duration": duration, "body": b, "actor": actor})
	}
	if rows.Err() != nil {
		WriteInternalError(w)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	WriteJSON(w, 200, map[string]interface{}{"data": data, "offset": offset, "limit": 50})
}
