Lockheed
========

A really simple auto-lock/ auto-suspend app that's really similar to `xautolock`, but has a bit more built in:
- Disable lock if chromium is playing sound (first cut at checking for html5 video playback).
- Prevent suspend if an OpenVPN connection is established.

```
$ ./lockheed --help
Usage of ./lockheed:
  -locker string
        Command to execute your screen lock. (default "pgrep -f i3lock || i3lock")
  -notifier string
        Command to execute your screen lock. (default "notify-send -u critical -t 10000 -- 'Locking screen soon.'")
  -notify uint
        Seconds before locking when a notification is sent. (default 30)
  -suspend uint
        Minutes of idle time before suspending. (default 15)
  -suspend-disabled
        Don't over suspend.
  -suspend-disabled-while-on-vpn
        Don't engage suspend if we're conected to a VPN. (default true)
  -suspender string
        Command for suspending computer. (default "systemctl suspend")
  -time uint
        Minutes of idle time before locking. (default 5)
```

### Building

```
$ go build lockheed.go
```
