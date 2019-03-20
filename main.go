package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/godbus/dbus"
	"github.com/sqp/pulseaudio"
)

func main() {
	err := switchSink()
	if err != nil {
		log.Fatalln(err)
	}
}

func switchSink() error {
	sinksFlag := flag.String("sinks", "", "sinks to switch")
	lastOnlyFlag := flag.Bool("last-only", false, "switch default sink but only last stream")
	flag.Parse()

	rawSinks := strings.Split(*sinksFlag, ",")
	var sinks []string
	for _, sink := range rawSinks {
		if strings.TrimSpace(sink) != "" {
			sinks = append(sinks, sink)
		}
	}

	// Load pulseaudio DBus module if needed.
	isLoaded, err := pulseaudio.ModuleIsLoaded()
	if err != nil {
		return fmt.Errorf("failed to check if dbus module is loaded: %v", err)
	}
	if !isLoaded {
		err = pulseaudio.LoadModule()
		if err != nil {
			return fmt.Errorf("failed to load dbus module: %v", err)
		}
	}

	// Connect to the pulseaudio dbus service.
	pulse, err := pulseaudio.New()
	if err != nil {
		return fmt.Errorf("failed to connect to pulse: %v", err)
	}
	defer pulse.Close()

	return doSwitch(pulse, sinks, *lastOnlyFlag)
}

func doSwitch(client *pulseaudio.Client, sinks []string, lastOnly bool) error {
	// find current default sink
	currentSinks, err := client.Core().ListPath("Sinks")
	if err != nil {
		return err
	}

	var currentSink string
	// ignore error if no fallback sink
	currentSinkPath, _ := client.Core().ObjectPath("FallbackSink")
	if currentSinkPath != "" {
		currentSink, err = client.Device(currentSinkPath).String("Name")
		if err != nil {
			return fmt.Errorf("failed to query current sink name: %v", err)
		}
	}

	sinkNameMap := map[string]dbus.ObjectPath{}
	sinkPathMap := map[dbus.ObjectPath]string{}

	// fill sinks automatically if user has not specified any
	fillSinks := len(sinks) == 0

	for _, path := range currentSinks {
		sink := client.Device(path)
		name, err := sink.String("Name")
		if err != nil {
			return fmt.Errorf("failed to query sink name: %v", err)
		}

		sinkNameMap[name] = path
		sinkPathMap[path] = name

		if fillSinks {
			sinks = append(sinks, name)
		}
	}

	// search for current sink in target sink list
	targetSink := ""
	found := false
	for _, name := range sinks {
		if _, ok := sinkNameMap[name]; !ok {
			log.Printf("can't find sink name '%v'", name)
		}

		if currentSink == name {
			found = true
			continue
		}

		// next after target
		if found {
			targetSink = name
			break
		}
	}

	if targetSink == "" {
		targetSink = sinks[0]
	}

	targetPath, ok := sinkNameMap[targetSink]
	if !ok {
		return fmt.Errorf("unknown sink named '%s'", targetSink)
	}

	// set default sink
	err = client.Core().Set("FallbackSink", targetPath)
	if err != nil {
		return fmt.Errorf("failed to set fallback sink: %v", err)
	}

	// switch active streams to target sink
	streams, err := client.Core().ListPath("PlaybackStreams")
	if err != nil {
		return fmt.Errorf("can't list current streams: %v", err)
	}

	if lastOnly {
		streams = streams[len(streams)-1:]
	}

	for _, stream := range streams {
		// Get the device to query properties for the stream referenced by his path.
		dev := client.Stream(stream)
		err = dev.Call("org.PulseAudio.Core1.Stream.Move", 0, targetPath).Err
		if err != nil {
			return fmt.Errorf("failed to switch stream '%s' to target '%v': %v", stream, targetSink, err)
		}
	}

	log.Printf("switched to sink %q", targetSink)

	return nil
}
