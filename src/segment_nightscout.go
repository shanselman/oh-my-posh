package main

import (
	"encoding/json"
)

// segment struct, makes templating easier
type nightscout struct {
	props *properties
	env   environmentInfo

	url          string
	cacheTimeout int

	getFunc func() (*nightscoutData, error)

	// array of nightscoutData (is often just 1, and we will pick the ZEROeth)

	Sgv       int64
	Direction string

	TrendIcon string
}

const (
	// Your complete Nightscout URL and APIKey like this
	URL Property = "url"

	DoubleUpIcon      Property = "doubleup_icon"
	SingleUpIcon      Property = "singleup_icon"
	FortyFiveUpIcon   Property = "fortyfiveup_icon"
	FlatIcon          Property = "flat_icon"
	FortyFiveDownIcon Property = "fortyfivedown_icon"
	SingleDownIcon    Property = "singledown_icon"
	DoubleDownIcon    Property = "doubledown_icon"

	NSCacheTimeout Property = "cache_timeout"
)

type nightscoutData struct {
	Sgv       int64  `json:"sgv"`
	Direction string `json:"direction"`
}

func (ns *nightscout) enabled() bool {
	data, err := ns.getFunc()
	if err != nil {
		return false
	}
	ns.Sgv = data.Sgv
	ns.Direction = data.Direction

	ns.TrendIcon = ns.getTrendIcon()

	return true
}

func (ns *nightscout) getTrendIcon() string {
	switch ns.Direction {
	case "DoubleUp":
		return ns.props.getString(DoubleUpIcon, "↑↑")
	case "SingleUp":
		return ns.props.getString(SingleUpIcon, "↑")
	case "FortyFiveUp":
		return ns.props.getString(FortyFiveUpIcon, "↗")
	case "Flat":
		return ns.props.getString(FlatIcon, "→")
	case "FortyFiveDown":
		return ns.props.getString(FortyFiveDownIcon, "↘")
	case "SingleDown":
		return ns.props.getString(SingleDownIcon, "↓")
	case "DoubleDown":
		return ns.props.getString(DoubleDownIcon, "↓↓")
	default:
		return ""
	}
}

func (ns *nightscout) string() string {
	segmentTemplate := ns.props.getString(SegmentTemplate, "{{.Sgv}}")
	template := &textTemplate{
		Template: segmentTemplate,
		Context:  ns,
		Env:      ns.env,
	}
	text, err := template.render()
	if err != nil {
		return err.Error()
	}

	return text
}

func (ns *nightscout) getData() (*nightscoutData, error) {
	httpTimeout := ns.props.getInt(HTTPTimeout, DefaultHTTPTimeout)

	body, err := ns.env.doGet(ns.url, httpTimeout)
	if err != nil {
		return nil, err
	}
	var arr []*nightscoutData
	err = json.Unmarshal(body, &arr)
	if err != nil {
		return nil, err
	}

	firstelement := arr[0]

	return firstelement, nil
}

func (ns *nightscout) withCache(wrappedFunc func() (*nightscoutData, error)) func() (*nightscoutData, error) {
	return func() (*nightscoutData, error) {
		cacheKey := ns.url

		serializedDataFromCache, foundInCache := ns.env.cache().get(cacheKey)

		if foundInCache {
			dataFromCache := &nightscoutData{}
			err := json.Unmarshal([]byte(serializedDataFromCache), dataFromCache)
			if err == nil {
				return dataFromCache, nil
			}
		}

		data, err := wrappedFunc()
		if err != nil {
			return nil, err
		}

		serializedData, err := json.Marshal(data)
		if err == nil {
			ns.env.cache().set(cacheKey, string(serializedData), ns.cacheTimeout)
		}

		return data, nil
	}
}

func (ns *nightscout) init(props *properties, env environmentInfo) {
	ns.props = props
	ns.env = env

	ns.url = ns.props.getString(URL, "")
	ns.cacheTimeout = ns.props.getInt(NSCacheTimeout, 5)

	ns.getFunc = ns.getData

	if ns.isCacheEnabled() {
		ns.getFunc = ns.withCache(ns.getFunc)
	}
}

func (ns *nightscout) isCacheEnabled() bool {
	return ns.cacheTimeout > 0
}
