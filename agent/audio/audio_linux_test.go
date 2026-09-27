package audio

import (
	"encoding/json"
	"testing"
)

func TestBuildState(t *testing.T) {
	var info paInfo
	var sinks, sources []paDevice
	var inputs []paStream
	must := func(s string, v any) {
		if err := json.Unmarshal([]byte(s), v); err != nil {
			t.Fatal(err)
		}
	}
	must(`{"server_name":"PulseAudio (on PipeWire 1.0.5)","default_sink_name":"headset","default_source_name":"mic"}`, &info)
	must(`[
		{"index":1,"name":"speakers","description":"Enceintes","mute":false,"volume":{"front-left":{"value":32768},"front-right":{"value":32768}},"properties":{}},
		{"index":2,"name":"headset","description":"Casque","mute":true,"volume":{"mono":{"value":98304}},"properties":{}}
	]`, &sinks)
	must(`[
		{"index":3,"name":"headset.monitor","mute":false,"volume":{"mono":{"value":65536}},"properties":{"device.class":"monitor"}},
		{"index":4,"name":"mic","mute":true,"volume":{"mono":{"value":65536}},"properties":{"device.class":"sound"}}
	]`, &sources)
	must(`[
		{"index":10,"mute":true,"volume":{"front-left":{"value":65536}},"properties":{"application.name":"WEBRTC VoiceEngine","application.process.binary":"Discord","application.process.id":"0"}},
		{"index":11,"mute":false,"volume":{"front-left":{"value":16384}},"properties":{"application.name":"WEBRTC VoiceEngine","application.process.binary":"Discord"}},
		{"index":12,"mute":false,"volume":{"front-left":{"value":6554}},"properties":{"application.name":"Firefox"}},
		{"index":13,"mute":false,"volume":{},"properties":{}}
	]`, &inputs)

	s := buildState(info, sinks, sources, inputs)

	if s.Master == nil || *s.Master != (Level{Volume: 100, Muted: true}) {
		t.Errorf("master = %+v, attendu 100 %% (borné) et coupé", s.Master)
	}
	if s.Mic == nil || !s.Mic.Muted {
		t.Errorf("mic = %+v, attendu le micro coupé", s.Mic)
	}
	if len(s.Outputs) != 2 || s.Outputs[0].Name != "Enceintes" || s.Outputs[0].Default || !s.Outputs[1].Default {
		t.Errorf("outputs = %+v", s.Outputs)
	}
	if len(s.Apps) != 2 {
		t.Fatalf("apps = %+v, attendu Discord et Firefox", s.Apps)
	}
	d := s.Apps[0]
	if d.ID != "discord" || d.Name != "Discord" || d.Volume != 100 || d.Muted {
		t.Errorf("discord = %+v, attendu 2 flux regroupés, non coupé", d)
	}
	if f := s.Apps[1]; f.ID != "firefox" || f.Volume != 10 {
		t.Errorf("firefox = %+v", f)
	}
}

func TestNoMicWhenDefaultIsMonitor(t *testing.T) {
	s := buildState(paInfo{DefaultSource: "out.monitor"}, nil,
		[]paDevice{{Name: "out.monitor", Properties: map[string]string{}}}, nil)
	if s.Mic != nil {
		t.Errorf("mic = %+v, attendu aucun micro", s.Mic)
	}
}
