// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package main

import (
	"fmt"
	"os"
	"testing"
	"time"
)

const (
	_ = iota
	EVENT
	FUNC
)

type testTable []testPhase

type testPhase struct {
	phaseType int
	events    []testEvent
	chkFunc   func() error
}

type testEvent struct {
	p           string
	srcip       string
	durationSub string
	n           int
}

// Tests invalidation of internal addresses
var testtab0 = testTable{
	{
		phaseType: EVENT,
		events: []testEvent{
			{"user@host.com", "10.0.0.1", "", 1},
			{"user@host.com", "172.16.0.1", "", 1},
			{"user@host.com", "192.168.100.1", "", 1},
			{"user@host.com", "0.0.0.0", "", 1},
			{"user@host.com", "63.245.214.133", "", 1},
		},
	},
	{
		phaseType: FUNC,
		chkFunc:   testtab0Func,
	},
}

// Ensure weight threshold is zero based on no deviation in model
var testtab1 = testTable{
	{
		phaseType: EVENT,
		events: []testEvent{
			{"user@host.com", "63.245.214.133", "120h", 4},
		},
	},
	{
		phaseType: EVENT,
		events: []testEvent{
			{"user@host.com", "63.245.214.133", "", 4},
		},
	},
	{
		phaseType: FUNC,
		chkFunc:   testtab1Func,
	},
}

var testtab2 = testTable{
	{
		phaseType: EVENT,
		events: []testEvent{
			{"user@host.com", "63.245.214.133", "", 20},
			{"user@host.com", "118.163.10.187", "", 1},
		},
	},
	{
		phaseType: FUNC,
		chkFunc:   testtab2Func,
	},
}

var testtab3 = testTable{
	{
		phaseType: EVENT,
		events: []testEvent{
			{"user@host.com", "63.245.214.133", "", 15},
			{"user@host.com", "118.163.10.187", "", 15},
		},
	},
	{
		phaseType: FUNC,
		chkFunc:   testtab3Func,
	},
}

var testtab4 = testTable{
	{
		phaseType: EVENT,
		events: []testEvent{
			{"user@host.com", "63.245.214.133", "", 15},
		},
	},
	{
		phaseType: FUNC,
		chkFunc:   testtab4FuncPre,
	},
	{
		phaseType: EVENT,
		events: []testEvent{
			{"user@host.com", "63.245.214.133", "", 15},
			{"user@host.com", "118.163.10.187", "", 5},
		},
	},
	{
		phaseType: FUNC,
		chkFunc:   testtab4FuncPost,
	},
}

var testtab5 = testTable{
	{
		phaseType: EVENT,
		events: []testEvent{
			{"user@host.com", "63.245.214.133", "2000h", 20},
			{"user@host.com", "63.245.214.133", "72h", 10},
		},
	},
	{
		phaseType: FUNC,
		chkFunc:   testtab5FuncPre,
	},
	{
		phaseType: EVENT,
		events: []testEvent{
			{"user@host.com", "63.245.214.133", "", 10},
		},
	},
	{
		phaseType: FUNC,
		chkFunc:   testtab5FuncPost,
	},
}

var testtab6 = testTable{
	{
		phaseType: EVENT,
		events: []testEvent{
			{"user@host.com", "63.245.214.133", "336h", 5},
			{"login@host.org", "63.245.214.133", "336h", 5},
		},
	},
	{
		phaseType: EVENT,
		events: []testEvent{
			{"user@host.com", "63.245.214.133", "312h", 5},
			{"login@host.org", "63.245.214.133", "312h", 5},
		},
	},
	{
		phaseType: EVENT,
		events: []testEvent{
			{"user@host.com", "63.245.214.133", "288h", 5},
			{"user@host.com", "118.163.10.187", "288h", 2},
			{"login@host.org", "63.245.214.133", "288h", 5},
		},
	},
	{
		phaseType: EVENT,
		events: []testEvent{
			{"user@host.com", "63.245.214.133", "264h", 5},
			{"login@host.org", "63.245.214.133", "264h", 5},
		},
	},
	{
		phaseType: EVENT,
		events: []testEvent{
			{"user@host.com", "63.245.214.133", "240h", 5},
			{"login@host.org", "63.245.214.133", "240h", 5},
		},
	},
	{
		phaseType: EVENT,
		events: []testEvent{
			{"user@host.com", "63.245.214.133", "216h", 5},
			{"login@host.org", "63.245.214.133", "216h", 5},
		},
	},
	{
		phaseType: EVENT,
		events: []testEvent{
			{"user@host.com", "63.245.214.133", "192h", 5},
			{"login@host.org", "63.245.214.133", "192h", 5},
		},
	},
	{
		phaseType: EVENT,
		events: []testEvent{
			{"user@host.com", "63.245.214.133", "168h", 5},
			{"login@host.org", "63.245.214.133", "168h", 5},
		},
	},
	{
		phaseType: FUNC,
		chkFunc:   testtab6FuncEntry1,
	},
	{
		phaseType: EVENT,
		events: []testEvent{
			{"user@host.com", "63.245.214.133", "144h", 5},
			{"user@host.com", "63.245.214.133", "120h", 5},
			{"user@host.com", "63.245.214.133", "96h", 5},
			{"user@host.com", "63.245.214.133", "72h", 5},
			{"user@host.com", "63.245.214.133", "48h", 5},
			{"user@host.com", "63.245.214.133", "24h", 5},
			{"user@host.com", "63.245.214.133", "", 5},
		},
	},
	{
		phaseType: FUNC,
		chkFunc:   testtab6FuncFinal,
	},
}

type simpleStateService struct {
	store map[string]object
}

func (s *simpleStateService) readObject(objid string) (o *object, err error) {
	r, ok := s.store[objid]
	if !ok {
		return nil, nil
	}
	return &r, nil
}

func (s *simpleStateService) writeObject(o object) (err error) {
	s.store[o.ObjectID] = o
	return nil
}

func (s *simpleStateService) doInit() (err error) {
	s.store = make(map[string]object)
	return nil
}

func (s *simpleStateService) getStore() map[string]object {
	return s.store
}

func makePhaseResults(tp []testEvent) (pluginResult, error) {
	var ret pluginResult

	ret.Results = make([]eventResult, 0)
	for i := range tp {
		for j := 1; j <= tp[i].n; j++ {
			newres := eventResult{}
			newres.Timestamp = time.Now().UTC()
			if tp[i].durationSub != "" {
				dur, err := time.ParseDuration(tp[i].durationSub)
				if err != nil {
					return ret, err
				}
				newres.Timestamp = newres.Timestamp.Add(-1 * dur)
			}
			newres.Principal = tp[i].p
			newres.SourceIPV4 = tp[i].srcip
			newres.Name = "test"
			newres.Valid = true
			err := newres.validate()
			if err != nil {
				return ret, err
			}
			ret.Results = append(ret.Results, newres)
		}
	}

	return ret, nil
}

func runTestPhase(tp testPhase) error {
	switch tp.phaseType {
	case EVENT:
		ev, err := makePhaseResults(tp.events)
		if err != nil {
			return err
		}
		integrate(ev)
		err = integrationMergeQueue()
		if err != nil {
			return err
		}
	case FUNC:
		err := tp.chkFunc()
		if err != nil {
			return err
		}
	}
	return nil
}

func runTestTable(testtab testTable, t *testing.T) {
	defer func() {
		if e := recover(); e != nil {
			t.Fatalf("test failed: %v", e)
		}
	}()

	err := testGenericInit()
	if err != nil {
		panic(err)
	}
	for i := range testtab {
		err := runTestPhase(testtab[i])
		if err != nil {
			panic(err)
		}
	}
}

func testGenericInit() error {
	cfg.General.MaxMind = os.Getenv("TESTMMF")
	cfg.Geo.CollapseMaximum = 500
	cfg.Timer.ExpireEvents = "720h"
	cfg.noSendAlert = true
	err := maxmindInit()
	if err != nil {
		return err
	}
	setStateService(&simpleStateService{})
	getStateService().(*simpleStateService).doInit()
	return nil
}

func testtab0Func() error {
	s := getStateService().(*simpleStateService).getStore()
	cnt := len(s)
	if cnt != 1 {
		return fmt.Errorf("more than one entry in state")
	}
	for _, v := range s {
		if len(v.Results) != 1 {
			panic("more than one result in state entry")
		}
		if v.Results[0].SourceIPV4 != "63.245.214.133" {
			panic("result we had was not the correct one")
		}
	}
	return nil
}

func testtab1Func() error {
	s := getStateService().(*simpleStateService).getStore()
	for _, v := range s {
		if v.WeightDeviation != 0 {
			return fmt.Errorf("weight deviation was not 0")
		}
		if v.NumCenters != 1 {
			return fmt.Errorf("incorrect number of geocenters")
		}
	}
	return nil
}

func testtab2Func() error {
	s := getStateService().(*simpleStateService).getStore()
	if len(s) != 1 {
		return fmt.Errorf("more than one entry in state")
	}
	for _, v := range s {
		if v.WeightDeviation == 0 {
			return fmt.Errorf("weight threshold was 0")
		}
		if v.NumCenters != 2 {
			return fmt.Errorf("incorrect number of geocenters")
		}
		collapseCnt := 0
		for _, x := range v.Results {
			if x.Collapsed {
				collapseCnt++
			}
		}
		if collapseCnt != 19 {
			return fmt.Errorf("incorrect number of collapsed results")
		}
		for _, x := range v.Results {
			if !x.Escalated {
				return fmt.Errorf("a result entry was not escalated")
			}
		}
	}
	return nil
}

func testtab3Func() error {
	s := getStateService().(*simpleStateService).getStore()
	if len(s) != 1 {
		return fmt.Errorf("more than one entry in state")
	}
	for _, v := range s {
		if v.WeightDeviation != 0 {
			return fmt.Errorf("weight threshold was not 0")
		}
		if v.NumCenters != 2 {
			return fmt.Errorf("incorrect number of geocenters")
		}
		collapseCnt := 0
		for _, x := range v.Results {
			if x.Collapsed {
				collapseCnt++
			}
		}
		if collapseCnt != 28 {
			return fmt.Errorf("incorrect number of collapsed results")
		}
		for _, x := range v.Results {
			if !x.Escalated {
				return fmt.Errorf("a result entry was not escalated")
			}
		}
	}
	return nil
}

func testtab4FuncPre() error {
	s := getStateService().(*simpleStateService).getStore()
	if len(s) != 1 {
		return fmt.Errorf("more than one entry in state")
	}
	for _, v := range s {
		if v.WeightDeviation != 0 {
			return fmt.Errorf("weight threshold was not 0")
		}
		if v.NumCenters != 1 {
			return fmt.Errorf("incorrect number of geocenters")
		}
		collapseCnt := 0
		for _, x := range v.Results {
			if x.Collapsed {
				collapseCnt++
			}
		}
		if collapseCnt != 14 {
			return fmt.Errorf("incorrect number of collapsed results")
		}
		for _, x := range v.Results {
			if !x.Escalated {
				return fmt.Errorf("a result entry was not escalated")
			}
		}
	}
	return nil
}

func testtab4FuncPost() error {
	s := getStateService().(*simpleStateService).getStore()
	if len(s) != 1 {
		return fmt.Errorf("more than one entry in state")
	}
	for _, v := range s {
		if v.WeightDeviation == 0 {
			return fmt.Errorf("weight threshold was 0")
		}
		if v.NumCenters != 2 {
			return fmt.Errorf("incorrect number of geocenters")
		}
		collapseCnt := 0
		for _, x := range v.Results {
			if x.Collapsed {
				collapseCnt++
			}
		}
		if collapseCnt != 33 {
			return fmt.Errorf("incorrect number of collapsed results")
		}
		for _, x := range v.Results {
			if !x.Escalated {
				return fmt.Errorf("a result entry was not escalated")
			}
		}

		// Locate the branch entry last created and validate alert
		// generation
		testStr := "user@host.com NEWLOCATION Taipei, Taiwan access from "
		testStr += "118.163.10.187 (test) [deviation:12.5]"
		testStr += " last activity was from San Francisco, United States within hour before"
		var o objectResult
		for _, x := range v.Results {
			if x.Collapsed {
				continue
			}
			if x.SourceIPV4 != "118.163.10.187" {
				continue
			}
			o = x
			break
		}
		ad, err := v.createAlertDetails(o.BranchID)
		if err != nil {
			return err
		}
		err = ad.addPreviousEvent(&v, o.BranchID)
		if err != nil {
			return err
		}
		sumstr := ad.makeSummary()
		if sumstr != testStr {
			return fmt.Errorf("alert summary did not match")
		}
	}
	return nil
}

func testtab5FuncPre() error {
	s := getStateService().(*simpleStateService).getStore()
	if len(s) != 1 {
		return fmt.Errorf("more than one entry in state")
	}
	for _, v := range s {
		if len(v.Results) != 10 {
			return fmt.Errorf("incorrect number of stored results in pre func")
		}
	}
	return nil
}

func testtab5FuncPost() error {
	s := getStateService().(*simpleStateService).getStore()
	if len(s) != 1 {
		return fmt.Errorf("more than one entry in state")
	}
	for _, v := range s {
		if len(v.Results) != 20 {
			return fmt.Errorf("incorrect number of stored results in post func")
		}
		if v.NumCenters != 1 {
			return fmt.Errorf("incorrect number of geocenters")
		}
	}
	return nil
}

func testtab6FuncEntry1() error {
	s := getStateService().(*simpleStateService).getStore()
	if len(s) != 2 {
		return fmt.Errorf("incorrect number of entries in state")
	}
	oid, err := getObjectID("user@host.com")
	if err != nil {
		return err
	}
	uhent := s[oid]
	oid, err = getObjectID("login@host.org")
	if err != nil {
		return err
	}
	lhent := s[oid]
	if len(uhent.Results) != 42 {
		return fmt.Errorf("incorrect number of events for user@host.com in entry1")
	}
	if uhent.NumCenters != 2 {
		return fmt.Errorf("incorrect number of geocenters for user@host.com in entry1")
	}
	if len(lhent.Results) != 40 {
		return fmt.Errorf("incorrect number of events for login@host.org in entry1")
	}
	if lhent.NumCenters != 1 {
		return fmt.Errorf("incorrect number of geocenters for login@host.org in entry1")
	}

	// Change the stored expiry duration value, which will result in some
	// of the events entered in the previous phase being removed
	cfg.Timer.ExpireEvents = "250h"
	return nil
}

func testtab6FuncFinal() error {
	s := getStateService().(*simpleStateService).getStore()
	if len(s) != 2 {
		return fmt.Errorf("incorrect number of entries in state")
	}
	oid, err := getObjectID("user@host.com")
	if err != nil {
		return err
	}
	uhent := s[oid]
	oid, err = getObjectID("login@host.org")
	if err != nil {
		return err
	}
	lhent := s[oid]
	if len(uhent.Results) != 55 {
		return fmt.Errorf("incorrect number of events for user@host.com in final")
	}
	if len(lhent.Results) != 40 {
		return fmt.Errorf("incorrect number of events for login@host.org in final")
	}
	return nil
}

func TestAnalyzeTab0(t *testing.T) {
	runTestTable(testtab0, t)
}

func TestAnalyzeTab1(t *testing.T) {
	runTestTable(testtab1, t)
}

func TestAnalyzeTab2(t *testing.T) {
	runTestTable(testtab2, t)
}

func TestAnalyzeTab3(t *testing.T) {
	runTestTable(testtab3, t)
}

func TestAnalyzeTab4(t *testing.T) {
	runTestTable(testtab4, t)
}

func TestAnalyzeTab5(t *testing.T) {
	runTestTable(testtab5, t)
}

func TestAnalyzeTab6(t *testing.T) {
	runTestTable(testtab6, t)
}
