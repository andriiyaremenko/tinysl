# tinysl

[![GoDoc](https://img.shields.io/badge/pkg.go.dev-doc-blue)](http://pkg.go.dev/github.com/andriiyaremenko/tinysl)

This package provides simple abstraction to manage lifetime scope of services.
This package does NOT try to be another IOC container.
It was created because of need to share same instances of services among gorutines
within lifetime of a context.
PerContext lifetime scope was main reason to create this package,
other scopes were created for convenience.

---
Readme created from Go doc with [goreadme](https://github.com/posener/goreadme)
