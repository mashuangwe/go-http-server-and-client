package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type BuildInfo struct {
	SvVersion      string `json:"svVersion"`
	ReleaseVersion string `json:"releaseVersion"`
}

func main() {
	buildInfo := BuildInfo{
		SvVersion:      "dev2.0",
		ReleaseVersion: "portal.v.0.0.2",
	}

	if bb, err := json.Marshal(buildInfo); err == nil {
		req := bytes.NewBuffer(bb)

		body_type := "application/json;charset=utf-8"
		resp, _ := http.Post("http://172.16.13.117:8001/", body_type, req)
		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Println(string(body))
	} else {
		fmt.Println(err)
	}
}
