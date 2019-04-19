<h1 align="center">Joe Bot - Cron Module</h1>
<p align="center">Emiting events on recurring schedules. https://github.com/go-joe/joe</p>
<p align="center">
	<a href="https://github.com/go-joe/cron/releases"><img src="https://img.shields.io/github/tag/go-joe/cron.svg?label=version&color=brightgreen"></a>
	<a href="https://circleci.com/gh/go-joe/cron/tree/master"><img src="https://circleci.com/gh/go-joe/cron/tree/master.svg?style=shield"></a>
	<a href="https://goreportcard.com/report/github.com/go-joe/cron"><img src="https://goreportcard.com/badge/github.com/go-joe/cron"></a>
	<a href="https://codecov.io/gh/go-joe/cron"><img src="https://codecov.io/gh/go-joe/cron/branch/master/graph/badge.svg"/></a>
	<a href="https://godoc.org/github.com/go-joe/cron"><img src="https://img.shields.io/badge/godoc-reference-blue.svg?color=blue"></a>
	<a href="https://github.com/go-joe/cron/blob/master/LICENSE"><img src="https://img.shields.io/badge/license-BSD--3--Clause-blue.svg"></a>
</p>

---

This repository contains a module for the [Joe Bot library][joe].

**THIS SOFTWARE IS STILL IN ALPHA AND THERE ARE NO GUARANTEES REGARDING API STABILITY YET.**

## Getting Started

This library is packaged using the new [Go modules][go-modules]. You can get it via:

```bash
go get github.com/go-joe/cron
```

### Example usage

This module allows you to run arbitrary functions or emit events on a schedule
using a [cron expressions][cron] or simply an interval expressed as `time.Duration`.

```go
package main

import (
	"time"
	"github.com/go-joe/joe"
	"github.com/go-joe/cron"
)

type MyEvent struct {}

func main() {
	b := joe.New("example-bot",
		// emit a cron.Event once every day at midnight
		cron.ScheduleEvent("0 0 * * *"),
		
		// emit your own custom event every day at 09:00
		cron.ScheduleEvent("0 9 * * *", MyEvent{}), 
		
		// cron expressions can be hard to read and might be overkill
		cron.ScheduleEventEvery(time.Hour, MyEvent{}), 
		
		// sometimes its easier to use a function
		cron.ScheduleFunc("0 9 * * *", func() { /* TODO */ }), 
		
		// functions can also be scheduled on simple intervals
		cron.ScheduleFuncEvery(5*time.Minute, func() { /* TODO */ }),
    )
	
	err := b.Run()
	if err != nil {
		b.Logger.Fatal(err.Error())
	}
}
```

## Built With

* [robfig/cron](https://github.com/robfig/cron) - A cron library for go
* [zap](https://github.com/uber-go/zap) - Blazing fast, structured, leveled logging in Go
* [pkg/errors](https://github.com/pkg/errors) - Simple error handling primitives
* [testify](https://github.com/stretchr/testify) - A simple unit test library

## Contributing

If you want to hack on this repository, please read the short [CONTRIBUTING.md](CONTRIBUTING.md)
guide first.

## Versioning

We use [SemVer](http://semver.org/) for versioning. For the versions available,
see the [tags on this repository][tags. 

## Authors

- **Friedrich Große** - *Initial work* - [fgrosse](https://github.com/fgrosse)

See also the list of [contributors][contributors] who participated in this project.

## License

This project is licensed under the BSD-3-Clause License - see the [LICENSE](LICENSE) file for details.

[joe]: https://github.com/go-joe/joe
[go-modules]: https://github.com/golang/go/wiki/Modules
[tags]: https://github.com/go-joe/cron/tags
[contributors]: https://github.com/github.com/go-joe/cron/contributors
[cron]: https://en.wikipedia.org/wiki/Cron#Overview
