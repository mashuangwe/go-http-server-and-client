package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"

	"github.com/astaxie/beego"
)

func goBuild(svVersion, releaseVersion string) error {
	// command := `./build_script.sh dev2.0 portal.v.0.0.1`
	command := `./build_script.sh ` + svVersion + ` ` + releaseVersion
	cmd := exec.Command("/bin/bash", "-c", command)

	output, err := cmd.Output()
	if err != nil {
		beego.Error("Execute Shell:%s failed with error:%s", command, err.Error())
		return err
	}
	beego.Info(string(output))
	return nil
}

type BuildInfo struct {
	SvVersion      string `json:"svVersion"`
	ReleaseVersion string `json:"releaseVersion"`
}

func build(w http.ResponseWriter, r *http.Request) {
	body, _ := ioutil.ReadAll(r.Body)
	body_str := string(body)
	beego.Debug(body_str)
	var buildInfo BuildInfo

	if err := json.Unmarshal(body, &buildInfo); err == nil {
		if goBuild(buildInfo.SvVersion, buildInfo.ReleaseVersion) == nil {
			portalUrl := "http://172.16.13.117:8000/sv/" + buildInfo.ReleaseVersion
			fmt.Fprintf(w, portalUrl)
		}
		beego.Debug(err)
	} else {
		beego.Debug(err)
	}
}

func main() {
	http.HandleFunc("/", build)

	if err := http.ListenAndServe("172.16.13.117:8001", nil); err != nil {
		beego.Debug("ListenAndServe: ", err)
	}
}
