// Copyright (c) 2016 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package main

import (
	"errors"
	"os/exec"
	"strings"

	"gopkg.in/yaml.v2"
)

type Session map[interface{}]SessionHost

type SessionHost struct {
	Bridge SessionBridge
	VHosts []*SessionVHost
}

type SessionBridge struct {
	Device string
	Iface  string
	Peers  []struct {
		Device string
		Host   string
	}
}

type SessionVHost struct {
	Device    string
	Iface     string
	Namespace string
	Target    string
}

func NewSession(cfg *configYaml) (Session, error) {
	// seshBts, err := exec.Command("cs", "create", "10.10.0.0/8", "h1/40", "h2/40", "h3/40").Output()
	// if err != nil {
	// 	return nil, err
	// }

	//TODO(wieger): Config to Session

	seshBts := []byte(seshStr)

	var s Session
	err := yaml.Unmarshal(seshBts, &s)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (s Session) StartedHosts() []string {
	var ret []string
	for _, h := range s {
		for _, vh := range h.VHosts {
			split := strings.Split(vh.Iface, "/")
			// ret = append(ret, split[0]+":"+ringpopPort)
			ret = append(ret, split[0])
		}
	}

	return ret
}

func (s Session) Start(n int) bool {
	for _, host := range s {
		for _, vh := range host.VHosts {
			if vh.Target == "started" {
				continue
			}
			vh.Target = "started"
			n--
			if n == 0 {
				return true
			}
		}
	}
	return false
}

func (s Session) Stop(n int) bool {
	for _, host := range s {
		for _, vh := range host.VHosts {
			if vh.Target == "stopped" {
				continue
			}
			vh.Target = "stopped"
			n--
			if n == 0 {
				return true
			}
		}
	}
	return false
}

func (s Session) StartAll() {
	for _, host := range s {
		for _, vh := range host.VHosts {
			vh.Target = "started"
		}
	}
}

func (s Session) StopAll() {
	for _, host := range s {
		for _, vh := range host.VHosts {
			vh.Target = "stopped"
		}
	}
}

func (s Session) Prepare() error {
	cmd := exec.Command("cs", "prepare", "./testpop")
	cmd.Start()
	s.writeToStdin(cmd)
	return cmd.Wait()
}

func (s Session) Apply() error {
	cmd := exec.Command("cs", "apply")
	cmd.Start()
	s.writeToStdin(cmd)
	return cmd.Wait()
}

func (s Session) writeToStdin(cmd *exec.Cmd) error {
	in, err := yaml.Marshal(s)
	if err != nil {
		return err
	}
	w, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	n, err := w.Write(in)
	if err != nil {
		return err
	}
	if n != len(in) {
		return errors.New("not written full Session to apply command")
	}

	return w.Close()
}

var seshStr = `
host1:
  bridge:
    device: vc_br0
    iface: 10.0.255.254/16
    peers:
    - {device: cv_vxlan0, host: t1}
  vhosts:
  - {device: vc_tap0, iface: 10.0.0.1/16, namespace: ns0, target: stopped}
  - {device: vc_tap1, iface: 10.0.0.2/16, namespace: ns1, target: stopped}
  - {device: vc_tap2, iface: 10.0.0.3/16, namespace: ns2, target: stopped}
  - {device: vc_tap3, iface: 10.0.0.4/16, namespace: ns3, target: stopped}
host2:
  bridge:
    device: vc_br0
    iface: 10.0.255.253/16
    peers:
    - {device: cv_vxlan0, host: commitverse.org}
  vhosts:
  - {device: vc_tap0, iface: 10.0.0.5/16, namespace: ns0, target: stopped}
  - {device: vc_tap1, iface: 10.0.0.6/16, namespace: ns1, target: stopped}
  - {device: vc_tap2, iface: 10.0.0.7/16, namespace: ns2, target: stopped}
  - {device: vc_tap3, iface: 10.0.0.8/16, namespace: ns3, target: stopped}
`