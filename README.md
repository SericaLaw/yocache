# Yocache: Your Own groupCache

Borrowed some code from [groupcache](https://github.com/golang/groupcache) and [geecache](https://github.com/geektutu/7days-golang/tree/master/gee-cache), with modifications:

- Removed the protobuf message format and replaced with a `Codec` interface for encoding and decoding internal HTTP messages.
- Supports key expiration for each cache group by setting `ttl`.
- More comments in code, easier to understand.
