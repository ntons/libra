package pubsub

import (
	"github.com/ntons/redis"
)

var luaDelay = redis.NewScript(`
local function add()
	return redis.call("ZADD", KEYS[1], unpack(ARGV))
end

local function try_pop()
	local r = {}
	local t = tonumber(ARGV[1])
	while true do
		local x = redis.call("ZRANGE", KEYS[1], 0, 0, "WITHSCORES")
		if #x < 2 or tonumber(x[2]) > t then break end
		r[#r+1] = redis.call("ZPOPMIN", KEYS[1])[1]
	end
	return r
end

local cmd = ARGV[1]
table.remove(ARGV, 1)
if cmd == "add" then
	return add()
elseif cmd == "try_pop" then
	return try_pop()
end
`)
