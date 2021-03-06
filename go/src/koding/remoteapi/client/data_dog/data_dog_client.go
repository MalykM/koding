package data_dog

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"github.com/go-openapi/runtime"

	strfmt "github.com/go-openapi/strfmt"
)

// New creates a new data dog API client.
func New(transport runtime.ClientTransport, formats strfmt.Registry) *Client {
	return &Client{transport: transport, formats: formats}
}

/*
Client for data dog API
*/
type Client struct {
	transport runtime.ClientTransport
	formats   strfmt.Registry
}

/*
DataDogSendEvent data dog send event API
*/
func (a *Client) DataDogSendEvent(params *DataDogSendEventParams, authInfo runtime.ClientAuthInfoWriter) (*DataDogSendEventOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewDataDogSendEventParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "DataDog.sendEvent",
		Method:             "POST",
		PathPattern:        "/remote.api/DataDog.sendEvent",
		ProducesMediaTypes: []string{""},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &DataDogSendEventReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return result.(*DataDogSendEventOK), nil

}

/*
DataDogSendMetrics data dog send metrics API
*/
func (a *Client) DataDogSendMetrics(params *DataDogSendMetricsParams, authInfo runtime.ClientAuthInfoWriter) (*DataDogSendMetricsOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewDataDogSendMetricsParams()
	}

	result, err := a.transport.Submit(&runtime.ClientOperation{
		ID:                 "DataDog.sendMetrics",
		Method:             "POST",
		PathPattern:        "/remote.api/DataDog.sendMetrics",
		ProducesMediaTypes: []string{""},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &DataDogSendMetricsReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return result.(*DataDogSendMetricsOK), nil

}

// SetTransport changes the transport on the client
func (a *Client) SetTransport(transport runtime.ClientTransport) {
	a.transport = transport
}
