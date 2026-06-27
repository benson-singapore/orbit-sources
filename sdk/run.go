package sdk

import (
	"encoding/json"
	"fmt"
	"os"
)

// Plugin is implemented by each WASM feed plugin.
type Plugin interface {
	Fetch(req *FetchRequest) (*FeedResult, error)
}

type envelope struct {
	Action string          `json:"action"`
	Data   json.RawMessage `json:"data"`
}

type response struct {
	OK    bool   `json:"ok"`
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

// Run reads one JSON request from stdin and writes one JSON response to stdout.
func Run(p Plugin) {
	var env envelope
	if err := json.NewDecoder(os.Stdin).Decode(&env); err != nil {
		writeError(err)
		return
	}
	switch env.Action {
	case "fetch":
		var req FetchRequest
		if err := json.Unmarshal(env.Data, &req); err != nil {
			writeError(err)
			return
		}
		result, err := p.Fetch(&req)
		if err != nil {
			writeError(err)
			return
		}
		writeOK(result)
	default:
		writeError(fmt.Errorf("unknown action: %s", env.Action))
	}
}

func writeOK(data any) {
	_ = json.NewEncoder(os.Stdout).Encode(response{OK: true, Data: data})
}

func writeError(err error) {
	_ = json.NewEncoder(os.Stdout).Encode(response{OK: false, Error: err.Error()})
}
