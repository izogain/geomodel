// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Julien Vehent jvehent@mozilla.com
// - Aaron Meihm ameihm@mozilla.com
//
// This code is directly based off work done by Julien Vehent in the
// geolog project. See https://github.com/jvehent/geolog.

package main

import (
	"fmt"
	geo "github.com/oschwald/geoip2-golang"
	"math"
	"net"
)

var maxmind *geo.Reader

func maxmindInit() (err error) {
	maxmind, err = geo.Open(cfg.General.MaxMind)
	if err != nil {
		return err
	}
	logf("initialized maxmind db")
	return nil
}

func geoObjectResult(o *objectResult) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("geoObjectResult() -> %v", e)
		}
	}()

	record, err := maxmind.City(net.ParseIP(o.SourceIPV4))
	if err != nil {
		panic(err)
	}
	o.Latitude = record.Location.Latitude
	o.Longitude = record.Location.Longitude
	o.Locality = record.City.Names["en"] + ", " + record.Country.Names["en"]
	o.Weight = 1

	return nil
}

// Collapse branches in the object based on proximity to tres; the number of
// branches collapsed during the operation is returned
func geoCollapseUsing(o *object, tres objectResult) float64 {
	var ret float64 = 0
	for i := range o.Results {
		p0 := &o.Results[i]

		if p0.BranchID == tres.BranchID {
			continue
		}

		la1 := tres.Latitude
		la2 := p0.Latitude
		lo1 := tres.Longitude
		lo2 := p0.Longitude
		dist := km_between_two_points(la1, lo1, la2, lo2)
		if dist > float64(cfg.Geo.CollapseMaximum) {
			continue
		}
		p0.Collapsed = true
		p0.CollapseBranch = tres.BranchID
		ret++
	}
	return ret
}

func geoCollapse(o *object) (err error) {
	for i := range o.Results {
		// If a node has already been collapsed, don't look at it again
		if o.Results[i].Collapsed {
			continue
		}
		o.Results[i].Weight += geoCollapseUsing(o, o.Results[i])
	}
	return nil
}

func geoFindGeocenter(o object) (gc objectGeocenter, err error) {
	var lat, lon_gw, lon_dl float64
	// First pass: calculate two geocenters: one on the greenwich meridian
	// and one of the dateline meridian
	for _, loc := range o.Results {
		lat += (loc.Latitude * loc.Weight)
		lon_gw += (loc.Longitude * loc.Weight)
		lon_dl += (switch_meridians(loc.Longitude) * loc.Weight)
		gc.Weight += loc.Weight
	}
	lat /= gc.Weight
	lon_gw /= gc.Weight
	lon_dl /= gc.Weight
	lon_dl = switch_meridians(lon_dl)

	// Second pass: calculate the distance of each location to the greenwich
	// meridian and the dateline meridian. The average distance that is the
	// shortest indicates which meridian is appropriate to use.
	var dist_to_gw, avg_dist_to_gw, dist_to_dl, avg_dist_to_dl float64
	for _, loc := range o.Results {
		dist_to_gw = km_between_two_points(loc.Latitude, loc.Longitude, lat, lon_gw)
		avg_dist_to_gw += (dist_to_gw * loc.Weight)
		dist_to_dl = km_between_two_points(loc.Latitude, loc.Longitude, lat, lon_dl)
		avg_dist_to_dl += (dist_to_dl * loc.Weight)
	}
	avg_dist_to_gw /= gc.Weight
	avg_dist_to_dl /= gc.Weight
	if avg_dist_to_gw > avg_dist_to_dl {
		// average distance to greenwich meridian is longer than average distance
		// to dateline meridian, so the dateline meridian is our geocenter
		gc.Longitude = lon_dl
		gc.AvgDist = avg_dist_to_dl
	} else {
		gc.Longitude = lon_gw
		gc.AvgDist = avg_dist_to_gw
	}
	gc.Latitude = lat
	return gc, nil
}

// haversin(θ) function
func hsin(theta float64) float64 {
	return math.Pow(math.Sin(theta/2), 2)
}

// Distance function returns the distance (in meters) between two points of
//     a given longitude and latitude relatively accurately (using a spherical
//     approximation of the Earth) through the Haversin Distance Formula for
//     great arc distance on a sphere with accuracy for small distances
//
// point coordinates are supplied in degrees and converted into rad. in the func
//
// distance returned is Kilometers
// http://en.wikipedia.org/wiki/Haversine_formula
func km_between_two_points(lat1, lon1, lat2, lon2 float64) float64 {
	// convert to radians
	// must cast radius as float to multiply later
	var la1, lo1, la2, lo2, r float64
	la1 = lat1 * math.Pi / 180
	lo1 = lon1 * math.Pi / 180
	la2 = lat2 * math.Pi / 180
	lo2 = lon2 * math.Pi / 180

	r = 6378 // Earth radius in Kilometers

	// calculate
	h := hsin(la2-la1) + math.Cos(la1)*math.Cos(la2)*hsin(lo2-lo1)

	return 2 * r * math.Asin(math.Sqrt(h))
}

func switch_meridians(lon float64) float64 {
	if lon < 0.0 {
		return lon + 180.0
	}
	return lon - 180.0
}
