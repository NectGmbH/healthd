package main

import (
    "context"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"
    "time"

    "github.com/sirupsen/logrus"
    etcdClient "go.etcd.io/etcd/client"
)

// Healthd is the service used for receiveing status updates and persisting them in etcd
type Healthd struct {
    etcds  StringSlice
    client etcdClient.Client
}

// NewHealthd creates a new service for receiving status updates and persisting them to etcd
func NewHealthd(etcds StringSlice) (*Healthd, error) {
    h := &Healthd{
        etcds: etcds,
    }

    var err error
    h.client, err = etcdClient.New(etcdClient.Config{
        Endpoints: etcds,
    })
    if err != nil {
        return nil, fmt.Errorf("couldn't connect to etcds `%#v`, see: %v", etcds, err)
    }

    return h, nil
}

// StatusUpdate represents the struct which should be passed to etcd
type StatusUpdate struct {
    Status []interface{}
    Time   int64
}

// HTTPHandler is the handlerFunc used as endpoint for receiving status updates
func (h *Healthd) HTTPHandler(w http.ResponseWriter, r *http.Request) {
    agentName := r.Header.Get("X-Agent-Name")
    if agentName == "" {
        errStr := "request missing X-Agent-Name header, ignoring."
        logrus.Error(errStr)
        http.Error(w, errStr, http.StatusBadRequest)
        return
    }

    if r.ContentLength <= 0 || r.ContentLength > 1000000 {
        errStr := fmt.Sprintf("invalid content length, expected some bytes with a max of 1mb, got `%d`", r.ContentLength)
        logrus.Error(errStr)
        http.Error(w, errStr, http.StatusBadRequest)
        return
    }

    body, err := ioutil.ReadAll(r.Body)
    if err != nil {
        errStr := fmt.Sprintf("couldn't read request body, see: %v", err)
        logrus.Error(errStr)
        http.Error(w, errStr, http.StatusBadRequest)
        return
    }

    var data []interface{}
    err = json.Unmarshal(body, &data)
    if err != nil {
        errStr := fmt.Sprintf("couldn't parse `%s` as json, see: %v", string(body), err)
        logrus.Error(errStr)
        http.Error(w, errStr, http.StatusBadRequest)
        return
    }

    update := &StatusUpdate{
        Status: data,
        Time:   time.Now().Unix(),
    }

    body, err = json.Marshal(update)
    if err != nil {
        errStr := fmt.Sprintf("couldn't serialize `%s` as json, see: %v", string(body), err)
        logrus.Error(errStr)
        http.Error(w, errStr, http.StatusInternalServerError)
        return
    }

    kapi := etcdClient.NewKeysAPI(h.client)
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    _, err = kapi.Set(ctx, agentName, string(body), nil)
    cancel()
    if err != nil {
        errStr := fmt.Sprintf("couldn't persist status update to etcd, see: %v", err)
        logrus.Error(errStr)
        http.Error(w, errStr, http.StatusInternalServerError)
    }

    w.WriteHeader(http.StatusOK)
    logrus.Infof("Updated state for agent `%s` to `%#v`", agentName, string(body))

    return
}
