package redis

// commitLua checks observed versions, then applies writes and outbox entries.
// It returns "conflict" before any write when a version moved, and "ok"
// otherwise. Redis runs the script atomically, so the check and the writes
// cannot interleave with another commit.
//
// KEYS[1] outbox sequence, KEYS[2] pending zset.
// ARGV is groups of: nchecks, (stateKey, version)×n, nwrites,
// (stateKey, kindKey, id, deleted, version, data)×n, noutbox,
// (itemKey, id, partitionKey, data, createdNano)×n.
//
//nolint:dupword // Lua closes every block with end, so end follows end.
const commitLua = `
local seqKey = KEYS[1]
local pending = KEYS[2]
local i = 1
local function take()
  local v = ARGV[i]
  i = i + 1
  return v
end

local nchecks = tonumber(take())
for _ = 1, nchecks do
  local key = take()
  local expect = tonumber(take())
  local cur = redis.call("HGET", key, "v")
  local n = 0
  if cur then n = tonumber(cur) end
  if n ~= expect then return "conflict" end
end

local nwrites = tonumber(take())
for _ = 1, nwrites do
  local key = take()
  local kindKey = take()
  local id = take()
  local deleted = take()
  local ver = take()
  local data = take()
  if deleted == "1" then
    redis.call("DEL", key)
    redis.call("SREM", kindKey, id)
  else
    redis.call("HSET", key, "v", ver, "d", data)
    redis.call("SADD", kindKey, id)
  end
end

local nout = tonumber(take())
for _ = 1, nout do
  local itemKey = take()
  local id = take()
  local pk = take()
  local data = take()
  local created = take()
  local seq = redis.call("INCR", seqKey)
  redis.call("HSET", itemKey, "pk", pk, "d", data, "created", created, "lease", "0")
  redis.call("ZADD", pending, seq, id)
end
return "ok"
`

// claimLua leases up to limit entries, oldest sequence first, skipping a key
// while an older entry with the same non-empty partition key is leased or
// was not returned by this call.
//
// KEYS[1] key prefix, KEYS[2] pending zset.
// ARGV: nowMillis, limit, leaseMillis.
// Returns a flat list of id, partitionKey, data, createdNano.
//
//nolint:dupword // Lua closes every block with end, so end follows end.
const claimLua = `
local prefix = KEYS[1]
local pending = KEYS[2]
local now = tonumber(ARGV[1])
local limit = tonumber(ARGV[2])
local leaseFor = tonumber(ARGV[3])
local ids = redis.call("ZRANGE", pending, 0, -1)
local blocked = {}
local out = {}
for _, id in ipairs(ids) do
  if #out >= limit * 4 then break end
  local item = prefix .. ":o:" .. id
  local pk = redis.call("HGET", item, "pk") or ""
  local lease = tonumber(redis.call("HGET", item, "lease") or "0")
  if lease > now or (pk ~= "" and blocked[pk]) then
    if pk ~= "" then blocked[pk] = true end
  else
    redis.call("HSET", item, "lease", tostring(now + leaseFor))
    local data = redis.call("HGET", item, "d") or ""
    local created = redis.call("HGET", item, "created") or "0"
    table.insert(out, id)
    table.insert(out, pk)
    table.insert(out, data)
    table.insert(out, created)
  end
end
return out
`

// ackLua removes entries. Unknown ids are ignored.
const ackLua = `
local prefix = KEYS[1]
local pending = KEYS[2]
for i = 1, #ARGV do
  local id = ARGV[i]
  redis.call("ZREM", pending, id)
  redis.call("DEL", prefix .. ":o:" .. id)
end
return 1
`

// statsLua returns {pending, oldestCreatedNano}. Pending includes leased
// entries. oldestCreatedNano is 0 when the outbox is empty.
//
//nolint:dupword // Lua closes every block with end, so end follows end.
const statsLua = `
local prefix = KEYS[1]
local pending = KEYS[2]
local ids = redis.call("ZRANGE", pending, 0, -1)
local oldest = nil
for _, id in ipairs(ids) do
  local c = tonumber(redis.call("HGET", prefix .. ":o:" .. id, "created") or "0")
  if c > 0 and (oldest == nil or c < oldest) then oldest = c end
end
if oldest == nil then return {#ids, 0} end
return {#ids, oldest}
`
