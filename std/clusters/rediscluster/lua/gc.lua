local key = KEYS[1]
local instancesKey = KEYS[2]
local id = ARGV[1]
local idleSec = tonumber(ARGV[2])
local timeNow = tonumber(ARGV[3])

if id ~= "" then
    redis.call("HSET", instancesKey, id, timeNow)
end

local ok = redis.call("SETNX", key, "running")
if ok == 1 then
    local members = redis.call("HGETALL", instancesKey)
    for i=1,#members,2 do
        if timeNow - tonumber(members[i+1]) > idleSec then
            redis.call("HDEL", instancesKey, members[i])
        end
    end
end
