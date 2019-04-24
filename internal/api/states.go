package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/hashicorp/terraform/state"
	log "github.com/sirupsen/logrus"
)

func (s *server) InsertState(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	timestamp, ok := params["timestamp"]
	if !ok {
		timestamp = time.Now().Format("20060102150405")
	}

	source, ok := params["source"]
	if !ok {
		source = "direct"
	}

	var document interface{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&document)
	if err != nil {
		log.Errorf("failed to decode body: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("500 - Internal server error: %s", err)))
		return
	}

	err = s.st.InsertState(document, timestamp, source, params["name"])
	if err != nil {
		log.Errorf("failed to insert state: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("500 - Internal server error: %s", err)))
		return
	}

	w.WriteHeader(http.StatusOK)
	return
}

func (s *server) ListStates(w http.ResponseWriter, r *http.Request) {
	states, err := s.st.ListStates()
	if err != nil {
		log.Errorf("failed to retrieve states: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("500 - Internal server error: %s", err)))
		return
	}

	data, err := json.Marshal(states)
	if err != nil {
		log.Errorf("failed to marshal states: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("500 - Internal server error: %s", err)))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
	return
}

func (s *server) GetState(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	var serial int
	if v := r.URL.Query().Get("serial"); v != "" {
		var err error
		serial, err = strconv.Atoi(v)
		if err != nil {
			log.Errorf("failed to parse serial: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("500 - Internal server error: %s", err)))
			return
		}
	}

	document, err := s.st.GetState(params["name"], serial)
	if err != nil {
		log.Errorf("failed to retrieve latest state: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("500 - Internal server error: %s", err)))
		return
	}

	if document == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	data, err := json.Marshal(document)
	if err != nil {
		log.Errorf("failed to marshal state: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("500 - Internal server error: %s", err)))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
	return
}

func (s *server) ListStateSerials(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	serials, err := s.st.ListStateSerials(params["name"])
	if err != nil {
		log.Errorf("failed to retrieve state serials: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("500 - Internal server error: %s", err)))
		return
	}

	data, err := json.Marshal(serials)
	if err != nil {
		log.Errorf("failed to marshal state serials: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("500 - Internal server error: %s", err)))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
	return
}

func (s *server) RemoveState(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	err := s.st.RemoveState(params["name"])
	if err != nil {
		log.Errorf("failed to remove state: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("500 - Internal server error: %s", err)))
		return
	}

	w.WriteHeader(http.StatusOK)
	return
}

func (s *server) LockState(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	var currentLock, remoteLock *state.LockInfo

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Errorf("failed to read body: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("500 - Internal server error: %s", err)))
		return
	}

	err = json.Unmarshal(body, &currentLock)
	if err != nil {
		log.Errorf("failed to unmarshal lock: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("500 - Internal server error: %s", err)))
		return
	}

	lock, err := s.st.GetLockStatus(params["name"])
	if err != nil {
		log.Errorf("failed to get lock status: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("500 - Internal server error: %s", err)))
		return
	}

	if lock != nil {
		remoteLock = lock.(*state.LockInfo)

		if currentLock.ID == remoteLock.ID {
			d, _ := json.Marshal(lock)
			w.WriteHeader(http.StatusLocked)
			w.Write(d)
			return
		}

		d, _ := json.Marshal(remoteLock)
		w.WriteHeader(http.StatusConflict)
		w.Write(d)
		return
	}

	err = s.st.LockState(params["name"], currentLock)
	if err != nil {
		log.Errorf("failed to lock state: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("500 - Internal server error: %s", err)))
		return
	}

	w.WriteHeader(http.StatusOK)
	return
}

func (s *server) UnlockState(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	var lockData *state.LockInfo

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Errorf("failed to read body: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("500 - Internal server error: %s", err)))
		return
	}

	err = json.Unmarshal(body, &lockData)
	if err != nil {
		log.Errorf("failed to unmarshal lock: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("500 - Internal server error: %s", err)))
		return
	}

	err = s.st.UnlockState(params["name"], lockData)
	if err != nil {
		log.Errorf("failed to unlock state: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("500 - Internal server error: %s", err)))
		return
	}

	w.WriteHeader(http.StatusOK)
	return

}
