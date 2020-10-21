* [x] honor default tag to populate default value during decoding (optionally?)
* [x] make it easy to plug additional hooks for custom tag names, e.g. `jwt:"username"` to decode Bearer token and unmarshal value
* [x] make decoderFunc GC-friendly // wontdo: does not make sense for net/http, will be done for fasthttp only
* [x] allow GET with body
* [x] customize usecase input with other struct? // very hard!
* [x] customize usecase output with header mapping
* [x] output validation
* [x] move response matters to response package from Handler
* [x] add tests to validate response
* [ ] write README and package docs
* [x] check nethttp.WrapHandler apply order
* [x] allow low level docs configuration
* [x] move jsonschema validator away to a separate module to make it modular
* [x] direct gzip
* [x] skip validation for empty or type-only schema
* [x] add basic example
* [x] http benchmark with bytes rw metric
* [x] refine response encoder and nil-checks around it
* [x] add benchmark test to ensure gzip pass through
* [x] ~~embed gzip writer into standard response encoder?~~
* [x] ~~response encoder by value?~~
* [x] new line at end of response?
* [ ] not rewriting already configured (e.g. with handler option) response encoder?
* [x] custom headers in direct gzip
* [ ] cache-control
* [x] method HEAD