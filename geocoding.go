package mapbox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

const (
	GeocodingBatchEndpoint   = "/search/geocode/v6/batch"
	GeocodingReverseEndpoint = "/search/geocode/v6/reverse"
	GeocodingForwardEndpoint = "/search/geocode/v6/forward"
)

//////////////////////////////////////////////////////////////////

type ReverseGeocodeRequest struct {
	Coordinate

	// optional
	Country  string `json:"country,omitempty"`
	Language string `json:"language,omitempty"`
	Limit    int    `json:"limit,omitempty"`
	Types    Types  `json:"types,omitempty"`
}

type ForwardGeocodeRequest struct {
	SearchText string `json:"q"`

	// optional
	Autocomplete bool        `json:"autocomplete,omitempty"`
	BBox         BoundingBox `json:"bbox,omitempty"`
	Country      string      `json:"country,omitempty"`
	Language     string      `json:"language,omitempty"`
	Limit        int         `json:"limit,omitempty"`
	Proximity    Coordinate  `json:"proximity,omitempty"`
	Types        Types       `json:"types,omitempty"`
}

type GeocodeResponse struct {
	Type        string     `json:"type"`
	Features    []*Feature `json:"features"`
	Attribution string     `json:"attribution"`
}

type GeocodeBatchRequest struct {
	Reverse []ReverseGeocodeRequest
	Forward []ForwardGeocodeRequest
}

type GeocodeBatchResponse struct {
	Batch []GeocodeResponse `json:"batch"`
}

//////////////////////////////////////////////////////////////////

type Feature struct {
	ID         string      `json:"id"`
	Type       string      `json:"type"`
	Geometry   *Geometry   `json:"geometry"`
	Properties *Properties `json:"properties,omitempty"`
}

type Properties struct {
	MapboxID       string             `json:"mapbox_id"`
	FeatureType    Type               `json:"feature_type"`
	Name           string             `json:"name"`
	NamePreferred  string             `json:"name_preferred"`
	PlaceFormatted string             `json:"place_formatted"`
	FullAddress    string             `json:"full_address"`
	Coordinates    ExtendedCoordinate `json:"coordinates"`
	Context        map[Type]Context   `json:"context,omitempty"`
	BoundingBox    []float64          `json:"bbox,omitempty"`
	MatchCode      map[string]string  `json:"match_code,omitempty"`
}

type Context struct {
	MapboxID string `json:"mapbox_id"`
	Name     string `json:"name"`

	// Possible properties from example of standard address
	WikidataID     string `json:"wikidata_id,omitempty"`
	RegionCode     string `json:"region_code,omitempty"`
	RegionCodeFull string `json:"region_code_full,omitempty"`
	AddressNumber  string `json:"address_number,omitempty"`
	StreetName     string `json:"street_name,omitempty"`
}

//////////////////////////////////////////////////////////////////

// https://docs.mapbox.com/api/search/geocoding/#forward-geocoding-with-search-text-input
func forwardGeocode(ctx context.Context, client *Client, req *ForwardGeocodeRequest) (*GeocodeResponse, error) {
	query := url.Values{}
	query.Set("q", req.SearchText)
	query.Set("access_token", client.apiKey)
	query.Set("autocomplete", strconv.FormatBool(req.Autocomplete))

	if !req.BBox.Min.IsZero() {
		query.Set("bbox", req.BBox.query())
	}

	if req.Country != "" {
		query.Set("country", req.Country)
	}

	if req.Language != "" {
		query.Set("language", req.Language)
	}

	if req.Limit != 0 {
		query.Set("limit", strconv.Itoa(req.Limit))
	}

	if !req.Proximity.IsZero() {
		query.Set("proximity", req.Proximity.WGS84Format())
	}

	if len(req.Types) != 0 {
		query.Set("types", req.Types.query())
	}

	apiResponse, err := client.get(ctx, GeocodingForwardEndpoint, query)
	if err != nil {
		return nil, err
	}

	var response GeocodeResponse
	if err := client.handleResponse(apiResponse, &response, GeocodingRateLimit); err != nil {
		return nil, err
	}

	return &response, nil
}

// https://docs.mapbox.com/api/search/geocoding/#reverse-geocoding
func reverseGeocode(ctx context.Context, client *Client, req *ReverseGeocodeRequest) (*GeocodeResponse, error) {
	query := url.Values{}
	query.Set("access_token", client.apiKey)
	query.Set("latitude", strconv.FormatFloat(req.Lat, 'f', -1, 64))
	query.Set("longitude", strconv.FormatFloat(req.Lng, 'f', -1, 64))

	if req.Country != "" {
		query.Set("country", req.Country)
	}

	if req.Language != "" {
		query.Set("language", req.Language)
	}

	if req.Limit > 0 {
		query.Set("limit", fmt.Sprintf("%v", req.Limit))
	}

	if len(req.Types) > 0 {
		query.Set("types", req.Types.query())
	}

	apiResponse, err := client.get(ctx, GeocodingReverseEndpoint, query)
	if err != nil {
		return nil, err
	}

	var response GeocodeResponse
	if err := client.handleResponse(apiResponse, &response, GeocodingRateLimit); err != nil {
		return nil, err
	}

	return &response, nil
}

// https://docs.mapbox.com/api/search/geocoding/#batch-geocoding, but only supports reverse
func reverseGeocodeBatch(ctx context.Context, client *Client, req *GeocodeBatchRequest) (*GeocodeBatchResponse, error) {
	query := url.Values{}
	query.Set("access_token", client.apiKey)

	var reqs []interface{}
	for _, rev := range req.Reverse {
		reqs = append(reqs, rev)
	}

	for _, fwd := range req.Forward {
		reqs = append(reqs, fwd)
	}

	b, err := json.Marshal(reqs)
	if err != nil {
		return nil, err
	}

	apiResponse, err := client.post(ctx, GeocodingBatchEndpoint, query, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	var response *GeocodeBatchResponse
	if err := client.handleResponse(apiResponse, &response, GeocodingRateLimit); err != nil {
		return nil, err
	}

	return response, nil
}
