pa-switch-sink - simple tool to switch current pulseaudio sink
==============

This utility sets pulseaudio's default sink and switches all streams to that
sink. Intended to use as bound to hotkey cmd.

Usage:
======

`pa-switch-sink -sinks jack,rtp`

Program will find current default sink and chose next one in list.

`pa-switch-sink -sinks jack,rtp -last-only`

When `-last-only` flag is set only default sink and last active stream will be
switched to another sink.

