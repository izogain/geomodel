// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	elastigo "github.com/mattbaird/elastigo/lib"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
)

// Describes input sent to a plugin; primarily a slice of raw JSON documents
// that have been returned by ES
type pluginRequest struct {
	Events []*json.RawMessage `json:"events"` // Slice of matching plugin events
}

// Describes the result of execution of a plugin; plugins must return data
// that conforms to this structure.
type pluginResult struct {
	Results []eventResult `json:"results"` // Slice of event results
}

func (p *pluginResult) validate() error {
	for i := range p.Results {
		err := p.Results[i].validate()
		if err != nil {
			return err
		}
	}
	return nil
}

// Corresponds to an individual event in a plugin result
type eventResult struct {
	Principal  string `json:"principal"`   // Authentication principal, identifier
	SourceIPV4 string `json:"source_ipv4"` // Source IPV4 for authentication
	Valid      bool   `json:"valid"`       // True if entry was parsed correctly by plugin
	Name       string `json:"name"`        // Name of plugin that created result
}

func (e *eventResult) validate() error {
	if e.Name == "" {
		return fmt.Errorf("plugin result has no name")
	}
	if !e.Valid {
		return nil
	}
	if e.Principal == "" {
		return fmt.Errorf("plugin result has no principal value")
	}
	if e.SourceIPV4 == "" {
		return fmt.Errorf("plugin result has no source_ipv4 value")
	}
	return nil
}

type plugin struct {
	name        string
	path        string
	searchTerms []pluginTerm
}

func (p *plugin) runPlugin(input []byte) (err error) {
	var output bytes.Buffer
	cmd := exec.Command(p.path)
	cmd.Stdin = bytes.NewReader(input)
	cmd.Stdout = &output
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("error running %v: %v", p.path, err)
	}
	var res pluginResult
	err = json.Unmarshal(output.Bytes(), &res)
	if err != nil {
		return fmt.Errorf("error parsing output from %v: %v", p.path, err)
	}
	err = res.validate()
	if err != nil {
		return fmt.Errorf("error validating plugin output from %v: %v", p.name, err)
	}
	pluginResultCh <- res
	return nil
}

type pluginTerm struct {
	key   string
	value string
}

var pluginList []plugin

// Given an event query response from ES, return a byte slice suitable to be
// passed to a plugin
func pluginRequestDataFromES(r elastigo.SearchResult) (ret []byte, err error) {
	pr := pluginRequest{}
	for _, x := range r.Hits.Hits {
		pr.Events = append(pr.Events, x.Source)
	}
	ret, err = json.Marshal(pr)
	if err != nil {
		return ret, err
	}
	return ret, nil
}

func pluginFromFile(ppath string) (plugin, error) {
	np := plugin{}
	np.path = ppath

	fd, err := os.Open(ppath)
	if err != nil {
		return np, err
	}
	defer fd.Close()

	scnr := bufio.NewScanner(fd)
	for scnr.Scan() {
		buf := scnr.Text()
		args := strings.Split(buf, " ")
		if len(args) < 3 {
			continue
		}
		if args[0] != "#" {
			continue
		}
		if args[1] == "@@" && len(args) >= 3 {
			np.name = args[2]
		} else if args[1] == "@T" && len(args) >= 4 {
			nterm := pluginTerm{}
			nterm.key = args[2]
			nterm.value = args[3]
			np.searchTerms = append(np.searchTerms, nterm)
		}
	}
	err = scnr.Err()
	if err != nil {
		return np, err
	}

	return np, nil
}

func loadPlugins() error {
	dirents, err := ioutil.ReadDir(cfg.General.Plugins)
	if err != nil {
		return err
	}
	for _, x := range dirents {
		if !strings.HasSuffix(x.Name(), ".py") {
			continue
		}
		fname := path.Join(cfg.General.Plugins, x.Name())
		newplugin, err := pluginFromFile(fname)
		if err != nil {
			return err
		}
		pluginList = append(pluginList, newplugin)
		logf("added plugin %v (%v terms)", newplugin.name, len(newplugin.searchTerms))
	}
	return nil
}
